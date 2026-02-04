package cicd

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gaga951/gagos/internal/storage"
	"github.com/rs/zerolog/log"
)

// NotificationType defines the notification channel type
type NotificationType string

const (
	NotificationTypeWebhook NotificationType = "webhook"
	NotificationTypeSlack   NotificationType = "slack"
	NotificationTypeEmail   NotificationType = "email"
)

// NotificationEvent defines when to send notifications
type NotificationEvent string

const (
	NotificationEventBuildStarted   NotificationEvent = "build_started"
	NotificationEventBuildSucceeded NotificationEvent = "build_succeeded"
	NotificationEventBuildFailed    NotificationEvent = "build_failed"
	NotificationEventBuildCancelled NotificationEvent = "build_cancelled"
	NotificationEventRunStarted     NotificationEvent = "run_started"
	NotificationEventRunSucceeded   NotificationEvent = "run_succeeded"
	NotificationEventRunFailed      NotificationEvent = "run_failed"
	NotificationEventRunCancelled   NotificationEvent = "run_cancelled"
)

// NotificationConfig represents a notification configuration
type NotificationConfig struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Type        NotificationType    `json:"type"`
	Enabled     bool                `json:"enabled"`
	Events      []NotificationEvent `json:"events"`       // Events to notify on
	URL         string              `json:"url"`          // Webhook URL
	Secret      string              `json:"secret"`       // For HMAC signing
	Headers     map[string]string   `json:"headers"`      // Custom headers
	JobIDs      []string            `json:"job_ids"`      // Filter by job IDs (empty = all)
	PipelineIDs []string            `json:"pipeline_ids"` // Filter by pipeline IDs (empty = all)
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// NotificationPayload is the webhook payload structure
type NotificationPayload struct {
	Event       NotificationEvent `json:"event"`
	Timestamp   time.Time         `json:"timestamp"`
	Build       *BuildNotification `json:"build,omitempty"`
	PipelineRun *RunNotification   `json:"pipeline_run,omitempty"`
}

// BuildNotification contains build info for notification
type BuildNotification struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	JobName     string `json:"job_name"`
	BuildNumber int    `json:"build_number"`
	Status      string `json:"status"`
	TriggerType string `json:"trigger_type"`
	Duration    int64  `json:"duration_ms,omitempty"`
	Error       string `json:"error,omitempty"`
	URL         string `json:"url,omitempty"`
}

// RunNotification contains pipeline run info for notification
type RunNotification struct {
	ID          string `json:"id"`
	PipelineID  string `json:"pipeline_id"`
	PipelineName string `json:"pipeline_name"`
	RunNumber   int    `json:"run_number"`
	Status      string `json:"status"`
	TriggerType string `json:"trigger_type"`
	Duration    int64  `json:"duration_ms,omitempty"`
	Error       string `json:"error,omitempty"`
	URL         string `json:"url,omitempty"`
}

var (
	notificationConfigs   = make(map[string]*NotificationConfig)
	notificationConfigsMu sync.RWMutex
	httpClient            = &http.Client{Timeout: 10 * time.Second}
)

// generateNotificationID generates a unique ID for a notification config
func generateNotificationID() string {
	return generateID("notif")
}

// CreateNotificationConfig creates a new notification configuration
func CreateNotificationConfig(config *NotificationConfig) (*NotificationConfig, error) {
	config.ID = generateNotificationID()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification config: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketNotifications, config.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save notification config: %w", err)
	}

	// Update in-memory cache
	notificationConfigsMu.Lock()
	notificationConfigs[config.ID] = config
	notificationConfigsMu.Unlock()

	log.Info().Str("id", config.ID).Str("name", config.Name).Msg("Notification config created")
	return config, nil
}

// GetNotificationConfig retrieves a notification config by ID
func GetNotificationConfig(id string) (*NotificationConfig, error) {
	// Check cache first
	notificationConfigsMu.RLock()
	if config, ok := notificationConfigs[id]; ok {
		notificationConfigsMu.RUnlock()
		return config, nil
	}
	notificationConfigsMu.RUnlock()

	data, err := storage.GetBackend().Get(storage.BucketNotifications, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification config: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("notification config not found: %s", id)
	}

	var config NotificationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification config: %w", err)
	}

	return &config, nil
}

// ListNotificationConfigs returns all notification configurations
func ListNotificationConfigs() ([]*NotificationConfig, error) {
	dataList, err := storage.GetBackend().List(storage.BucketNotifications)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification configs: %w", err)
	}

	configs := make([]*NotificationConfig, 0, len(dataList))
	for _, data := range dataList {
		var config NotificationConfig
		if err := json.Unmarshal(data, &config); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal notification config")
			continue
		}
		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateNotificationConfig updates an existing notification config
func UpdateNotificationConfig(id string, config *NotificationConfig) (*NotificationConfig, error) {
	existing, err := GetNotificationConfig(id)
	if err != nil {
		return nil, err
	}

	config.ID = existing.ID
	config.CreatedAt = existing.CreatedAt
	config.UpdatedAt = time.Now()

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification config: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketNotifications, config.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save notification config: %w", err)
	}

	// Update cache
	notificationConfigsMu.Lock()
	notificationConfigs[config.ID] = config
	notificationConfigsMu.Unlock()

	log.Info().Str("id", config.ID).Str("name", config.Name).Msg("Notification config updated")
	return config, nil
}

