package monitoring

import (
	"context"
	"fmt"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListHPAs retrieves HorizontalPodAutoscalers for a namespace
func ListHPAs(ctx context.Context, namespace string) ([]HPAInfo, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	hpas, err := k8sClient.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list HPAs: %w", err)
	}

	var result []HPAInfo
	for _, hpa := range hpas.Items {
		info := HPAInfo{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			TargetKind:      hpa.Spec.ScaleTargetRef.Kind,
			TargetName:      hpa.Spec.ScaleTargetRef.Name,
			MinReplicas:     getMinReplicas(hpa.Spec.MinReplicas),
			MaxReplicas:     hpa.Spec.MaxReplicas,
			CurrentReplicas: hpa.Status.CurrentReplicas,
			DesiredReplicas: hpa.Status.DesiredReplicas,
			CreatedAt:       hpa.CreationTimestamp.Format(time.RFC3339),
			Age:             formatAge(hpa.CreationTimestamp.Time),
		}

		// Extract CPU/Memory targets and current values
		for _, metric := range hpa.Spec.Metrics {
			if metric.Type == autoscalingv2.ResourceMetricSourceType {
				if metric.Resource.Name == "cpu" && metric.Resource.Target.AverageUtilization != nil {
					target := *metric.Resource.Target.AverageUtilization
					info.TargetCPU = &target
				}
				if metric.Resource.Name == "memory" && metric.Resource.Target.AverageUtilization != nil {
					target := *metric.Resource.Target.AverageUtilization
					info.TargetMemory = &target
				}
			}
		}

		// Get current metric values
		for _, metric := range hpa.Status.CurrentMetrics {
			if metric.Type == autoscalingv2.ResourceMetricSourceType {
				if metric.Resource.Name == "cpu" && metric.Resource.Current.AverageUtilization != nil {
					current := *metric.Resource.Current.AverageUtilization
					info.CurrentCPU = &current
				}
				if metric.Resource.Name == "memory" && metric.Resource.Current.AverageUtilization != nil {
					current := *metric.Resource.Current.AverageUtilization
					info.CurrentMemory = &current
				}
			}
		}

		// Extract conditions
		var conditions []string
		for _, cond := range hpa.Status.Conditions {
			if cond.Status == "True" {
				conditions = append(conditions, string(cond.Type))
			}
		}
		info.Conditions = conditions

		// Last scale time
		if hpa.Status.LastScaleTime != nil {
			info.LastScaleTime = hpa.Status.LastScaleTime.Format(time.RFC3339)
		}

		result = append(result, info)
	}

	return result, nil
}

func getMinReplicas(minReplicas *int32) int32 {
	if minReplicas == nil {
		return 1
	}
	return *minReplicas
}
