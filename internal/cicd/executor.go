// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0

package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/gaga951/gagos/internal/k8s"
	"github.com/gaga951/gagos/internal/storage"
)

var (
	cicdNamespace   string
	artifactPath    string
)

func init() {
	cicdNamespace = os.Getenv("GAGOS_CICD_NAMESPACE")
	if cicdNamespace == "" {
		cicdNamespace = "default"
	}
	artifactPath = os.Getenv("GAGOS_ARTIFACT_PATH")
	if artifactPath == "" {
		artifactPath = "/data/artifacts"
	}
}

// TriggerPipeline creates a new pipeline run and starts execution
func TriggerPipeline(ctx context.Context, pipeline *Pipeline, triggerType, triggerRef string, vars map[string]string) (*PipelineRun, error) {
	clientset := k8s.GetClient()
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	// Merge variables
	mergedVars := make(map[string]string)
	for k, v := range pipeline.Spec.Variables {
		mergedVars[k] = v
	}
	for k, v := range vars {
		mergedVars[k] = v
	}

	// Create the run
	now := time.Now()
	runID := generateID("run")
	runNumber := pipeline.Status.TotalRuns + 1

	run := &PipelineRun{
		ID:           runID,
		PipelineID:   pipeline.ID,
		PipelineName: pipeline.Name,
		RunNumber:    runNumber,
		Status:       RunStatusPending,
		TriggerType:  triggerType,
		TriggerRef:   triggerRef,
		Variables:    mergedVars,
		Jobs:         make([]JobRun, 0, len(pipeline.Spec.Jobs)),
		CreatedAt:    now,
	}

	// Initialize job runs
	for _, jobSpec := range pipeline.Spec.Jobs {
		run.Jobs = append(run.Jobs, JobRun{
			Name:   jobSpec.Name,
			Status: RunStatusPending,
		})
	}

	// Save the run
	if err := saveRun(run); err != nil {
		return nil, fmt.Errorf("failed to save run: %w", err)
	}

	// Update pipeline status
	pipeline.Status.TotalRuns = runNumber
	pipeline.Status.LastRunID = runID
	pipeline.Status.LastRunAt = &now
	pipeline.UpdatedAt = now
	if err := savePipeline(pipeline); err != nil {
		log.Warn().Err(err).Msg("Failed to update pipeline status")
	}

	// Start execution in background
	go executeRun(pipeline, run, clientset)

	return run, nil
}

// executeRun executes all jobs in the pipeline run
func executeRun(pipeline *Pipeline, run *PipelineRun, clientset *kubernetes.Clientset) {
	ctx := context.Background()

	// Mark run as running
	now := time.Now()
	run.Status = RunStatusRunning
	run.StartedAt = &now
	saveRun(run)

	// Send run started notification
	NotifyPipelineRunEvent(NotificationEventRunStarted, run, pipeline.Name)

	log.Info().Str("run_id", run.ID).Str("pipeline", pipeline.Name).Msg("Starting pipeline run")

	// Execute jobs sequentially (respecting dependencies)
	completed := make(map[string]bool)
	failed := false

	for i := range run.Jobs {
		if failed {
			run.Jobs[i].Status = RunStatusCancelled
			continue
		}

		jobSpec := pipeline.Spec.Jobs[i]

		// Check if job should be skipped via skipIf variable
		if jobSpec.SkipIf != "" {
			skipVal := strings.ToLower(run.Variables[jobSpec.SkipIf])
			if skipVal == "true" || skipVal == "1" || skipVal == "yes" {
				log.Info().Str("job", jobSpec.Name).Str("skipIf", jobSpec.SkipIf).Msg("Job skipped by variable")
				run.Jobs[i].Status = RunStatusSkipped
				completed[jobSpec.Name] = true // Treat as passed for dependencies
				saveRun(run)
				continue
			}
		}

		// Check dependencies (skipped jobs count as passed)
		for _, dep := range jobSpec.DependsOn {
			if !completed[dep] {
				// Find and wait for dependency (should already be complete due to order)
				for j := 0; j < i; j++ {
					if run.Jobs[j].Name == dep && run.Jobs[j].Status != RunStatusSucceeded && run.Jobs[j].Status != RunStatusSkipped {
						failed = true
						run.Jobs[i].Status = RunStatusCancelled
						run.Jobs[i].Error = fmt.Sprintf("dependency %s failed", dep)
						break
					}
				}
			}
		}

		if failed {
			continue
		}

		// Execute the job
		err := executeJob(ctx, clientset, pipeline, run, &run.Jobs[i], &jobSpec)
		if err != nil {
			log.Error().Err(err).Str("job", jobSpec.Name).Msg("Job execution failed")
			run.Jobs[i].Status = RunStatusFailed
			run.Jobs[i].Error = err.Error()
			failed = true
		} else {
			completed[jobSpec.Name] = true
		}

		saveRun(run)
	}

	// Mark run as complete
	finishedAt := time.Now()
	run.FinishedAt = &finishedAt
	if run.StartedAt != nil {
		run.Duration = finishedAt.Sub(*run.StartedAt).Milliseconds()
	}

	if failed {
		run.Status = RunStatusFailed
	} else {
		run.Status = RunStatusSucceeded
	}

	saveRun(run)

	// Send notification based on status
	var event NotificationEvent
	if failed {
		event = NotificationEventRunFailed
	} else {
		event = NotificationEventRunSucceeded
	}
	NotifyPipelineRunEvent(event, run, pipeline.Name)

	log.Info().
		Str("run_id", run.ID).
		Str("status", string(run.Status)).
		Int64("duration_ms", run.Duration).
		Msg("Pipeline run completed")
}

