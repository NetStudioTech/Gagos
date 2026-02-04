package cicd

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

var (
	scheduler     *Scheduler
	schedulerOnce sync.Once
)

// Scheduler manages cron-based pipeline and freestyle job triggers
type Scheduler struct {
	cron           *cron.Cron
	jobs           map[string]cron.EntryID // pipelineID -> entryID
	freestyleJobs  map[string]cron.EntryID // freestyleJobID -> entryID
	mu             sync.RWMutex
	stopChan       chan struct{}
}

// InitScheduler initializes the global scheduler
func InitScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		scheduler = &Scheduler{
			cron:          cron.New(cron.WithSeconds()),
			jobs:          make(map[string]cron.EntryID),
			freestyleJobs: make(map[string]cron.EntryID),
			stopChan:      make(chan struct{}),
		}
	})
	return scheduler
}

// GetScheduler returns the global scheduler instance
func GetScheduler() *Scheduler {
	return scheduler
}

// Start starts the scheduler and loads all pipelines and freestyle jobs
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting CI/CD scheduler")

	// Load all pipelines with cron triggers
	if err := s.RefreshPipelines(); err != nil {
		log.Warn().Err(err).Msg("Failed to load pipelines for scheduling")
	}

	// Load all freestyle jobs with cron triggers
	if err := s.RefreshFreestyleJobs(); err != nil {
		log.Warn().Err(err).Msg("Failed to load freestyle jobs for scheduling")
	}

	// Start cleanup scheduler
	if err := s.StartCleanupScheduler(); err != nil {
		log.Warn().Err(err).Msg("Failed to start cleanup scheduler")
	}

	// Start the cron scheduler
	s.cron.Start()

	log.Info().
		Int("pipelines", len(s.jobs)).
		Int("freestyle_jobs", len(s.freestyleJobs)).
		Msg("CI/CD scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping CI/CD scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	close(s.stopChan)
}

// RefreshPipelines reloads all pipeline schedules
func (s *Scheduler) RefreshPipelines() error {
	pipelines, err := ListPipelines()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove all existing jobs
	for _, entryID := range s.jobs {
		s.cron.Remove(entryID)
	}
	s.jobs = make(map[string]cron.EntryID)

	// Re-register all pipelines
	for _, p := range pipelines {
		s.registerPipelineUnsafe(p)
	}

	return nil
}

// RegisterPipeline registers a pipeline's cron triggers
func (s *Scheduler) RegisterPipeline(p *Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.jobs[p.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, p.ID)
	}

	return s.registerPipelineUnsafe(p)
}

// registerPipelineUnsafe registers without locking (caller must hold lock)
func (s *Scheduler) registerPipelineUnsafe(p *Pipeline) error {
	for _, trigger := range p.Spec.Triggers {
		if trigger.Type != "cron" || !trigger.Enabled || trigger.Schedule == "" {
			continue
		}

		schedule := trigger.Schedule

		// Create a closure for the trigger
		pipelineID := p.ID
		pipelineName := p.Name

		entryID, err := s.cron.AddFunc(schedule, func() {
			s.triggerPipeline(pipelineID, pipelineName, schedule)
		})

		if err != nil {
			log.Warn().
				Err(err).
				Str("pipeline", p.Name).
				Str("schedule", schedule).
				Msg("Failed to register cron schedule")
			continue
		}

		s.jobs[p.ID] = entryID
		log.Info().
			Str("pipeline", p.Name).
			Str("schedule", schedule).
			Msg("Registered cron trigger")
	}

	return nil
}

// UnregisterPipeline removes a pipeline's cron triggers
func (s *Scheduler) UnregisterPipeline(pipelineID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[pipelineID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, pipelineID)
		log.Info().Str("pipeline_id", pipelineID).Msg("Unregistered cron trigger")
	}
}

