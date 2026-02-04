package monitoring

import (
	"fmt"
	"sync"

	"github.com/gaga951/gagos/internal/k8s"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	metricsClient *metricsv.Clientset
	k8sClient     *kubernetes.Clientset
	costConfig    CostConfig
	configMu      sync.RWMutex
	initOnce      sync.Once
	initErr       error
)

// Init initializes the monitoring package
func Init() error {
	initOnce.Do(func() {
		// Get K8s client
		k8sClient = k8s.GetClient()
		if k8sClient == nil {
			initErr = fmt.Errorf("kubernetes client not initialized")
			return
		}

		// Get REST config for metrics client
		config := k8s.GetConfig()
		if config == nil {
			initErr = fmt.Errorf("kubernetes config not available")
			return
		}

		// Create metrics client
		var err error
		metricsClient, err = metricsv.NewForConfig(config)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to create metrics client - metrics will be unavailable")
			// Don't fail - other monitoring features can still work
		} else {
			log.Info().Msg("Metrics client initialized")
		}

		// Initialize default cost config
		costConfig = DefaultCostConfig()
	})

	return initErr
}

// GetMetricsClient returns the metrics client
func GetMetricsClient() *metricsv.Clientset {
	return metricsClient
}

// GetK8sClient returns the kubernetes client
func GetK8sClient() *kubernetes.Clientset {
	return k8sClient
}

// IsMetricsAvailable returns true if metrics-server is available
func IsMetricsAvailable() bool {
	return metricsClient != nil
}

// GetCostConfig returns the current cost configuration
func GetCostConfig() CostConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return costConfig
}

// SetCostConfig updates the cost configuration
func SetCostConfig(config CostConfig) {
	configMu.Lock()
	defer configMu.Unlock()
	costConfig = config
	log.Info().
		Float64("cpu_cost", config.CPUCostPerHour).
		Float64("memory_cost", config.MemoryCostPerGBHr).
		Str("currency", config.Currency).
		Msg("Cost configuration updated")
}