// executeJob creates and monitors a K8s Job for a pipeline job
func executeJob(ctx context.Context, clientset *kubernetes.Clientset, pipeline *Pipeline, run *PipelineRun, jobRun *JobRun, jobSpec *JobSpec) error {
	// Mark job as running
	now := time.Now()
	jobRun.Status = RunStatusRunning
	jobRun.StartedAt = &now

	// Build the K8s Job
	k8sJob := buildK8sJob(pipeline, run, jobSpec)
	jobRun.K8sJobName = k8sJob.Name

	log.Info().
		Str("job", jobSpec.Name).
		Str("k8s_job", k8sJob.Name).
		Str("image", jobSpec.Image).
		Msg("Creating K8s Job")

	// Create the Job
	createdJob, err := clientset.BatchV1().Jobs(cicdNamespace).Create(ctx, k8sJob, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create k8s job: %w", err)
	}

	// Watch the Job for completion
	timeout := time.Duration(jobSpec.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	watchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = watchJobCompletion(watchCtx, clientset, createdJob.Name, jobRun)
	if err != nil {
		// Try to cleanup the job
		deletePolicy := metav1.DeletePropagationBackground
		clientset.BatchV1().Jobs(cicdNamespace).Delete(ctx, createdJob.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})
		return err
	}

	// Mark as succeeded
	finishedAt := time.Now()
	jobRun.FinishedAt = &finishedAt
	if jobRun.StartedAt != nil {
		jobRun.Duration = finishedAt.Sub(*jobRun.StartedAt).Milliseconds()
	}
	jobRun.Status = RunStatusSucceeded

	// Cleanup job (leave pod for log viewing)
	deletePolicy := metav1.DeletePropagationOrphan
	clientset.BatchV1().Jobs(cicdNamespace).Delete(ctx, createdJob.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	return nil
}

// buildK8sJob creates a K8s Job spec from a pipeline job
func buildK8sJob(pipeline *Pipeline, run *PipelineRun, jobSpec *JobSpec) *batchv1.Job {
	jobName := fmt.Sprintf("cicd-%s-%s", run.ID[:12], sanitizeName(jobSpec.Name))

	// Build environment variables
	envVars := []corev1.EnvVar{
		{Name: "PIPELINE_ID", Value: pipeline.ID},
		{Name: "PIPELINE_NAME", Value: pipeline.Name},
		{Name: "RUN_ID", Value: run.ID},
		{Name: "RUN_NUMBER", Value: fmt.Sprintf("%d", run.RunNumber)},
		{Name: "JOB_NAME", Value: jobSpec.Name},
		{Name: "TRIGGER_TYPE", Value: run.TriggerType},
	}

	// Add pipeline variables
	for k, v := range run.Variables {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	// Add job-specific env vars
	for _, ev := range jobSpec.Env {
		envVars = append(envVars, corev1.EnvVar{Name: ev.Name, Value: ev.Value})
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{}
	if jobSpec.Resources.Limits.Memory != "" || jobSpec.Resources.Limits.CPU != "" {
		resources.Limits = corev1.ResourceList{}
		if jobSpec.Resources.Limits.Memory != "" {
			resources.Limits[corev1.ResourceMemory] = resource.MustParse(jobSpec.Resources.Limits.Memory)
		}
		if jobSpec.Resources.Limits.CPU != "" {
			resources.Limits[corev1.ResourceCPU] = resource.MustParse(jobSpec.Resources.Limits.CPU)
		}
	}
	if jobSpec.Resources.Requests.Memory != "" || jobSpec.Resources.Requests.CPU != "" {
		resources.Requests = corev1.ResourceList{}
		if jobSpec.Resources.Requests.Memory != "" {
			resources.Requests[corev1.ResourceMemory] = resource.MustParse(jobSpec.Resources.Requests.Memory)
		}
		if jobSpec.Resources.Requests.CPU != "" {
			resources.Requests[corev1.ResourceCPU] = resource.MustParse(jobSpec.Resources.Requests.CPU)
		}
	}

	// Build workdir
	workdir := jobSpec.Workdir
	if workdir == "" {
		workdir = "/workspace"
	}

	// Create the script as a command
	script := jobSpec.Script

	backoffLimit := int32(0)
	ttlSeconds := int32(3600) // Keep completed jobs for 1 hour

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cicdNamespace,
			Labels: map[string]string{
				"app":               "gagos-cicd",
				"gagos.io/pipeline": pipeline.ID,
				"gagos.io/run":      run.ID,
				"gagos.io/job":      jobSpec.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":               "gagos-cicd",
						"gagos.io/pipeline": pipeline.ID,
						"gagos.io/run":      run.ID,
						"gagos.io/job":      jobSpec.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:       "runner",
							Image:      jobSpec.Image,
							Command:    []string{"/bin/sh", "-c"},
							Args:       []string{script},
							Env:        envVars,
							WorkingDir: workdir,
							Resources:  resources,
						},
					},
				},
			},
		},
	}

	// Add volume mounts for secrets if specified
	if len(jobSpec.Secrets) > 0 {
		volumes := []corev1.Volume{}
		volumeMounts := []corev1.VolumeMount{}

		for i, secret := range jobSpec.Secrets {
			volName := fmt.Sprintf("secret-%d", i)
			volumes = append(volumes, corev1.Volume{
				Name: volName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secret.Name,
						Items: []corev1.KeyToPath{
							{Key: secret.Key, Path: "secret"},
						},
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volName,
				MountPath: secret.MountPath,
				SubPath:   "secret",
				ReadOnly:  true,
			})
		}

		job.Spec.Template.Spec.Volumes = volumes
		job.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	}

	// Handle privileged containers (for Docker-in-Docker)
	if jobSpec.Privileged {
		privileged := true
		job.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged: &privileged,
		}
	}

	return job
}