// DeleteNotificationConfig deletes a notification config
func DeleteNotificationConfig(id string) error {
	if err := storage.GetBackend().Delete(storage.BucketNotifications, id); err != nil {
		return fmt.Errorf("failed to delete notification config: %w", err)
	}

	// Remove from cache
	notificationConfigsMu.Lock()
	delete(notificationConfigs, id)
	notificationConfigsMu.Unlock()

	log.Info().Str("id", id).Msg("Notification config deleted")
	return nil
}

// LoadNotificationConfigs loads all configs into memory
func LoadNotificationConfigs() error {
	configs, err := ListNotificationConfigs()
	if err != nil {
		return err
	}

	notificationConfigsMu.Lock()
	defer notificationConfigsMu.Unlock()

	notificationConfigs = make(map[string]*NotificationConfig)
	for _, config := range configs {
		notificationConfigs[config.ID] = config
	}

	log.Info().Int("count", len(configs)).Msg("Notification configs loaded")
	return nil
}

// NotifyBuildEvent sends notifications for a build event
func NotifyBuildEvent(event NotificationEvent, build *FreestyleBuild) {
	go func() {
		notificationConfigsMu.RLock()
		configs := make([]*NotificationConfig, 0, len(notificationConfigs))
		for _, c := range notificationConfigs {
			configs = append(configs, c)
		}
		notificationConfigsMu.RUnlock()

		for _, config := range configs {
			if !config.Enabled {
				continue
			}

			// Check if event is in config events
			eventMatch := false
			for _, e := range config.Events {
				if e == event {
					eventMatch = true
					break
				}
			}
			if !eventMatch {
				continue
			}

			// Check job filter
			if len(config.JobIDs) > 0 {
				jobMatch := false
				for _, jid := range config.JobIDs {
					if jid == build.JobID {
						jobMatch = true
						break
					}
				}
				if !jobMatch {
					continue
				}
			}

			// Send notification
			payload := NotificationPayload{
				Event:     event,
				Timestamp: time.Now(),
				Build: &BuildNotification{
					ID:          build.ID,
					JobID:       build.JobID,
					JobName:     build.JobName,
					BuildNumber: build.BuildNumber,
					Status:      string(build.Status),
					TriggerType: build.TriggerType,
					Duration:    build.Duration,
					Error:       build.Error,
				},
			}

			sendWebhookNotification(config, payload)
		}
	}()
}

// NotifyPipelineRunEvent sends notifications for a pipeline run event
func NotifyPipelineRunEvent(event NotificationEvent, run *PipelineRun, pipelineName string) {
	go func() {
		notificationConfigsMu.RLock()
		configs := make([]*NotificationConfig, 0, len(notificationConfigs))
		for _, c := range notificationConfigs {
			configs = append(configs, c)
		}
		notificationConfigsMu.RUnlock()

		for _, config := range configs {
			if !config.Enabled {
				continue
			}

			// Check if event is in config events
			eventMatch := false
			for _, e := range config.Events {
				if e == event {
					eventMatch = true
					break
				}
			}
			if !eventMatch {
				continue
			}

			// Check pipeline filter
			if len(config.PipelineIDs) > 0 {
				pipelineMatch := false
				for _, pid := range config.PipelineIDs {
					if pid == run.PipelineID {
						pipelineMatch = true
						break
					}
				}
				if !pipelineMatch {
					continue
				}
			}

			// Calculate duration
			var duration int64
			if run.StartedAt != nil && run.FinishedAt != nil {
				duration = run.FinishedAt.Sub(*run.StartedAt).Milliseconds()
			}

			// Send notification
			payload := NotificationPayload{
				Event:     event,
				Timestamp: time.Now(),
				PipelineRun: &RunNotification{
					ID:           run.ID,
					PipelineID:   run.PipelineID,
					PipelineName: pipelineName,
					RunNumber:    run.RunNumber,
					Status:       string(run.Status),
					TriggerType:  run.TriggerType,
					Duration:     duration,
					Error:        run.Error,
				},
			}

			sendWebhookNotification(config, payload)
		}
	}()
}

// sendWebhookNotification sends a webhook notification
func sendWebhookNotification(config *NotificationConfig, payload NotificationPayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("config", config.Name).Msg("Failed to marshal notification payload")
		return
	}

	req, err := http.NewRequest(http.MethodPost, config.URL, bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Str("config", config.Name).Msg("Failed to create notification request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GAGOS-Webhook/1.0")
	req.Header.Set("X-GAGOS-Event", string(payload.Event))

	// Add custom headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Add HMAC signature if secret is configured
	if config.Secret != "" {
		signature := computeHMAC(data, config.Secret)
		req.Header.Set("X-GAGOS-Signature", "sha256="+signature)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("config", config.Name).Str("url", config.URL).Msg("Failed to send notification")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Warn().
			Str("config", config.Name).
			Str("url", config.URL).
			Int("status", resp.StatusCode).
			Msg("Notification webhook returned error")
	} else {
		log.Debug().
			Str("config", config.Name).
			Str("event", string(payload.Event)).
			Int("status", resp.StatusCode).
			Msg("Notification sent")
	}
}

// computeHMAC computes HMAC-SHA256 signature
func computeHMAC(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyWebhookSignature verifies an incoming webhook signature
func VerifyWebhookSignature(payload []byte, signature string, secret string) bool {
	expected := "sha256=" + computeHMAC(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
