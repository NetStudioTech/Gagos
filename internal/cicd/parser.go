package cicd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ParsePipelineYAML parses and validates pipeline YAML
func ParsePipelineYAML(yamlContent string) (*Pipeline, error) {
	var pipelineYAML PipelineYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &pipelineYAML); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Validate required fields
	if err := validatePipelineYAML(&pipelineYAML); err != nil {
		return nil, err
	}

	// Convert to Pipeline
	pipeline := convertYAMLToPipeline(&pipelineYAML, yamlContent)

	return pipeline, nil
}

// validatePipelineYAML validates the parsed YAML structure
func validatePipelineYAML(p *PipelineYAML) error {
	// Validate apiVersion
	if p.APIVersion != "gagos.io/v1" {
		return fmt.Errorf("unsupported apiVersion: %s (expected gagos.io/v1)", p.APIVersion)
	}

	// Validate kind
	if p.Kind != "Pipeline" {
		return fmt.Errorf("invalid kind: %s (expected Pipeline)", p.Kind)
	}

	// Validate metadata
	if p.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Validate name format (alphanumeric, dashes, underscores)
	if !isValidName(p.Metadata.Name) {
		return fmt.Errorf("metadata.name must contain only alphanumeric characters, dashes, and underscores")
	}

	// Validate jobs
	if len(p.Spec.Jobs) == 0 {
		return fmt.Errorf("at least one job is required in spec.jobs")
	}

	jobNames := make(map[string]bool)
	for i, job := range p.Spec.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job[%d].name is required", i)
		}
		if !isValidName(job.Name) {
			return fmt.Errorf("job[%d].name must contain only alphanumeric characters, dashes, and underscores", i)
		}
		if jobNames[job.Name] {
			return fmt.Errorf("duplicate job name: %s", job.Name)
		}
		jobNames[job.Name] = true

		if job.Image == "" {
			return fmt.Errorf("job[%d].image is required", i)
		}
		if job.Script == "" {
			return fmt.Errorf("job[%d].script is required", i)
		}

		// Validate dependsOn references
		for _, dep := range job.DependsOn {
			if !jobNames[dep] {
				// Check if it's defined later
				found := false
				for _, j := range p.Spec.Jobs {
					if j.Name == dep {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("job[%d] references unknown dependency: %s", i, dep)
				}
			}
		}
	}

	// Validate triggers
	for i, trigger := range p.Spec.Triggers {
		if trigger.Type != "webhook" && trigger.Type != "cron" {
			return fmt.Errorf("trigger[%d].type must be 'webhook' or 'cron'", i)
		}
		if trigger.Type == "cron" && trigger.Schedule == "" {
			return fmt.Errorf("trigger[%d].schedule is required for cron triggers", i)
		}
	}

	return nil
}

// isValidName checks if a name contains only valid characters
func isValidName(name string) bool {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return len(name) > 0 && len(name) <= 63
}