// triggerPipeline is called when a cron schedule fires
func (s *Scheduler) triggerPipeline(pipelineID, pipelineName, schedule string) {
	log.Info().
		Str("pipeline_id", pipelineID).
		Str("pipeline", pipelineName).
		Str("schedule", schedule).
		Msg("Cron trigger fired")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the pipeline
	pipeline, err := GetPipeline(pipelineID)
	if err != nil {
		log.Error().Err(err).Str("pipeline_id", pipelineID).Msg("Failed to get pipeline for cron trigger")
		return
	}

	// Trigger the pipeline
	triggerRef := "cron:" + schedule + "@" + time.Now().Format(time.RFC3339)
	run, err := TriggerPipeline(ctx, pipeline, "cron", triggerRef, nil)
	if err != nil {
		log.Error().Err(err).Str("pipeline", pipelineName).Msg("Failed to trigger pipeline from cron")
		return
	}

	log.Info().
		Str("pipeline", pipelineName).
		Str("run_id", run.ID).
		Int("run_number", run.RunNumber).
		Msg("Pipeline triggered by cron")
}

// GetScheduledJobs returns info about scheduled jobs
func (s *Scheduler) GetScheduledJobs() []ScheduledJobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []ScheduledJobInfo
	for pipelineID, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		jobs = append(jobs, ScheduledJobInfo{
			PipelineID: pipelineID,
			NextRun:    entry.Next,
			PrevRun:    entry.Prev,
		})
	}
	return jobs
}

// ScheduledJobInfo contains information about a scheduled job
type ScheduledJobInfo struct {
	PipelineID string    `json:"pipeline_id"`
	NextRun    time.Time `json:"next_run"`
	PrevRun    time.Time `json:"prev_run"`
}

// RefreshFreestyleJobs reloads all freestyle job schedules
func (s *Scheduler) RefreshFreestyleJobs() error {
	jobs, err := ListFreestyleJobs()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove all existing freestyle jobs
	for _, entryID := range s.freestyleJobs {
		s.cron.Remove(entryID)
	}
	s.freestyleJobs = make(map[string]cron.EntryID)

	// Re-register all freestyle jobs
	for _, j := range jobs {
		s.registerFreestyleJobUnsafe(j)
	}

	return nil
}

// RegisterFreestyleJob registers a freestyle job's cron triggers
func (s *Scheduler) RegisterFreestyleJob(j *FreestyleJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.freestyleJobs[j.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.freestyleJobs, j.ID)
	}

	return s.registerFreestyleJobUnsafe(j)
}

// registerFreestyleJobUnsafe registers without locking (caller must hold lock)
func (s *Scheduler) registerFreestyleJobUnsafe(j *FreestyleJob) error {
	if !j.Enabled {
		return nil
	}

	for _, trigger := range j.Triggers {
		if trigger.Type != "cron" || !trigger.Enabled || trigger.Schedule == "" {
			continue
		}

		schedule := trigger.Schedule

		// Create a closure for the trigger
		jobID := j.ID
		jobName := j.Name

		entryID, err := s.cron.AddFunc(schedule, func() {
			s.triggerFreestyleJob(jobID, jobName, schedule)
		})

		if err != nil {
			log.Warn().
				Err(err).
				Str("job", j.Name).
				Str("schedule", schedule).
				Msg("Failed to register freestyle job cron schedule")
			continue
		}

		s.freestyleJobs[j.ID] = entryID
		log.Info().
			Str("job", j.Name).
			Str("schedule", schedule).
			Msg("Registered freestyle job cron trigger")
	}

	return nil
}

// UnregisterFreestyleJob removes a freestyle job's cron triggers
func (s *Scheduler) UnregisterFreestyleJob(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.freestyleJobs[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.freestyleJobs, jobID)
		log.Info().Str("job_id", jobID).Msg("Unregistered freestyle job cron trigger")
	}
}

