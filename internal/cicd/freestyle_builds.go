package cicd

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gaga951/gagos/internal/storage"

	"github.com/rs/zerolog/log"
)

var (
	// runningBuilds tracks currently running freestyle builds for cancellation
	runningBuilds     = make(map[string]chan struct{})
	runningBuildsMu   sync.RWMutex

	// buildOutputs stores build output for streaming
	buildOutputs   = make(map[string]*BuildOutputStream)
	buildOutputsMu sync.RWMutex
)

// BuildOutputStream handles streaming output for a build
type BuildOutputStream struct {
	mu         sync.RWMutex
	output     []byte
	listeners  []chan []byte
	closed     bool
}

// NewBuildOutputStream creates a new output stream
func NewBuildOutputStream() *BuildOutputStream {
	return &BuildOutputStream{
		output:    make([]byte, 0, 4096),
		listeners: make([]chan []byte, 0),
	}
}

// Write implements io.Writer for streaming output
func (s *BuildOutputStream) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, fmt.Errorf("stream closed")
	}

	s.output = append(s.output, p...)

	// Notify all listeners
	for _, ch := range s.listeners {
		select {
		case ch <- p:
		default:
			// Skip if listener buffer is full
		}
	}

	return len(p), nil
}

// Subscribe returns a channel that receives new output
func (s *BuildOutputStream) Subscribe() chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan []byte, 100)
	s.listeners = append(s.listeners, ch)

	// Send existing output
	if len(s.output) > 0 {
		ch <- s.output
	}

	return ch
}

// Unsubscribe removes a listener
func (s *BuildOutputStream) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, listener := range s.listeners {
		if listener == ch {
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// Close closes the stream
func (s *BuildOutputStream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	for _, ch := range s.listeners {
		close(ch)
	}
	s.listeners = nil
}

// GetOutput returns all accumulated output
func (s *BuildOutputStream) GetOutput() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]byte{}, s.output...)
}

// generateFreestyleBuildID generates a unique ID for a freestyle build
func generateFreestyleBuildID() string {
	return generateID("fsb")
}

// CreateFreestyleBuild creates a new build for a job
func CreateFreestyleBuild(jobID string, triggerType string, triggerRef string, params map[string]string) (*FreestyleBuild, error) {
	job, err := GetFreestyleJob(jobID)
	if err != nil {
		return nil, err
	}

	if !job.Enabled {
		return nil, fmt.Errorf("job is disabled")
	}

	buildNum, err := GetNextBuildNumber(jobID)
	if err != nil {
		buildNum = 1
	}

	// Merge job environment with provided parameters
	env := make(map[string]string)
	for k, v := range job.Environment {
		env[k] = v
	}

	// Apply default values for missing parameters
	for _, p := range job.Parameters {
		if _, ok := params[p.Name]; !ok && p.DefaultValue != "" {
			params[p.Name] = p.DefaultValue
		}
	}

	// Validate required parameters
	for _, p := range job.Parameters {
		if p.Required {
			if _, ok := params[p.Name]; !ok {
				return nil, fmt.Errorf("required parameter missing: %s", p.Name)
			}
		}
	}

	// Initialize build steps from job
	steps := make([]FreestyleBuildStep, len(job.BuildSteps))
	for i, s := range job.BuildSteps {
		// Get host name for display
		hostName := ""
		if host, err := GetSSHHost(s.HostID); err == nil {
			hostName = host.Name
		}

		steps[i] = FreestyleBuildStep{
			StepID:   s.ID,
			Name:     s.Name,
			Type:     s.Type,
			HostID:   s.HostID,
			HostName: hostName,
			Status:   RunStatusPending,
			ExitCode: -1,
		}
	}

	build := &FreestyleBuild{
		ID:          generateFreestyleBuildID(),
		JobID:       jobID,
		JobName:     job.Name,
		BuildNumber: buildNum,
		Status:      RunStatusPending,
		TriggerType: triggerType,
		TriggerRef:  triggerRef,
		Parameters:  params,
		Environment: env,
		Steps:       steps,
		CreatedAt:   time.Now(),
	}

	// Save to storage
	data, err := json.Marshal(build)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal build: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketFreestyleBuilds, build.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save build: %w", err)
	}

	log.Info().
		Str("id", build.ID).
		Str("job", job.Name).
		Int("number", buildNum).
		Msg("Freestyle build created")

	return build, nil
}

