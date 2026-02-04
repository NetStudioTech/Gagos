// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
)

func InitClient() error {
	var err error

	// Try in-cluster config first
	restConfig, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create k8s config: %w", err)
		}
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	return nil
}

func GetClient() *kubernetes.Clientset {
	return clientset
}

// GetConfig returns the rest.Config for creating additional clients (e.g., metrics)
func GetConfig() *rest.Config {
	return restConfig
}

type NamespaceInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	Age       string            `json:"age"`
}

func ListNamespaces(ctx context.Context) ([]NamespaceInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []NamespaceInfo
	for _, ns := range namespaces.Items {
		result = append(result, NamespaceInfo{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			Labels:    ns.Labels,
			CreatedAt: ns.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(ns.CreationTimestamp.Time),
		})
	}

	return result, nil
}

type PodInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Ready      string            `json:"ready"`
	Restarts   int32             `json:"restarts"`
	Node       string            `json:"node"`
	IP         string            `json:"ip"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Age        string            `json:"age"`
	Containers []ContainerInfo   `json:"containers"`
}

type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restart_count"`
	State        string `json:"state"`
}

func ListPods(ctx context.Context, namespace string) ([]PodInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []PodInfo
	for _, pod := range pods.Items {
		var containers []ContainerInfo
		var totalRestarts int32
		readyCount := 0

		for _, cs := range pod.Status.ContainerStatuses {
			state := "unknown"
			if cs.State.Running != nil {
				state = "running"
			} else if cs.State.Waiting != nil {
				state = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				state = cs.State.Terminated.Reason
			}

			containers = append(containers, ContainerInfo{
				Name:         cs.Name,
				Image:        cs.Image,
				Ready:        cs.Ready,
				RestartCount: cs.RestartCount,
				State:        state,
			})
			totalRestarts += cs.RestartCount
			if cs.Ready {
				readyCount++
			}
		}

		result = append(result, PodInfo{
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			Status:     string(pod.Status.Phase),
			Ready:      fmt.Sprintf("%d/%d", readyCount, len(pod.Spec.Containers)),
			Restarts:   totalRestarts,
			Node:       pod.Spec.NodeName,
			IP:         pod.Status.PodIP,
			Labels:     pod.Labels,
			CreatedAt:  pod.CreationTimestamp.Format(time.RFC3339),
			Age:        formatAge(pod.CreationTimestamp.Time),
			Containers: containers,
		})
	}

	return result, nil
}

type ServiceInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"cluster_ip"`
	ExternalIP string            `json:"external_ip,omitempty"`
	Ports      []ServicePort     `json:"ports"`
	Labels     map[string]string `json:"labels,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Age        string            `json:"age"`
}

type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
	Protocol   string `json:"protocol"`
}

func ListServices(ctx context.Context, namespace string) ([]ServiceInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []ServiceInfo
	for _, svc := range services.Items {
		var ports []ServicePort
		for _, p := range svc.Spec.Ports {
			ports = append(ports, ServicePort{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				NodePort:   p.NodePort,
				Protocol:   string(p.Protocol),
			})
		}

		externalIP := ""
		if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = svc.Spec.ExternalIPs[0]
		} else if svc.Spec.Type == "LoadBalancer" && len(svc.Status.LoadBalancer.Ingress) > 0 {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
		}

		result = append(result, ServiceInfo{
			Name:       svc.Name,
			Namespace:  svc.Namespace,
			Type:       string(svc.Spec.Type),
			ClusterIP:  svc.Spec.ClusterIP,
			ExternalIP: externalIP,
			Ports:      ports,
			Labels:     svc.Labels,
			Selector:   svc.Spec.Selector,
			CreatedAt:  svc.CreationTimestamp.Format(time.RFC3339),
			Age:        formatAge(svc.CreationTimestamp.Time),
		})
	}

	return result, nil
}

type DeploymentInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Ready      string            `json:"ready"`
	UpToDate   int32             `json:"up_to_date"`
	Available  int32             `json:"available"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Age        string            `json:"age"`
}

func ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []DeploymentInfo
	for _, dep := range deployments.Items {
		result = append(result, DeploymentInfo{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, *dep.Spec.Replicas),
			UpToDate:  dep.Status.UpdatedReplicas,
			Available: dep.Status.AvailableReplicas,
			Labels:    dep.Labels,
			CreatedAt: dep.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(dep.CreationTimestamp.Time),
		})
	}

	return result, nil
}

type NodeInfo struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Roles            []string          `json:"roles"`
	InternalIP       string            `json:"internal_ip"`
	ExternalIP       string            `json:"external_ip,omitempty"`
	OS               string            `json:"os"`
	KernelVersion    string            `json:"kernel_version"`
	ContainerRuntime string            `json:"container_runtime"`
	KubeletVersion   string            `json:"kubelet_version"`
	Labels           map[string]string `json:"labels,omitempty"`
	CreatedAt        string            `json:"created_at"`
	Age              string            `json:"age"`
}

func ListNodes(ctx context.Context) ([]NodeInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []NodeInfo
	for _, node := range nodes.Items {
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}

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

		internalIP := ""
		externalIP := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				internalIP = addr.Address
			}
			if addr.Type == "ExternalIP" {
				externalIP = addr.Address
			}
		}

		result = append(result, NodeInfo{
			Name:             node.Name,
			Status:           status,
			Roles:            roles,
			InternalIP:       internalIP,
			ExternalIP:       externalIP,
			OS:               node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			Labels:           node.Labels,
			CreatedAt:        node.CreationTimestamp.Format(time.RFC3339),
			Age:              formatAge(node.CreationTimestamp.Time),
		})
	}

	return result, nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d.Hours() >= 24*365 {
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
	if d.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// Additional Resource Types

type ConfigMapInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      int               `json:"data_count"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	Age       string            `json:"age"`
}

func ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMapInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	cms, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []ConfigMapInfo
	for _, cm := range cms.Items {
		result = append(result, ConfigMapInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Data:      len(cm.Data),
			Labels:    cm.Labels,
			CreatedAt: cm.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(cm.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type SecretInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	Data      int               `json:"data_count"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	Age       string            `json:"age"`
}

func ListSecrets(ctx context.Context, namespace string) ([]SecretInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []SecretInfo
	for _, s := range secrets.Items {
		result = append(result, SecretInfo{
			Name:      s.Name,
			Namespace: s.Namespace,
			Type:      string(s.Type),
			Data:      len(s.Data),
			Labels:    s.Labels,
			CreatedAt: s.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(s.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type ServiceAccountInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Secrets   int               `json:"secrets"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	Age       string            `json:"age"`
}

func ListServiceAccounts(ctx context.Context, namespace string) ([]ServiceAccountInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	sas, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []ServiceAccountInfo
	for _, sa := range sas.Items {
		result = append(result, ServiceAccountInfo{
			Name:      sa.Name,
			Namespace: sa.Namespace,
			Secrets:   len(sa.Secrets),
			Labels:    sa.Labels,
			CreatedAt: sa.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(sa.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type PVInfo struct {
	Name            string            `json:"name"`
	Capacity        string            `json:"capacity"`
	AccessModes     []string          `json:"access_modes"`
	ReclaimPolicy   string            `json:"reclaim_policy"`
	Status          string            `json:"status"`
	Claim           string            `json:"claim"`
	StorageClass    string            `json:"storage_class"`
	Labels          map[string]string `json:"labels,omitempty"`
	CreatedAt       string            `json:"created_at"`
	Age             string            `json:"age"`
}

func ListPersistentVolumes(ctx context.Context) ([]PVInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pvs, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []PVInfo
	for _, pv := range pvs.Items {
		var accessModes []string
		for _, am := range pv.Spec.AccessModes {
			accessModes = append(accessModes, string(am))
		}

		claim := ""
		if pv.Spec.ClaimRef != nil {
			claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}

		capacity := ""
		if qty, ok := pv.Spec.Capacity["storage"]; ok {
			capacity = qty.String()
		}

		result = append(result, PVInfo{
			Name:          pv.Name,
			Capacity:      capacity,
			AccessModes:   accessModes,
			ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
			Status:        string(pv.Status.Phase),
			Claim:         claim,
			StorageClass:  pv.Spec.StorageClassName,
			Labels:        pv.Labels,
			CreatedAt:     pv.CreationTimestamp.Format(time.RFC3339),
			Age:           formatAge(pv.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type PVCInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Status       string            `json:"status"`
	Volume       string            `json:"volume"`
	Capacity     string            `json:"capacity"`
	AccessModes  []string          `json:"access_modes"`
	StorageClass string            `json:"storage_class"`
	Labels       map[string]string `json:"labels,omitempty"`
	CreatedAt    string            `json:"created_at"`
	Age          string            `json:"age"`
}

func ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]PVCInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []PVCInfo
	for _, pvc := range pvcs.Items {
		var accessModes []string
		for _, am := range pvc.Spec.AccessModes {
			accessModes = append(accessModes, string(am))
		}

		capacity := ""
		if qty, ok := pvc.Status.Capacity["storage"]; ok {
			capacity = qty.String()
		}

		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}

		result = append(result, PVCInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Volume:       pvc.Spec.VolumeName,
			Capacity:     capacity,
			AccessModes:  accessModes,
			StorageClass: storageClass,
			Labels:       pvc.Labels,
			CreatedAt:    pvc.CreationTimestamp.Format(time.RFC3339),
			Age:          formatAge(pvc.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type IngressInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Class      string            `json:"class"`
	Hosts      []string          `json:"hosts"`
	Address    string            `json:"address"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Age        string            `json:"age"`
}

func ListIngresses(ctx context.Context, namespace string) ([]IngressInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []IngressInfo
	for _, ing := range ingresses.Items {
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}

		class := ""
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}

		address := ""
		if len(ing.Status.LoadBalancer.Ingress) > 0 {
			if ing.Status.LoadBalancer.Ingress[0].IP != "" {
				address = ing.Status.LoadBalancer.Ingress[0].IP
			} else {
				address = ing.Status.LoadBalancer.Ingress[0].Hostname
			}
		}

		result = append(result, IngressInfo{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     class,
			Hosts:     hosts,
			Address:   address,
			Labels:    ing.Labels,
			CreatedAt: ing.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(ing.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type DaemonSetInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Desired         int32             `json:"desired"`
	Current         int32             `json:"current"`
	Ready           int32             `json:"ready"`
	UpToDate        int32             `json:"up_to_date"`
	Available       int32             `json:"available"`
	NodeSelector    string            `json:"node_selector"`
	Labels          map[string]string `json:"labels,omitempty"`
	CreatedAt       string            `json:"created_at"`
	Age             string            `json:"age"`
}

func ListDaemonSets(ctx context.Context, namespace string) ([]DaemonSetInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	dss, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []DaemonSetInfo
	for _, ds := range dss.Items {
		nodeSelector := ""
		if len(ds.Spec.Template.Spec.NodeSelector) > 0 {
			for k, v := range ds.Spec.Template.Spec.NodeSelector {
				nodeSelector += k + "=" + v + " "
			}
		}

		result = append(result, DaemonSetInfo{
			Name:         ds.Name,
			Namespace:    ds.Namespace,
			Desired:      ds.Status.DesiredNumberScheduled,
			Current:      ds.Status.CurrentNumberScheduled,
			Ready:        ds.Status.NumberReady,
			UpToDate:     ds.Status.UpdatedNumberScheduled,
			Available:    ds.Status.NumberAvailable,
			NodeSelector: nodeSelector,
			Labels:       ds.Labels,
			CreatedAt:    ds.CreationTimestamp.Format(time.RFC3339),
			Age:          formatAge(ds.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type StatefulSetInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Ready      string            `json:"ready"`
	Replicas   int32             `json:"replicas"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Age        string            `json:"age"`
}

func ListStatefulSets(ctx context.Context, namespace string) ([]StatefulSetInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	sss, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []StatefulSetInfo
	for _, ss := range sss.Items {
		replicas := int32(0)
		if ss.Spec.Replicas != nil {
			replicas = *ss.Spec.Replicas
		}
		result = append(result, StatefulSetInfo{
			Name:      ss.Name,
			Namespace: ss.Namespace,
			Ready:     fmt.Sprintf("%d/%d", ss.Status.ReadyReplicas, replicas),
			Replicas:  replicas,
			Labels:    ss.Labels,
			CreatedAt: ss.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(ss.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type JobInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Completions  string            `json:"completions"`
	Duration     string            `json:"duration"`
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels,omitempty"`
	CreatedAt    string            `json:"created_at"`
	Age          string            `json:"age"`
}

func ListJobs(ctx context.Context, namespace string) ([]JobInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []JobInfo
	for _, job := range jobs.Items {
		completions := int32(1)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}

		status := "Running"
		if job.Status.Succeeded > 0 && job.Status.Succeeded >= completions {
			status = "Complete"
		} else if job.Status.Failed > 0 {
			status = "Failed"
		}

		duration := "-"
		if job.Status.StartTime != nil && job.Status.CompletionTime != nil {
			d := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
			duration = d.Round(time.Second).String()
		}

		result = append(result, JobInfo{
			Name:        job.Name,
			Namespace:   job.Namespace,
			Completions: fmt.Sprintf("%d/%d", job.Status.Succeeded, completions),
			Duration:    duration,
			Status:      status,
			Labels:      job.Labels,
			CreatedAt:   job.CreationTimestamp.Format(time.RFC3339),
			Age:         formatAge(job.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type CronJobInfo struct {
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	Schedule      string            `json:"schedule"`
	Suspend       bool              `json:"suspend"`
	Active        int               `json:"active"`
	LastSchedule  string            `json:"last_schedule"`
	Labels        map[string]string `json:"labels,omitempty"`
	CreatedAt     string            `json:"created_at"`
	Age           string            `json:"age"`
}

func ListCronJobs(ctx context.Context, namespace string) ([]CronJobInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	cjs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []CronJobInfo
	for _, cj := range cjs.Items {
		lastSchedule := "-"
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = formatAge(cj.Status.LastScheduleTime.Time) + " ago"
		}

		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}

		result = append(result, CronJobInfo{
			Name:         cj.Name,
			Namespace:    cj.Namespace,
			Schedule:     cj.Spec.Schedule,
			Suspend:      suspend,
			Active:       len(cj.Status.Active),
			LastSchedule: lastSchedule,
			Labels:       cj.Labels,
			CreatedAt:    cj.CreationTimestamp.Format(time.RFC3339),
			Age:          formatAge(cj.CreationTimestamp.Time),
		})
	}
	return result, nil
}

type EventInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Object    string `json:"object"`
	Message   string `json:"message"`
	Count     int32  `json:"count"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
	Age       string `json:"age"`
}

func ListEvents(ctx context.Context, namespace string) ([]EventInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []EventInfo
	for _, e := range events.Items {
		result = append(result, EventInfo{
			Name:      e.Name,
			Namespace: e.Namespace,
			Type:      e.Type,
			Reason:    e.Reason,
			Object:    e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
			Message:   e.Message,
			Count:     e.Count,
			FirstSeen: e.FirstTimestamp.Format(time.RFC3339),
			LastSeen:  e.LastTimestamp.Format(time.RFC3339),
			Age:       formatAge(e.LastTimestamp.Time),
		})
	}
	return result, nil
}

type ReplicaSetInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Desired   int32             `json:"desired"`
	Current   int32             `json:"current"`
	Ready     int32             `json:"ready"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
	Age       string            `json:"age"`
}

func ListReplicaSets(ctx context.Context, namespace string) ([]ReplicaSetInfo, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	rss, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []ReplicaSetInfo
	for _, rs := range rss.Items {
		desired := int32(0)
		if rs.Spec.Replicas != nil {
			desired = *rs.Spec.Replicas
		}
		result = append(result, ReplicaSetInfo{
			Name:      rs.Name,
			Namespace: rs.Namespace,
			Desired:   desired,
			Current:   rs.Status.Replicas,
			Ready:     rs.Status.ReadyReplicas,
			Labels:    rs.Labels,
			CreatedAt: rs.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(rs.CreationTimestamp.Time),
		})
	}
	return result, nil
}
