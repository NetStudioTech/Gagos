package cicd

import (
	"time"
)

// Pipeline represents a CI/CD pipeline definition
type Pipeline struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Spec        PipelineSpec      `json:"spec"`
	Status      PipelineStatus    `json:"status"`
	YAML        string            `json:"yaml"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PipelineSpec defines the pipeline specification
type PipelineSpec struct {
	Triggers  []Trigger         `json:"triggers,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
	Jobs      []JobSpec         `json:"jobs"`
	Artifacts []ArtifactSpec    `json:"artifacts,omitempty"`
}

// Trigger defines how a pipeline can be triggered
type Trigger struct {
	Type     string `json:"type"` // webhook, cron
	Secret   string `json:"secret,omitempty"`
	Schedule string `json:"schedule,omitempty"` // cron expression
	Enabled  bool   `json:"enabled"`
}

// JobSpec defines a single job in the pipeline
type JobSpec struct {
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Workdir    string            `json:"workdir,omitempty"`
	Script     string            `json:"script"`
	Env        []EnvVar          `json:"env,omitempty"`
	Secrets    []SecretMount     `json:"secrets,omitempty"`
	Resources  ResourceSpec      `json:"resources,omitempty"`
	Timeout    int               `json:"timeout,omitempty"` // seconds, default 600
	Privileged bool              `json:"privileged,omitempty"`
	DependsOn  []string          `json:"dependsOn,omitempty"`
	SkipIf     string            `json:"skipIf,omitempty"` // Variable name - if set to "true", job is skipped
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SecretMount defines how to mount a K8s secret
type SecretMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	Key       string `json:"key"`
}

// ResourceSpec defines resource limits/requests
type ResourceSpec struct {
	Limits   ResourceList `json:"limits,omitempty"`
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList defines CPU and memory
type ResourceList struct {
	Memory string `json:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty"`
}

// ArtifactSpec defines artifacts to collect
type ArtifactSpec struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// PipelineStatus holds runtime status info
type PipelineStatus struct {
	WebhookURL   string     `json:"webhook_url,omitempty"`
	WebhookToken string     `json:"webhook_token,omitempty"`
	LastRunID    string     `json:"last_run_id,omitempty"`
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	TotalRuns    int        `json:"total_runs"`
}

// RunStatus represents the status of a pipeline run
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusSkipped   RunStatus = "skipped"
)

// PipelineRun represents a single execution of a pipeline
type PipelineRun struct {
	ID           string            `json:"id"`
	PipelineID   string            `json:"pipeline_id"`
	PipelineName string            `json:"pipeline_name"`
	RunNumber    int               `json:"run_number"`
	Status       RunStatus         `json:"status"`
	TriggerType  string            `json:"trigger_type"` // manual, webhook, cron
	TriggerRef   string            `json:"trigger_ref,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Jobs         []JobRun          `json:"jobs"`
	Artifacts    []ArtifactResult  `json:"artifacts,omitempty"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	FinishedAt   *time.Time        `json:"finished_at,omitempty"`
	Duration     int64             `json:"duration_ms,omitempty"`
	Error        string            `json:"error,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// JobRun represents a single job execution within a run
type JobRun struct {
	Name       string     `json:"name"`
	Status     RunStatus  `json:"status"`
	K8sJobName string     `json:"k8s_job_name,omitempty"`
	K8sPodName string     `json:"k8s_pod_name,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Duration   int64      `json:"duration_ms,omitempty"`
	ExitCode   int        `json:"exit_code,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// ArtifactResult represents a collected artifact
type ArtifactResult struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	StorageID string    `json:"storage_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ArtifactMetadata for stored artifacts
type ArtifactMetadata struct {
	ID         string     `json:"id"`
	RunID      string     `json:"run_id"`
	PipelineID string     `json:"pipeline_id"`
	Name       string     `json:"name"`
	Filename   string     `json:"filename"`
	Path       string     `json:"path"` // Storage path
	Size       int64      `json:"size"`
	MimeType   string     `json:"mime_type"`
	Checksum   string     `json:"checksum"` // SHA256
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// PipelineYAML represents the YAML structure for parsing
type PipelineYAML struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   MetadataYAML `yaml:"metadata"`
	Spec       SpecYAML     `yaml:"spec"`
}

// MetadataYAML for pipeline metadata
type MetadataYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

// SpecYAML for pipeline spec from YAML
type SpecYAML struct {
	Triggers  []TriggerYAML         `yaml:"triggers,omitempty"`
	Variables map[string]string     `yaml:"variables,omitempty"`
	Jobs      []JobYAML             `yaml:"jobs"`
	Artifacts []ArtifactSpecYAML    `yaml:"artifacts,omitempty"`
}

// TriggerYAML for trigger definition
type TriggerYAML struct {
	Type     string `yaml:"type"`
	Secret   string `yaml:"secret,omitempty"`
	Schedule string `yaml:"schedule,omitempty"`
	Enabled  *bool  `yaml:"enabled,omitempty"`
}

// JobYAML for job definition
type JobYAML struct {
	Name       string            `yaml:"name"`
	Image      string            `yaml:"image"`
	Workdir    string            `yaml:"workdir,omitempty"`
	Script     string            `yaml:"script"`
	Env        []EnvVarYAML      `yaml:"env,omitempty"`
	Secrets    []SecretMountYAML `yaml:"secrets,omitempty"`
	Resources  ResourceSpecYAML  `yaml:"resources,omitempty"`
	Timeout    int               `yaml:"timeout,omitempty"`
	Privileged bool              `yaml:"privileged,omitempty"`
	DependsOn  []string          `yaml:"dependsOn,omitempty"`
	SkipIf     string            `yaml:"skipIf,omitempty"`
}

// EnvVarYAML for env var
type EnvVarYAML struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// SecretMountYAML for secret mount
type SecretMountYAML struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
	Key       string `yaml:"key"`
}

// ResourceSpecYAML for resources
type ResourceSpecYAML struct {
	Limits   ResourceListYAML `yaml:"limits,omitempty"`
	Requests ResourceListYAML `yaml:"requests,omitempty"`
}

// ResourceListYAML for resource limits
type ResourceListYAML struct {
	Memory string `yaml:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty"`
}

// ArtifactSpecYAML for artifact spec
type ArtifactSpecYAML struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// WebSocket message types
type WsMessage struct {
	Type      string `json:"type"` // log, status, complete, error
	Line      string `json:"line,omitempty"`
	Status    string `json:"status,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
}

// API Request/Response types

type CreatePipelineRequest struct {
	YAML string `json:"yaml"`
}

type CreatePipelineResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	WebhookURL string `json:"webhook_url,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type TriggerPipelineRequest struct {
	Variables map[string]string `json:"variables,omitempty"`
}

type TriggerPipelineResponse struct {
	RunID     string `json:"run_id"`
	RunNumber int    `json:"run_number"`
	Status    string `json:"status"`
}

type ListPipelinesResponse struct {
	Count     int        `json:"count"`
	Pipelines []Pipeline `json:"pipelines"`
}

type ListRunsResponse struct {
	Count int           `json:"count"`
	Runs  []PipelineRun `json:"runs"`
}

type ListArtifactsResponse struct {
	Count     int                `json:"count"`
	Artifacts []ArtifactMetadata `json:"artifacts"`
}

// Stats for overview dashboard
type CICDStats struct {
	TotalPipelines int `json:"total_pipelines"`
	TotalRuns      int `json:"total_runs"`
	RunningRuns    int `json:"running_runs"`
	Succeeded24h   int `json:"succeeded_24h"`
	Failed24h      int `json:"failed_24h"`
}