// GetFreestyleBuild retrieves a build by ID
func GetFreestyleBuild(id string) (*FreestyleBuild, error) {
	data, err := storage.GetBackend().Get(storage.BucketFreestyleBuilds, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("build not found: %s", id)
	}

	var build FreestyleBuild
	if err := json.Unmarshal(data, &build); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build: %w", err)
	}

	return &build, nil
}

// ListFreestyleBuilds returns all builds
func ListFreestyleBuilds() ([]*FreestyleBuild, error) {
	dataList, err := storage.GetBackend().List(storage.BucketFreestyleBuilds)
	if err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}

	builds := make([]*FreestyleBuild, 0, len(dataList))
	for _, data := range dataList {
		var build FreestyleBuild
		if err := json.Unmarshal(data, &build); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal freestyle build")
			continue
		}
		builds = append(builds, &build)
	}

	// Sort by created time, newest first
	sort.Slice(builds, func(i, j int) bool {
		return builds[i].CreatedAt.After(builds[j].CreatedAt)
	})

	return builds, nil
}

// ListFreestyleBuildsForJob returns all builds for a specific job
func ListFreestyleBuildsForJob(jobID string) ([]*FreestyleBuild, error) {
	all, err := ListFreestyleBuilds()
	if err != nil {
		return nil, err
	}

	builds := make([]*FreestyleBuild, 0)
	for _, b := range all {
		if b.JobID == jobID {
			builds = append(builds, b)
		}
	}

	return builds, nil
}

// UpdateFreestyleBuild saves build changes
func UpdateFreestyleBuild(build *FreestyleBuild) error {
	data, err := json.Marshal(build)
	if err != nil {
		return fmt.Errorf("failed to marshal build: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketFreestyleBuilds, build.ID, data); err != nil {
		return fmt.Errorf("failed to save build: %w", err)
	}

	return nil
}

// StartFreestyleBuild marks a build as running
func StartFreestyleBuild(buildID string) error {
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		return err
	}

	now := time.Now()
	build.Status = RunStatusRunning
	build.StartedAt = &now

	// Create output stream
	buildOutputsMu.Lock()
	buildOutputs[buildID] = NewBuildOutputStream()
	buildOutputsMu.Unlock()

	// Create cancellation channel
	runningBuildsMu.Lock()
	runningBuilds[buildID] = make(chan struct{})
	runningBuildsMu.Unlock()

	// Send build started notification
	NotifyBuildEvent(NotificationEventBuildStarted, build)

	return UpdateFreestyleBuild(build)
}

// CompleteFreestyleBuild marks a build as complete
func CompleteFreestyleBuild(buildID string, status RunStatus, errMsg string) error {
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		return err
	}

	now := time.Now()
	build.Status = status
	build.FinishedAt = &now
	build.Error = errMsg

	if build.StartedAt != nil {
		build.Duration = now.Sub(*build.StartedAt).Milliseconds()
	}

	// Close output stream
	buildOutputsMu.Lock()
	if stream, ok := buildOutputs[buildID]; ok {
		stream.Close()
		delete(buildOutputs, buildID)
	}
	buildOutputsMu.Unlock()

	// Remove cancellation channel
	runningBuildsMu.Lock()
	delete(runningBuilds, buildID)
	runningBuildsMu.Unlock()

	// Update job status
	UpdateFreestyleJobStatus(build.JobID, buildID, string(status))

	// Send notification based on status
	var event NotificationEvent
	switch status {
	case RunStatusSucceeded:
		event = NotificationEventBuildSucceeded
	case RunStatusFailed:
		event = NotificationEventBuildFailed
	case RunStatusCancelled:
		event = NotificationEventBuildCancelled
	}
	if event != "" {
		NotifyBuildEvent(event, build)
	}

	return UpdateFreestyleBuild(build)
}