// triggerFreestyleJob is called when a cron schedule fires for freestyle job
func (s *Scheduler) triggerFreestyleJob(jobID, jobName, schedule string) {
	log.Info().
		Str("job_id", jobID).
		Str("job", jobName).
		Str("schedule", schedule).
		Msg("Freestyle job cron trigger fired")

	// Trigger the freestyle job
	triggerRef := "cron:" + schedule + "@" + time.Now().Format(time.RFC3339)
	build, err := TriggerFreestyleBuild(jobID, "cron", triggerRef, nil)
	if err != nil {
		log.Error().Err(err).Str("job", jobName).Msg("Failed to trigger freestyle job from cron")
		return
	}

	log.Info().
		Str("job", jobName).
		Str("build_id", build.ID).
		Int("build_number", build.BuildNumber).
		Msg("Freestyle job triggered by cron")
}

// GetScheduledFreestyleJobs returns info about scheduled freestyle jobs
func (s *Scheduler) GetScheduledFreestyleJobs() []ScheduledFreestyleJobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []ScheduledFreestyleJobInfo
	for jobID, entryID := range s.freestyleJobs {
		entry := s.cron.Entry(entryID)
		jobs = append(jobs, ScheduledFreestyleJobInfo{
			JobID:   jobID,
			NextRun: entry.Next,
			PrevRun: entry.Prev,
		})
	}
	return jobs
}

// ScheduledFreestyleJobInfo contains information about a scheduled freestyle job
type ScheduledFreestyleJobInfo struct {
	JobID   string    `json:"job_id"`
	NextRun time.Time `json:"next_run"`
	PrevRun time.Time `json:"prev_run"`
}

// Retention policy settings
const (
	DefaultFreestyleBuildRetention = 50          // Keep last 50 builds per job
	DefaultPipelineRunRetention    = 100         // Keep last 100 runs per pipeline
	DefaultMaxRetentionDays        = 30          // Maximum age in days
	CleanupSchedule                = "0 0 3 * * *" // Run cleanup at 3 AM daily
)

// RetentionConfig holds retention policy settings
type RetentionConfig struct {
	FreestyleBuildsPerJob int `json:"freestyle_builds_per_job"`
	PipelineRunsPerPipeline int `json:"pipeline_runs_per_pipeline"`
	MaxRetentionDays      int `json:"max_retention_days"`
}

var (
	cleanupEntryID cron.EntryID
	retentionConfig = RetentionConfig{
		FreestyleBuildsPerJob:   DefaultFreestyleBuildRetention,
		PipelineRunsPerPipeline: DefaultPipelineRunRetention,
		MaxRetentionDays:        DefaultMaxRetentionDays,
	}
)

// StartCleanupScheduler registers the cleanup job with the scheduler
func (s *Scheduler) StartCleanupScheduler() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing cleanup job if any
	if cleanupEntryID != 0 {
		s.cron.Remove(cleanupEntryID)
	}

	entryID, err := s.cron.AddFunc(CleanupSchedule, func() {
		s.RunCleanup()
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to register cleanup scheduler")
		return err
	}

	cleanupEntryID = entryID
	log.Info().Str("schedule", CleanupSchedule).Msg("Cleanup scheduler registered")
	return nil
}

// RunCleanup performs cleanup of old builds and runs
func (s *Scheduler) RunCleanup() {
	log.Info().Msg("Starting scheduled cleanup")
	startTime := time.Now()

	var totalDeleted int

	// Cleanup freestyle builds
	freestyleDeleted, err := s.cleanupFreestyleBuilds()
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup freestyle builds")
	} else {
		totalDeleted += freestyleDeleted
	}

	// Cleanup pipeline runs
	pipelineDeleted, err := s.cleanupPipelineRuns()
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup pipeline runs")
	} else {
		totalDeleted += pipelineDeleted
	}

	duration := time.Since(startTime)
	log.Info().
		Int("total_deleted", totalDeleted).
		Int("freestyle_builds", freestyleDeleted).
		Int("pipeline_runs", pipelineDeleted).
		Dur("duration", duration).
		Msg("Cleanup completed")
}

