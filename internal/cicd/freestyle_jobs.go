package cicd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gaga951/gagos/internal/storage"

	"github.com/rs/zerolog/log"
)

// generateFreestyleJobID generates a unique ID for a freestyle job
func generateFreestyleJobID() string {
	return generateID("fsj")
}

// generateBuildStepID generates a unique ID for a build step
func generateBuildStepID() string {
	return generateID("step")
}

// CreateFreestyleJob creates a new freestyle job
func CreateFreestyleJob(req *CreateFreestyleJobRequest) (*FreestyleJob, error) {
	// Assign IDs to build steps
	for i := range req.BuildSteps {
		if req.BuildSteps[i].ID == "" {
			req.BuildSteps[i].ID = generateBuildStepID()
		}
		req.BuildSteps[i].Order = i
		// Set default timeout if not specified
		if req.BuildSteps[i].Timeout == 0 {
			req.BuildSteps[i].Timeout = 300 // 5 minutes default
		}
	}

	job := &FreestyleJob{
		ID:          generateFreestyleJobID(),
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		Parameters:  req.Parameters,
		Environment: req.Environment,
		BuildSteps:  req.BuildSteps,
		Triggers:    req.Triggers,
		Status: FreestyleJobStatus{
			TotalBuilds: 0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate webhook URL, token, and secret if webhook trigger is enabled
	for _, t := range job.Triggers {
		if t.Type == "webhook" && t.Enabled {
			job.Status.WebhookToken = generateID("wh")
			job.Status.WebhookSecret = generateID("whsec")
			job.Status.WebhookURL = fmt.Sprintf("/api/v1/cicd/freestyle/webhook/%s", job.Status.WebhookToken)
			break
		}
	}

	// Save to storage
	data, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketFreestyleJobs, job.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Register with scheduler if has cron trigger
	if sched := GetScheduler(); sched != nil {
		sched.RegisterFreestyleJob(job)
	}

	log.Info().Str("id", job.ID).Str("name", job.Name).Msg("Freestyle job created")
	return job, nil
}

// GetFreestyleJob retrieves a freestyle job by ID
func GetFreestyleJob(id string) (*FreestyleJob, error) {
	data, err := storage.GetBackend().Get(storage.BucketFreestyleJobs, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("freestyle job not found: %s", id)
	}

	var job FreestyleJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetFreestyleJobByWebhookToken retrieves a job by its webhook token
func GetFreestyleJobByWebhookToken(token string) (*FreestyleJob, error) {
	jobs, err := ListFreestyleJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range jobs {
		if job.Status.WebhookToken == token {
			return job, nil
		}
	}

	return nil, fmt.Errorf("job not found for webhook token")
}

// ListFreestyleJobs returns all freestyle jobs
func ListFreestyleJobs() ([]*FreestyleJob, error) {
	dataList, err := storage.GetBackend().List(storage.BucketFreestyleJobs)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	jobs := make([]*FreestyleJob, 0, len(dataList))
	for _, data := range dataList {
		var job FreestyleJob
		if err := json.Unmarshal(data, &job); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal freestyle job")
			continue
		}
		jobs = append(jobs, &job)
	}

	// Sort by name
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Name < jobs[j].Name
	})

	return jobs, nil
}

// UpdateFreestyleJob updates an existing freestyle job
func UpdateFreestyleJob(id string, req *CreateFreestyleJobRequest) (*FreestyleJob, error) {
	job, err := GetFreestyleJob(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	job.Name = req.Name
	job.Description = req.Description
	job.Enabled = req.Enabled
	job.Parameters = req.Parameters
	job.Environment = req.Environment
	job.Triggers = req.Triggers

	// Update build steps with IDs
	for i := range req.BuildSteps {
		if req.BuildSteps[i].ID == "" {
			req.BuildSteps[i].ID = generateBuildStepID()
		}
		req.BuildSteps[i].Order = i
		if req.BuildSteps[i].Timeout == 0 {
			req.BuildSteps[i].Timeout = 300
		}
	}
	job.BuildSteps = req.BuildSteps

	// Update webhook URL/token if webhook trigger changed
	hasWebhook := false
	for _, t := range job.Triggers {
		if t.Type == "webhook" && t.Enabled {
			hasWebhook = true
			break
		}
	}

	if hasWebhook && job.Status.WebhookToken == "" {
		job.Status.WebhookToken = generateID("wh")
		job.Status.WebhookSecret = generateID("whsec")
		job.Status.WebhookURL = fmt.Sprintf("/api/v1/cicd/freestyle/webhook/%s", job.Status.WebhookToken)
	} else if !hasWebhook {
		job.Status.WebhookToken = ""
		job.Status.WebhookSecret = ""
		job.Status.WebhookURL = ""
	}

	job.UpdatedAt = time.Now()

	// Save to storage
	data, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketFreestyleJobs, job.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Update scheduler registration
	if sched := GetScheduler(); sched != nil {
		sched.RegisterFreestyleJob(job)
	}

	log.Info().Str("id", job.ID).Str("name", job.Name).Msg("Freestyle job updated")
	return job, nil
}

// DeleteFreestyleJob deletes a freestyle job
func DeleteFreestyleJob(id string) error {
	// Check if job exists
	_, err := GetFreestyleJob(id)
	if err != nil {
		return err
	}

	// Unregister from scheduler
	if sched := GetScheduler(); sched != nil {
		sched.UnregisterFreestyleJob(id)
	}

	// Delete all builds for this job
	builds, err := ListFreestyleBuildsForJob(id)
	if err == nil {
		for _, build := range builds {
			storage.GetBackend().Delete(storage.BucketFreestyleBuilds, build.ID)
		}
	}

	if err := storage.GetBackend().Delete(storage.BucketFreestyleJobs, id); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	log.Info().Str("id", id).Msg("Freestyle job deleted")
	return nil
}

// UpdateFreestyleJobStatus updates the job's runtime status after a build
func UpdateFreestyleJobStatus(jobID string, buildID string, status string) error {
	job, err := GetFreestyleJob(jobID)
	if err != nil {
		return err
	}

	now := time.Now()
	job.Status.LastBuildID = buildID
	job.Status.LastBuildAt = &now
	job.Status.LastStatus = status
	job.Status.TotalBuilds++
	job.UpdatedAt = now

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketFreestyleJobs, job.ID, data); err != nil {
		return fmt.Errorf("failed to save job status: %w", err)
	}

	return nil
}

// GetNextBuildNumber returns the next build number for a job
func GetNextBuildNumber(jobID string) (int, error) {
	builds, err := ListFreestyleBuildsForJob(jobID)
	if err != nil {
		return 1, nil // Start at 1 if no builds
	}

	maxNum := 0
	for _, b := range builds {
		if b.BuildNumber > maxNum {
			maxNum = b.BuildNumber
		}
	}

	return maxNum + 1, nil
}
