package monitoring

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodeMetrics retrieves resource metrics for all nodes
func GetNodeMetrics(ctx context.Context) ([]NodeMetrics, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	// Get node list for capacity info
	nodes, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get pods to count per node
	pods, err := k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Count pods per node
	podCountByNode := make(map[string]int)
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" && pod.Status.Phase == corev1.PodRunning {
			podCountByNode[pod.Spec.NodeName]++
		}
	}

	// Try to get metrics from metrics-server
	var nodeMetricsMap map[string]struct {
		CPUUsage    int64
		MemoryUsage int64
	}

	if metricsClient != nil {
		nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err == nil {
			nodeMetricsMap = make(map[string]struct {
				CPUUsage    int64
				MemoryUsage int64
			})
			for _, nm := range nodeMetricsList.Items {
				nodeMetricsMap[nm.Name] = struct {
					CPUUsage    int64
					MemoryUsage int64
				}{
					CPUUsage:    nm.Usage.Cpu().MilliValue(),
					MemoryUsage: nm.Usage.Memory().Value(),
				}
			}
		}
	}

	var result []NodeMetrics
	for _, node := range nodes.Items {
		// Get conditions
		var conditions []string
		for _, cond := range node.Status.Conditions {
			if cond.Status == corev1.ConditionTrue {
				conditions = append(conditions, string(cond.Type))
			}
		}

		// Get roles
		var roles []string
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				roles = append(roles, "control-plane")
			}
			if label == "node-role.kubernetes.io/worker" {
				roles = append(roles, "worker")
			}
		}
		if len(roles) == 0 {
			roles = []string{"worker"}
		}

		// Get capacity
		cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
		memCapacity := node.Status.Capacity.Memory().Value()
		podCapacity := int(node.Status.Capacity.Pods().Value())

		// Get usage (from metrics or estimate)
		var cpuUsage, memUsage int64
		if nodeMetricsMap != nil {
			if m, ok := nodeMetricsMap[node.Name]; ok {
				cpuUsage = m.CPUUsage
				memUsage = m.MemoryUsage
			}
		}

		// Calculate percentages
		cpuPercent := float64(0)
		if cpuCapacity > 0 {
			cpuPercent = float64(cpuUsage) / float64(cpuCapacity) * 100
		}
		memPercent := float64(0)
		if memCapacity > 0 {
			memPercent = float64(memUsage) / float64(memCapacity) * 100
		}

		result = append(result, NodeMetrics{
			Name:           node.Name,
			CPUUsage:       cpuUsage,
			CPUCapacity:    cpuCapacity,
			CPUPercent:     cpuPercent,
			MemoryUsage:    memUsage,
			MemoryCapacity: memCapacity,
			MemoryPercent:  memPercent,
			PodCount:       podCountByNode[node.Name],
			PodCapacity:    podCapacity,
			Conditions:     conditions,
			Roles:          roles,
		})
	}

	return result, nil
}

// GetPodMetrics retrieves resource metrics for pods
func GetPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	// Get pod list
	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Try to get metrics from metrics-server
	var podMetricsMap map[string]map[string]struct {
		CPUUsage    int64
		MemoryUsage int64
	}

	if metricsClient != nil {
		podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			podMetricsMap = make(map[string]map[string]struct {
				CPUUsage    int64
				MemoryUsage int64
			})
			for _, pm := range podMetricsList.Items {
				key := pm.Namespace + "/" + pm.Name
				podMetricsMap[key] = make(map[string]struct {
					CPUUsage    int64
					MemoryUsage int64
				})
				for _, container := range pm.Containers {
					podMetricsMap[key][container.Name] = struct {
						CPUUsage    int64
						MemoryUsage int64
					}{
						CPUUsage:    container.Usage.Cpu().MilliValue(),
						MemoryUsage: container.Usage.Memory().Value(),
					}
				}
			}
		}
	}

	var result []PodMetrics
	for _, pod := range pods.Items {
		// Skip non-running pods
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		key := pod.Namespace + "/" + pod.Name
		var totalCPU, totalMem int64
		var containers []ContainerMetrics

		for _, c := range pod.Spec.Containers {
			var cpuUsage, memUsage int64
			if podMetricsMap != nil {
				if podMetrics, ok := podMetricsMap[key]; ok {
					if cm, ok := podMetrics[c.Name]; ok {
						cpuUsage = cm.CPUUsage
						memUsage = cm.MemoryUsage
					}
				}
			}
			totalCPU += cpuUsage
			totalMem += memUsage

			containers = append(containers, ContainerMetrics{
				Name:        c.Name,
				CPUUsage:    cpuUsage,
				MemoryUsage: memUsage,
			})
		}

		result = append(result, PodMetrics{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			CPUUsage:    totalCPU,
			MemoryUsage: totalMem,
			Containers:  containers,
			Node:        pod.Spec.NodeName,
			Status:      string(pod.Status.Phase),
		})
	}

	return result, nil
}

// GetClusterSummary returns aggregated cluster metrics
func GetClusterSummary(ctx context.Context) (*ClusterSummary, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	// Get nodes
	nodes, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get pods
	pods, err := k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Count nodes
	totalNodes := len(nodes.Items)
	readyNodes := 0
	var totalCPU, totalMem int64

	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				readyNodes++
				break
			}
		}
		totalCPU += node.Status.Capacity.Cpu().MilliValue()
		totalMem += node.Status.Capacity.Memory().Value()
	}

	// Count pods
	totalPods := len(pods.Items)
	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods++
		}
	}

	// Get used CPU/Memory from metrics
	var usedCPU, usedMem int64
	if metricsClient != nil {
		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, nm := range nodeMetrics.Items {
				usedCPU += nm.Usage.Cpu().MilliValue()
				usedMem += nm.Usage.Memory().Value()
			}
		}
	}

	// Calculate percentages
	cpuPercent := float64(0)
	if totalCPU > 0 {
		cpuPercent = float64(usedCPU) / float64(totalCPU) * 100
	}
	memPercent := float64(0)
	if totalMem > 0 {
		memPercent = float64(usedMem) / float64(totalMem) * 100
	}

	return &ClusterSummary{
		TotalNodes:         totalNodes,
		ReadyNodes:         readyNodes,
		TotalCPUMillicores: totalCPU,
		UsedCPUMillicores:  usedCPU,
		CPUPercent:         cpuPercent,
		TotalMemoryBytes:   totalMem,
		UsedMemoryBytes:    usedMem,
		MemoryPercent:      memPercent,
		TotalPods:          totalPods,
		RunningPods:        runningPods,
		Timestamp:          time.Now(),
	}, nil
}