// convertYAMLToPipeline converts the YAML struct to Pipeline
func convertYAMLToPipeline(p *PipelineYAML, yamlContent string) *Pipeline {
	now := time.Now()
	id := generateID("pl")
	webhookToken := generateToken()

	pipeline := &Pipeline{
		ID:          id,
		Name:        p.Metadata.Name,
		Description: p.Metadata.Description,
		Labels:      p.Metadata.Labels,
		YAML:        yamlContent,
		CreatedAt:   now,
		UpdatedAt:   now,
		Spec: PipelineSpec{
			Variables: p.Spec.Variables,
			Jobs:      make([]JobSpec, 0, len(p.Spec.Jobs)),
			Artifacts: make([]ArtifactSpec, 0, len(p.Spec.Artifacts)),
			Triggers:  make([]Trigger, 0, len(p.Spec.Triggers)),
		},
		Status: PipelineStatus{
			TotalRuns: 0,
		},
	}

	// Convert jobs
	for _, j := range p.Spec.Jobs {
		job := JobSpec{
			Name:       j.Name,
			Image:      j.Image,
			Workdir:    j.Workdir,
			Script:     j.Script,
			Timeout:    j.Timeout,
			Privileged: j.Privileged,
			DependsOn:  j.DependsOn,
			SkipIf:     j.SkipIf,
		}

		if job.Timeout == 0 {
			job.Timeout = 600 // Default 10 minutes
		}

		// Convert env vars
		for _, e := range j.Env {
			job.Env = append(job.Env, EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		// Convert secrets
		for _, s := range j.Secrets {
			job.Secrets = append(job.Secrets, SecretMount{
				Name:      s.Name,
				MountPath: s.MountPath,
				Key:       s.Key,
			})
		}

		// Convert resources
		job.Resources = ResourceSpec{
			Limits: ResourceList{
				Memory: j.Resources.Limits.Memory,
				CPU:    j.Resources.Limits.CPU,
			},
			Requests: ResourceList{
				Memory: j.Resources.Requests.Memory,
				CPU:    j.Resources.Requests.CPU,
			},
		}

		pipeline.Spec.Jobs = append(pipeline.Spec.Jobs, job)
	}

	// Convert artifacts
	for _, a := range p.Spec.Artifacts {
		pipeline.Spec.Artifacts = append(pipeline.Spec.Artifacts, ArtifactSpec{
			Name: a.Name,
			Path: a.Path,
		})
	}

	// Convert triggers and set webhook URL if needed
	hasWebhook := false
	for _, t := range p.Spec.Triggers {
		enabled := true
		if t.Enabled != nil {
			enabled = *t.Enabled
		}
		trigger := Trigger{
			Type:     t.Type,
			Secret:   t.Secret,
			Schedule: t.Schedule,
			Enabled:  enabled,
		}
		pipeline.Spec.Triggers = append(pipeline.Spec.Triggers, trigger)

		if t.Type == "webhook" {
			hasWebhook = true
		}
	}

	// Set webhook URL if there's a webhook trigger
	if hasWebhook {
		pipeline.Status.WebhookURL = fmt.Sprintf("/api/v1/cicd/webhooks/%s/%s", id, webhookToken)
		pipeline.Status.WebhookToken = webhookToken
	}

	return pipeline
}

// generateID generates a unique ID with prefix
func generateID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes))
}

// generateToken generates a secure webhook token
func generateToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetSamplePipelineYAML returns a sample pipeline for the UI
func GetSamplePipelineYAML() string {
	return strings.TrimSpace(`
apiVersion: gagos.io/v1
kind: Pipeline
metadata:
  name: app-pipeline
  description: "Build, test and deploy application"

spec:
  triggers:
    - type: webhook

  variables:
    GIT_REPO: "https://github.com/user/repo.git"
    GIT_BRANCH: "main"
    REGISTRY: "localhost:5000"
    IMAGE_NAME: "myapp"
    IMAGE_TAG: "latest"
    DOCKERFILE: "Dockerfile"
    DEPLOY_NAMESPACE: "default"
    DEPLOY_NAME: "myapp"
    SKIP_TEST: "false"
    SKIP_DEPLOY: "false"

  jobs:
    - name: build
      image: gcr.io/kaniko-project/executor:debug
      script: |
        /kaniko/executor \
          --dockerfile=${DOCKERFILE} \
          --context=git://${GIT_REPO}#refs/heads/${GIT_BRANCH} \
          --destination=${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} \
          --insecure \
          --skip-tls-verify
      timeout: 600
      resources:
        limits:
          memory: "1Gi"
          cpu: "500m"

    - name: test
      image: alpine:latest
      skipIf: SKIP_TEST
      script: |
        echo "Running tests for ${IMAGE_NAME}:${IMAGE_TAG}..."
        # Add your test commands here
        echo "Tests passed!"
      dependsOn: [build]
      timeout: 300

    - name: deploy
      image: bitnami/kubectl:latest
      skipIf: SKIP_DEPLOY
      script: |
        kubectl set image deployment/${DEPLOY_NAME} \
          ${DEPLOY_NAME}=${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} \
          -n ${DEPLOY_NAMESPACE}
        kubectl rollout status deployment/${DEPLOY_NAME} \
          -n ${DEPLOY_NAMESPACE} --timeout=120s
      dependsOn: [test]
      timeout: 300
`)
}