// watchJobCompletion watches a K8s Job until completion
func watchJobCompletion(ctx context.Context, clientset *kubernetes.Clientset, jobName string, jobRun *JobRun) error {
	// First, get the pod name
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for job completion")
		default:
		}

		pods, err := clientset.CoreV1().Pods(cicdNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		})
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		if len(pods.Items) > 0 {
			jobRun.K8sPodName = pods.Items[0].Name
			break
		}

		time.Sleep(time.Second)
	}

	// Watch the job
	watcher, err := clientset.BatchV1().Jobs(cicdNamespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", jobName),
	})
	if err != nil {
		return fmt.Errorf("failed to watch job: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for job completion")
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("job watch channel closed")
			}

			if event.Type == watch.Modified || event.Type == watch.Added {
				job, ok := event.Object.(*batchv1.Job)
				if !ok {
					continue
				}

				// Check for completion
				for _, condition := range job.Status.Conditions {
					if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
						jobRun.ExitCode = 0
						return nil
					}
					if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
						jobRun.ExitCode = 1
						return fmt.Errorf("job failed: %s", condition.Message)
					}
				}
			}
		}
	}
}

// CancelRun cancels a running pipeline
func CancelRun(ctx context.Context, runID string) error {
	run, err := GetRun(runID)
	if err != nil {
		return err
	}

	if run.Status != RunStatusRunning && run.Status != RunStatusPending {
		return fmt.Errorf("run is not active")
	}

	clientset := k8s.GetClient()
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	// Delete all jobs for this run
	deletePolicy := metav1.DeletePropagationBackground
	err = clientset.BatchV1().Jobs(cicdNamespace).DeleteCollection(ctx, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("gagos.io/run=%s", runID),
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to delete jobs")
	}

	// Update run status
	now := time.Now()
	run.Status = RunStatusCancelled
	run.FinishedAt = &now
	if run.StartedAt != nil {
		run.Duration = now.Sub(*run.StartedAt).Milliseconds()
	}

	// Mark running jobs as cancelled
	for i := range run.Jobs {
		if run.Jobs[i].Status == RunStatusRunning || run.Jobs[i].Status == RunStatusPending {
			run.Jobs[i].Status = RunStatusCancelled
		}
	}

	if err := saveRun(run); err != nil {
		return err
	}

	// Send cancelled notification
	pipelineName := run.PipelineID
	if pipeline, err := GetPipeline(run.PipelineID); err == nil {
		pipelineName = pipeline.Name
	}
	NotifyPipelineRunEvent(NotificationEventRunCancelled, run, pipelineName)

	return nil
}

