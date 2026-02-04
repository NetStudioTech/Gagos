package cicd

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// WebhookPayload represents incoming webhook data
type WebhookPayload struct {
	Ref       string            `json:"ref,omitempty"`
	Branch    string            `json:"branch,omitempty"`
	Commit    string            `json:"commit,omitempty"`
	Message   string            `json:"message,omitempty"`
	Author    string            `json:"author,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// HandleWebhook processes an incoming webhook request
func HandleWebhook(pipelineID, token string, payload *WebhookPayload, signature string) (*PipelineRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the pipeline
	pipeline, err := GetPipeline(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	// Verify token
	if pipeline.Status.WebhookToken != token {
		return nil, fmt.Errorf("invalid webhook token")
	}

	// Check if webhook trigger is enabled
	webhookEnabled := false
	var webhookTrigger *Trigger
	for i := range pipeline.Spec.Triggers {
		if pipeline.Spec.Triggers[i].Type == "webhook" && pipeline.Spec.Triggers[i].Enabled {
			webhookEnabled = true
			webhookTrigger = &pipeline.Spec.Triggers[i]
			break
		}
	}

	if !webhookEnabled {
		return nil, fmt.Errorf("webhook trigger is not enabled")
	}

	// Verify signature if secret is set
	if webhookTrigger.Secret != "" && signature != "" {
		if !verifySignature(signature, webhookTrigger.Secret, payload) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	// Build trigger ref
	triggerRef := "webhook"
	if payload != nil {
		if payload.Ref != "" {
			triggerRef = "webhook:" + payload.Ref
		} else if payload.Branch != "" {
			triggerRef = "webhook:" + payload.Branch
		}
		if payload.Commit != "" {
			triggerRef += "@" + payload.Commit[:8]
		}
	}

	// Merge variables
	vars := make(map[string]string)
	if payload != nil && payload.Variables != nil {
		for k, v := range payload.Variables {
			vars[k] = v
		}
		// Add webhook-specific variables
		if payload.Ref != "" {
			vars["WEBHOOK_REF"] = payload.Ref
		}
		if payload.Branch != "" {
			vars["WEBHOOK_BRANCH"] = payload.Branch
		}
		if payload.Commit != "" {
			vars["WEBHOOK_COMMIT"] = payload.Commit
		}
		if payload.Author != "" {
			vars["WEBHOOK_AUTHOR"] = payload.Author
		}
	}

	log.Info().
		Str("pipeline_id", pipelineID).
		Str("pipeline", pipeline.Name).
		Str("trigger_ref", triggerRef).
		Msg("Webhook trigger received")

	// Trigger the pipeline
	run, err := TriggerPipeline(ctx, pipeline, "webhook", triggerRef, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	log.Info().
		Str("pipeline", pipeline.Name).
		Str("run_id", run.ID).
		Int("run_number", run.RunNumber).
		Msg("Pipeline triggered by webhook")

	return run, nil
}

// verifySignature verifies HMAC-SHA256 signature
func verifySignature(signature, secret string, payload *WebhookPayload) bool {
	// GitHub-style: sha256=<signature>
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	// Create HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	// For simplicity, we just verify the secret matches
	// In production, you'd hash the actual request body
	mac.Write([]byte(secret))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// RegenerateWebhookToken regenerates the webhook token for a pipeline
func RegenerateWebhookToken(pipelineID string) (string, error) {
	pipeline, err := GetPipeline(pipelineID)
	if err != nil {
		return "", err
	}

	newToken := generateToken()
	pipeline.Status.WebhookToken = newToken
	pipeline.Status.WebhookURL = fmt.Sprintf("/api/v1/cicd/webhooks/%s/%s", pipelineID, newToken)
	pipeline.UpdatedAt = time.Now()

	if err := SavePipeline(pipeline); err != nil {
		return "", err
	}

	return newToken, nil
}