// UpdateFreestyleBuildStep updates a single step's status
func UpdateFreestyleBuildStep(buildID string, stepID string, status RunStatus, exitCode int, output string, errMsg string) error {
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range build.Steps {
		if build.Steps[i].StepID == stepID {
			if build.Steps[i].Status == RunStatusPending && status == RunStatusRunning {
				build.Steps[i].StartedAt = &now
			}
			build.Steps[i].Status = status
			build.Steps[i].ExitCode = exitCode
			build.Steps[i].Output = output
			build.Steps[i].Error = errMsg

			if status == RunStatusSucceeded || status == RunStatusFailed || status == RunStatusCancelled {
				build.Steps[i].FinishedAt = &now
				if build.Steps[i].StartedAt != nil {
					build.Steps[i].Duration = now.Sub(*build.Steps[i].StartedAt).Milliseconds()
				}
			}
			break
		}
	}

	return UpdateFreestyleBuild(build)
}

// CancelFreestyleBuild cancels a running build
func CancelFreestyleBuild(buildID string) error {
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		return err
	}

	if build.Status != RunStatusRunning && build.Status != RunStatusPending {
		return fmt.Errorf("build is not running or pending")
	}

	// Signal cancellation
	runningBuildsMu.RLock()
	if cancelCh, ok := runningBuilds[buildID]; ok {
		close(cancelCh)
	}
	runningBuildsMu.RUnlock()

	return CompleteFreestyleBuild(buildID, RunStatusCancelled, "Build cancelled by user")
}

// GetBuildCancelChannel returns the cancellation channel for a build
func GetBuildCancelChannel(buildID string) <-chan struct{} {
	runningBuildsMu.RLock()
	defer runningBuildsMu.RUnlock()
	return runningBuilds[buildID]
}

// GetBuildOutputStream returns the output stream for a build
func GetBuildOutputStream(buildID string) *BuildOutputStream {
	buildOutputsMu.RLock()
	defer buildOutputsMu.RUnlock()
	return buildOutputs[buildID]
}

// WriteBuildOutput writes to the build's output stream
func WriteBuildOutput(buildID string, data []byte) {
	buildOutputsMu.RLock()
	stream := buildOutputs[buildID]
	buildOutputsMu.RUnlock()

	if stream != nil {
		stream.Write(data)
	}
}

// GetBuildLogs returns the accumulated logs for a build
func GetBuildLogs(buildID string) (string, error) {
	// First check if there's an active stream
	buildOutputsMu.RLock()
	stream := buildOutputs[buildID]
	buildOutputsMu.RUnlock()

	if stream != nil {
		return string(stream.GetOutput()), nil
	}

	// Otherwise, concatenate step outputs
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		return "", err
	}

	var logs string
	for _, step := range build.Steps {
		if step.Output != "" {
			logs += fmt.Sprintf("=== Step: %s ===\n%s\n", step.Name, step.Output)
		}
		if step.Error != "" {
			logs += fmt.Sprintf("ERROR: %s\n", step.Error)
		}
	}

	return logs, nil
}

// DeleteFreestyleBuild deletes a build
func DeleteFreestyleBuild(id string) error {
	build, err := GetFreestyleBuild(id)
	if err != nil {
		return err
	}

	if build.Status == RunStatusRunning {
		return fmt.Errorf("cannot delete running build")
	}

	if err := storage.GetBackend().Delete(storage.BucketFreestyleBuilds, id); err != nil {
		return fmt.Errorf("failed to delete build: %w", err)
	}

	log.Info().Str("id", id).Msg("Freestyle build deleted")
	return nil
}

// CleanupOldBuilds removes builds older than the specified duration
func CleanupOldBuilds(jobID string, keepCount int) error {
	builds, err := ListFreestyleBuildsForJob(jobID)
	if err != nil {
		return err
	}

	if len(builds) <= keepCount {
		return nil
	}

	// Builds are already sorted newest first
	for _, build := range builds[keepCount:] {
		if build.Status != RunStatusRunning {
			DeleteFreestyleBuild(build.ID)
		}
	}

	return nil
}