// Helper functions

func sanitizeName(name string) string {
	// K8s names must be lowercase and alphanumeric with dashes
	name = strings.ToLower(name)
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, byte(c))
		} else if c == '-' || c == '_' {
			result = append(result, '-')
		}
	}
	// Limit length
	if len(result) > 20 {
		result = result[:20]
	}
	return string(result)
}

func saveRun(run *PipelineRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	return storage.SaveRun(run.ID, data)
}

func savePipeline(pipeline *Pipeline) error {
	data, err := json.Marshal(pipeline)
	if err != nil {
		return err
	}
	return storage.SavePipeline(pipeline.ID, data)
}

// GetRun retrieves a run by ID
func GetRun(id string) (*PipelineRun, error) {
	data, err := storage.GetRun(id)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("run not found: %s", id)
	}

	var run PipelineRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

// GetPipeline retrieves a pipeline by ID
func GetPipeline(id string) (*Pipeline, error) {
	data, err := storage.GetPipeline(id)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}

	var pipeline Pipeline
	if err := json.Unmarshal(data, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// ListPipelines returns all pipelines
func ListPipelines() ([]*Pipeline, error) {
	items, err := storage.ListPipelines()
	if err != nil {
		return nil, err
	}

	pipelines := make([]*Pipeline, 0, len(items))
	for _, data := range items {
		var p Pipeline
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		pipelines = append(pipelines, &p)
	}
	return pipelines, nil
}

// ListRuns returns all runs, optionally filtered by pipeline ID
func ListRuns(pipelineID string, limit int) ([]*PipelineRun, error) {
	items, err := storage.ListRuns()
	if err != nil {
		return nil, err
	}

	runs := make([]*PipelineRun, 0)
	for _, data := range items {
		var r PipelineRun
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		if pipelineID != "" && r.PipelineID != pipelineID {
			continue
		}
		runs = append(runs, &r)
	}

	// Sort by created_at descending (newest first)
	for i := 0; i < len(runs)-1; i++ {
		for j := i + 1; j < len(runs); j++ {
			if runs[j].CreatedAt.After(runs[i].CreatedAt) {
				runs[i], runs[j] = runs[j], runs[i]
			}
		}
	}

	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}

	return runs, nil
}

// SavePipeline saves a pipeline to storage
func SavePipeline(pipeline *Pipeline) error {
	return savePipeline(pipeline)
}

// DeletePipeline removes a pipeline
func DeletePipeline(id string) error {
	return storage.DeletePipeline(id)
}

// DeleteRun removes a run
func DeleteRun(id string) error {
	return storage.DeleteRun(id)
}

// GetStats returns CI/CD statistics
func GetStats() (*CICDStats, error) {
	pipelines, err := ListPipelines()
	if err != nil {
		return nil, err
	}

	runs, err := ListRuns("", 0)
	if err != nil {
		return nil, err
	}

	stats := &CICDStats{
		TotalPipelines: len(pipelines),
		TotalRuns:      len(runs),
	}

	// Count last 24 hours
	dayAgo := time.Now().Add(-24 * time.Hour)
	for _, run := range runs {
		if run.Status == RunStatusRunning {
			stats.RunningRuns++
		}
		if run.CreatedAt.After(dayAgo) {
			if run.Status == RunStatusSucceeded {
				stats.Succeeded24h++
			} else if run.Status == RunStatusFailed {
				stats.Failed24h++
			}
		}
	}

	return stats, nil
}
