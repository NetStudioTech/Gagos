package monitoring

import "time"

// NodeMetrics represents resource metrics for a single node
type NodeMetrics struct {
	Name           string   `json:"name"`
	CPUUsage       int64    `json:"cpu_usage_millicores"`
	CPUCapacity    int64    `json:"cpu_capacity_millicores"`
	CPUPercent     float64  `json:"cpu_percent"`
	MemoryUsage    int64    `json:"memory_usage_bytes"`
	MemoryCapacity int64    `json:"memory_capacity_bytes"`
	MemoryPercent  float64  `json:"memory_percent"`
	PodCount       int      `json:"pod_count"`
	PodCapacity    int      `json:"pod_capacity"`
	Conditions     []string `json:"conditions"`
	Roles          []string `json:"roles,omitempty"`
}

// PodMetrics represents resource metrics for a single pod
type PodMetrics struct {
	Name        string             `json:"name"`
	Namespace   string             `json:"namespace"`
	CPUUsage    int64              `json:"cpu_usage_millicores"`
	MemoryUsage int64              `json:"memory_usage_bytes"`
	Containers  []ContainerMetrics `json:"containers"`
	Node        string             `json:"node"`
	Status      string             `json:"status"`
}

// ContainerMetrics represents metrics for a container
type ContainerMetrics struct {
	Name        string `json:"name"`
	CPUUsage    int64  `json:"cpu_usage_millicores"`
	MemoryUsage int64  `json:"memory_usage_bytes"`
}

// ClusterSummary aggregates cluster-wide metrics
type ClusterSummary struct {
	TotalNodes         int       `json:"total_nodes"`
	ReadyNodes         int       `json:"ready_nodes"`
	TotalCPUMillicores int64     `json:"total_cpu_millicores"`
	UsedCPUMillicores  int64     `json:"used_cpu_millicores"`
	CPUPercent         float64   `json:"cpu_percent"`
	TotalMemoryBytes   int64     `json:"total_memory_bytes"`
	UsedMemoryBytes    int64     `json:"used_memory_bytes"`
	MemoryPercent      float64   `json:"memory_percent"`
	TotalPods          int       `json:"total_pods"`
	RunningPods        int       `json:"running_pods"`
	Timestamp          time.Time `json:"timestamp"`
}

// ResourceQuotaInfo represents a resource quota and its usage
type ResourceQuotaInfo struct {
	Name      string               `json:"name"`
	Namespace string               `json:"namespace"`
	Hard      map[string]string    `json:"hard"`
	Used      map[string]string    `json:"used"`
	Usage     []ResourceQuotaUsage `json:"usage"`
	CreatedAt string               `json:"created_at"`
	Age       string               `json:"age"`
}

// ResourceQuotaUsage represents usage for a single resource type
type ResourceQuotaUsage struct {
	Resource string  `json:"resource"`
	Hard     string  `json:"hard"`
	Used     string  `json:"used"`
	Percent  float64 `json:"percent"`
	Status   string  `json:"status"` // ok, warning, critical
}

// LimitRangeInfo represents a LimitRange
type LimitRangeInfo struct {
	Name      string           `json:"name"`
	Namespace string           `json:"namespace"`
	Limits    []LimitRangeItem `json:"limits"`
	CreatedAt string           `json:"created_at"`
	Age       string           `json:"age"`
}

// LimitRangeItem represents a single limit item
type LimitRangeItem struct {
	Type           string            `json:"type"`
	Max            map[string]string `json:"max,omitempty"`
	Min            map[string]string `json:"min,omitempty"`
	Default        map[string]string `json:"default,omitempty"`
	DefaultRequest map[string]string `json:"default_request,omitempty"`
}

// HPAInfo represents a HorizontalPodAutoscaler
type HPAInfo struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	TargetKind      string   `json:"target_kind"`
	TargetName      string   `json:"target_name"`
	MinReplicas     int32    `json:"min_replicas"`
	MaxReplicas     int32    `json:"max_replicas"`
	CurrentReplicas int32    `json:"current_replicas"`
	DesiredReplicas int32    `json:"desired_replicas"`
	CurrentCPU      *int32   `json:"current_cpu_percent,omitempty"`
	TargetCPU       *int32   `json:"target_cpu_percent,omitempty"`
	CurrentMemory   *int32   `json:"current_memory_percent,omitempty"`
	TargetMemory    *int32   `json:"target_memory_percent,omitempty"`
	Conditions      []string `json:"conditions"`
	LastScaleTime   string   `json:"last_scale_time,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Age             string   `json:"age"`
}

// CostConfig holds pricing configuration
type CostConfig struct {
	CPUCostPerHour    float64 `json:"cpu_cost_per_hour"`    // $ per vCPU-hour
	MemoryCostPerGBHr float64 `json:"memory_cost_per_gb_hour"` // $ per GiB-hour
	StorageCostPerGB  float64 `json:"storage_cost_per_gb_month"`  // $ per GB-month
	Currency          string  `json:"currency"`
}

// NamespaceCost represents cost estimation for a namespace
type NamespaceCost struct {
	Namespace        string        `json:"namespace"`
	PodCount         int           `json:"pod_count"`
	CPURequested     int64         `json:"cpu_requested_millicores"`
	CPUUsed          int64         `json:"cpu_used_millicores"`
	MemoryRequested  int64         `json:"memory_requested_bytes"`
	MemoryUsed       int64         `json:"memory_used_bytes"`
	StorageRequested int64         `json:"storage_requested_bytes"`
	CostHourly       float64       `json:"cost_hourly"`
	CostDaily        float64       `json:"cost_daily"`
	CostMonthly      float64       `json:"cost_monthly"`
	Breakdown        CostBreakdown `json:"breakdown"`
}

// CostBreakdown shows cost by resource type
type CostBreakdown struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Storage float64 `json:"storage"`
}

// ClusterCostSummary aggregates cluster-wide costs
type ClusterCostSummary struct {
	TotalHourly   float64         `json:"total_hourly"`
	TotalDaily    float64         `json:"total_daily"`
	TotalMonthly  float64         `json:"total_monthly"`
	ByNamespace   []NamespaceCost `json:"by_namespace"`
	TopNamespaces []NamespaceCost `json:"top_namespaces"`
	Currency      string          `json:"currency"`
	CostConfig    CostConfig      `json:"cost_config"`
	Timestamp     time.Time       `json:"timestamp"`
}

// DefaultCostConfig returns the default cost configuration
func DefaultCostConfig() CostConfig {
	return CostConfig{
		CPUCostPerHour:    0.05,  // $0.05 per vCPU-hour
		MemoryCostPerGBHr: 0.01,  // $0.01 per GiB-hour
		StorageCostPerGB:  0.10,  // $0.10 per GB-month
		Currency:          "USD",
	}
}