// cleanupFreestyleBuilds removes old freestyle builds based on retention policy
func (s *Scheduler) cleanupFreestyleBuilds() (int, error) {
	jobs, err := ListFreestyleJobs()
	if err != nil {
		return 0, err
	}

	var totalDeleted int
	cutoffTime := time.Now().AddDate(0, 0, -retentionConfig.MaxRetentionDays)

	for _, job := range jobs {
		builds, err := ListFreestyleBuildsForJob(job.ID)
		if err != nil {
			log.Warn().Err(err).Str("job_id", job.ID).Msg("Failed to list builds for cleanup")
			continue
		}

		// Builds are already sorted newest first
		for i, build := range builds {
			// Keep if within count limit
			if i < retentionConfig.FreestyleBuildsPerJob {
				continue
			}

			// Skip running builds
			if build.Status == RunStatusRunning {
				continue
			}

			// Delete if beyond count limit
			if err := DeleteFreestyleBuild(build.ID); err != nil {
				log.Warn().Err(err).Str("build_id", build.ID).Msg("Failed to delete old build")
			} else {
				totalDeleted++
			}
		}

		// Also delete builds older than max retention days (even if within count)
		for _, build := range builds {
			if build.Status == RunStatusRunning {
				continue
			}
			if build.CreatedAt.Before(cutoffTime) {
				if err := DeleteFreestyleBuild(build.ID); err != nil {
					// May already be deleted, ignore error
					continue
				}
				totalDeleted++
			}
		}
	}

	if totalDeleted > 0 {
		log.Info().Int("deleted", totalDeleted).Msg("Cleaned up old freestyle builds")
	}
	return totalDeleted, nil
}

// cleanupPipelineRuns removes old pipeline runs based on retention policy
func (s *Scheduler) cleanupPipelineRuns() (int, error) {
	pipelines, err := ListPipelines()
	if err != nil {
		return 0, err
	}

	var totalDeleted int
	cutoffTime := time.Now().AddDate(0, 0, -retentionConfig.MaxRetentionDays)

	for _, pipeline := range pipelines {
		runs, err := ListRuns(pipeline.ID, 0)
		if err != nil {
			log.Warn().Err(err).Str("pipeline_id", pipeline.ID).Msg("Failed to list runs for cleanup")
			continue
		}

		// Delete runs beyond retention count
		for i, run := range runs {
			// Keep if within count limit
			if i < retentionConfig.PipelineRunsPerPipeline {
				continue
			}

			// Skip running
			if run.Status == RunStatusRunning {
				continue
			}

			// Delete if beyond count limit
			if err := DeleteRun(run.ID); err != nil {
				log.Warn().Err(err).Str("run_id", run.ID).Msg("Failed to delete old pipeline run")
			} else {
				totalDeleted++
			}
		}

		// Also delete runs older than max retention days
		for _, run := range runs {
			if run.Status == RunStatusRunning {
				continue
			}
			if run.StartedAt != nil && run.StartedAt.Before(cutoffTime) {
				if err := DeleteRun(run.ID); err != nil {
					continue
				}
				totalDeleted++
			}
		}
	}

	if totalDeleted > 0 {
		log.Info().Int("deleted", totalDeleted).Msg("Cleaned up old pipeline runs")
	}
	return totalDeleted, nil
}

// SetRetentionConfig updates the retention policy settings
func SetRetentionConfig(config RetentionConfig) {
	if config.FreestyleBuildsPerJob > 0 {
		retentionConfig.FreestyleBuildsPerJob = config.FreestyleBuildsPerJob
	}
	if config.PipelineRunsPerPipeline > 0 {
		retentionConfig.PipelineRunsPerPipeline = config.PipelineRunsPerPipeline
	}
	if config.MaxRetentionDays > 0 {
		retentionConfig.MaxRetentionDays = config.MaxRetentionDays
	}
	log.Info().
		Int("freestyle_builds", retentionConfig.FreestyleBuildsPerJob).
		Int("pipeline_runs", retentionConfig.PipelineRunsPerPipeline).
		Int("max_days", retentionConfig.MaxRetentionDays).
		Msg("Retention config updated")
}

// GetRetentionConfig returns the current retention policy settings
func GetRetentionConfig() RetentionConfig {
	return retentionConfig
}

// RunCleanupNow triggers an immediate cleanup (for manual invocation)
func (s *Scheduler) RunCleanupNow() {
	go s.RunCleanup()
}
