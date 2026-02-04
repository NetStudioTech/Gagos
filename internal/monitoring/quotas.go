package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListResourceQuotas retrieves resource quotas for a namespace
func ListResourceQuotas(ctx context.Context, namespace string) ([]ResourceQuotaInfo, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	quotas, err := k8sClient.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list resource quotas: %w", err)
	}

	var result []ResourceQuotaInfo
	for _, quota := range quotas.Items {
		hard := make(map[string]string)
		used := make(map[string]string)
		var usage []ResourceQuotaUsage

		for resource, qty := range quota.Status.Hard {
			hard[string(resource)] = qty.String()
		}
		for resource, qty := range quota.Status.Used {
			used[string(resource)] = qty.String()
		}

		// Calculate usage percentages
		for resource, hardQty := range quota.Status.Hard {
			usedQty := quota.Status.Used[resource]
			hardVal := hardQty.Value()
			usedVal := usedQty.Value()

			percent := float64(0)
			if hardVal > 0 {
				percent = float64(usedVal) / float64(hardVal) * 100
			}

			status := "ok"
			if percent >= 90 {
				status = "critical"
			} else if percent >= 75 {
				status = "warning"
			}

			usage = append(usage, ResourceQuotaUsage{
				Resource: string(resource),
				Hard:     hardQty.String(),
				Used:     usedQty.String(),
				Percent:  percent,
				Status:   status,
			})
		}

		result = append(result, ResourceQuotaInfo{
			Name:      quota.Name,
			Namespace: quota.Namespace,
			Hard:      hard,
			Used:      used,
			Usage:     usage,
			CreatedAt: quota.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(quota.CreationTimestamp.Time),
		})
	}

	return result, nil
}

// ListLimitRanges retrieves limit ranges for a namespace
func ListLimitRanges(ctx context.Context, namespace string) ([]LimitRangeInfo, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	limitRanges, err := k8sClient.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list limit ranges: %w", err)
	}

	var result []LimitRangeInfo
	for _, lr := range limitRanges.Items {
		var limits []LimitRangeItem

		for _, limit := range lr.Spec.Limits {
			item := LimitRangeItem{
				Type: string(limit.Type),
			}

			if len(limit.Max) > 0 {
				item.Max = make(map[string]string)
				for r, q := range limit.Max {
					item.Max[string(r)] = q.String()
				}
			}
			if len(limit.Min) > 0 {
				item.Min = make(map[string]string)
				for r, q := range limit.Min {
					item.Min[string(r)] = q.String()
				}
			}
			if len(limit.Default) > 0 {
				item.Default = make(map[string]string)
				for r, q := range limit.Default {
					item.Default[string(r)] = q.String()
				}
			}
			if len(limit.DefaultRequest) > 0 {
				item.DefaultRequest = make(map[string]string)
				for r, q := range limit.DefaultRequest {
					item.DefaultRequest[string(r)] = q.String()
				}
			}

			limits = append(limits, item)
		}

		result = append(result, LimitRangeInfo{
			Name:      lr.Name,
			Namespace: lr.Namespace,
			Limits:    limits,
			CreatedAt: lr.CreationTimestamp.Format(time.RFC3339),
			Age:       formatAge(lr.CreationTimestamp.Time),
		})
	}

	return result, nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d.Hours() >= 24*365 {
		return strconv.Itoa(int(d.Hours()/(24*365))) + "y"
	}
	if d.Hours() >= 24 {
		return strconv.Itoa(int(d.Hours()/24)) + "d"
	}
	if d.Hours() >= 1 {
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	if d.Minutes() >= 1 {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	return strconv.Itoa(int(d.Seconds())) + "s"
}
