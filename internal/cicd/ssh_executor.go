package cicd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ExecuteFreestyleBuild executes a freestyle build
func ExecuteFreestyleBuild(buildID string) {
	build, err := GetFreestyleBuild(buildID)
	if err != nil {
		log.Error().Err(err).Str("build", buildID).Msg("Failed to get build")
		return
	}

	job, err := GetFreestyleJob(build.JobID)
	if err != nil {
		log.Error().Err(err).Str("job", build.JobID).Msg("Failed to get job")
		CompleteFreestyleBuild(buildID, RunStatusFailed, err.Error())
		return
	}

	// Mark build as started
	if err := StartFreestyleBuild(buildID); err != nil {
		log.Error().Err(err).Str("build", buildID).Msg("Failed to start build")
		return
	}

	// Get cancellation channel
	cancelCh := GetBuildCancelChannel(buildID)

	log.Info().
		Str("build", buildID).
		Str("job", job.Name).
		Int("steps", len(job.BuildSteps)).
		Msg("Starting freestyle build execution")

	// Write header to output
	WriteBuildOutput(buildID, []byte(fmt.Sprintf("=== Build #%d for %s ===\n", build.BuildNumber, job.Name)))
	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Started at: %s\n", time.Now().Format(time.RFC3339))))
	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Trigger: %s\n\n", build.TriggerType)))

	// Execute SCM checkout if configured
	var scmResult *GitCloneResult
	if job.SCM != nil && job.SCM.Type == "git" && len(job.BuildSteps) > 0 {
		// Use the first build step's host for SCM checkout
		firstHost, err := GetSSHHost(job.BuildSteps[0].HostID)
		if err != nil {
			WriteBuildOutput(buildID, []byte(fmt.Sprintf("SCM Error: Failed to get SSH host: %s\n", err)))
			CompleteFreestyleBuild(buildID, RunStatusFailed, fmt.Sprintf("SCM checkout failed: %s", err))
			return
		}

		session, err := NewSSHSession(firstHost)
		if err != nil {
			WriteBuildOutput(buildID, []byte(fmt.Sprintf("SCM Error: Failed to connect: %s\n", err)))
			CompleteFreestyleBuild(buildID, RunStatusFailed, fmt.Sprintf("SCM checkout failed: %s", err))
			return
		}

		scmResult, err = ExecuteGitSCM(buildID, session, job, build)
		session.Close()

		if err != nil {
			WriteBuildOutput(buildID, []byte(fmt.Sprintf("SCM Error: %s\n", err)))
			CompleteFreestyleBuild(buildID, RunStatusFailed, fmt.Sprintf("SCM checkout failed: %s", err))
			return
		}

		// Add Git environment variables to build
		if scmResult != nil {
			gitEnv := SetGitEnvironmentVariables(scmResult)
			if build.Environment == nil {
				build.Environment = make(map[string]string)
			}
			for k, v := range gitEnv {
				build.Environment[k] = v
			}
		}
	}

	// Execute each step
	var buildFailed bool
	var buildError string

	for _, step := range job.BuildSteps {
		select {
		case <-cancelCh:
			WriteBuildOutput(buildID, []byte("\n!!! Build cancelled !!!\n"))
			CompleteFreestyleBuild(buildID, RunStatusCancelled, "Build cancelled")
			return
		default:
		}

		stepErr := executeStep(buildID, build, job, &step, cancelCh)
		if stepErr != nil {
			if !step.ContinueOnError {
				buildFailed = true
				buildError = stepErr.Error()
				break
			}
		}
	}

	// Complete the build
	if buildFailed {
		WriteBuildOutput(buildID, []byte(fmt.Sprintf("\n=== Build FAILED: %s ===\n", buildError)))
		CompleteFreestyleBuild(buildID, RunStatusFailed, buildError)
	} else {
		WriteBuildOutput(buildID, []byte(fmt.Sprintf("\n=== Build SUCCEEDED ===\n")))
		WriteBuildOutput(buildID, []byte(fmt.Sprintf("Finished at: %s\n", time.Now().Format(time.RFC3339))))
		CompleteFreestyleBuild(buildID, RunStatusSucceeded, "")
	}

	log.Info().
		Str("build", buildID).
		Bool("success", !buildFailed).
		Msg("Freestyle build execution completed")
}

// executeStep executes a single build step
func executeStep(buildID string, build *FreestyleBuild, job *FreestyleJob, step *BuildStep, cancelCh <-chan struct{}) error {
	WriteBuildOutput(buildID, []byte(fmt.Sprintf("\n--- Step: %s (%s) ---\n", step.Name, step.Type)))

	// Mark step as running
	UpdateFreestyleBuildStep(buildID, step.ID, RunStatusRunning, -1, "", "")

	// Check if this is a local execution (no host specified or "local")
	if step.HostID == "" || step.HostID == "local" {
		return executeLocalStep(buildID, build, job, step, cancelCh)
	}

	// Get SSH host for remote execution
	host, err := GetSSHHost(step.HostID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get SSH host: %s", err)
		WriteBuildOutput(buildID, []byte(errMsg+"\n"))
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, -1, "", errMsg)
		return err
	}

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Host: %s (%s@%s:%d)\n", host.Name, host.Username, host.Host, host.Port)))

	// Create SSH session
	session, err := NewSSHSession(host)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect: %s", err)
		WriteBuildOutput(buildID, []byte(errMsg+"\n"))
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, -1, "", errMsg)
		return err
	}
	defer session.Close()

	// Set timeout
	timeout := time.Duration(step.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle cancellation
	go func() {
		select {
		case <-cancelCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	var exitCode int
	var output string
	var stepErr error

	switch step.Type {
	case StepTypeShell:
		exitCode, output, stepErr = executeShellStep(ctx, session, step, build, job, timeout, buildID)

	case StepTypeScript:
		exitCode, output, stepErr = executeScriptStep(ctx, session, step, build, job, timeout, buildID)

	case StepTypeSCPPush:
		exitCode, output, stepErr = executeSCPPushStep(session, step, build)

	case StepTypeSCPPull:
		exitCode, output, stepErr = executeSCPPullStep(session, step, build)

	default:
		stepErr = fmt.Errorf("unsupported step type: %s", step.Type)
	}

	// Update step status
	if stepErr != nil {
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, exitCode, output, stepErr.Error())
		return stepErr
	}

	if exitCode != 0 {
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, exitCode, output, fmt.Sprintf("Exit code: %d", exitCode))
		return fmt.Errorf("step failed with exit code %d", exitCode)
	}

	UpdateFreestyleBuildStep(buildID, step.ID, RunStatusSucceeded, exitCode, output, "")
	return nil
}

// executeLocalStep executes a step locally inside the container
func executeLocalStep(buildID string, build *FreestyleBuild, job *FreestyleJob, step *BuildStep, cancelCh <-chan struct{}) error {
	WriteBuildOutput(buildID, []byte("Host: local (container)\n"))

	// Set timeout
	timeout := time.Duration(step.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Handle cancellation
	go func() {
		select {
		case <-cancelCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	var exitCode int
	var output string
	var stepErr error

	switch step.Type {
	case StepTypeShell:
		exitCode, output, stepErr = executeLocalShellStep(ctx, step, build, job, buildID)

	case StepTypeScript:
		exitCode, output, stepErr = executeLocalScriptStep(ctx, step, build, job, buildID)

	default:
		stepErr = fmt.Errorf("step type %s not supported for local execution", step.Type)
	}

	// Update step status
	if stepErr != nil {
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, exitCode, output, stepErr.Error())
		return stepErr
	}

	if exitCode != 0 {
		UpdateFreestyleBuildStep(buildID, step.ID, RunStatusFailed, exitCode, output, fmt.Sprintf("Exit code: %d", exitCode))
		return fmt.Errorf("step failed with exit code %d", exitCode)
	}

	UpdateFreestyleBuildStep(buildID, step.ID, RunStatusSucceeded, exitCode, output, "")
	return nil
}

// executeLocalShellStep executes a shell command locally
func executeLocalShellStep(ctx context.Context, step *BuildStep, build *FreestyleBuild, job *FreestyleJob, buildID string) (int, string, error) {
	cmdStr := expandVariables(step.Command, build, job)

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("$ %s\n", cmdStr)))

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range build.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	stream := GetBuildOutputStream(buildID)
	if stream != nil {
		cmd.Stdout = &streamWriter{stream: stream, buffer: &stdout}
		cmd.Stderr = &streamWriter{stream: stream, buffer: &stderr}
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return -1, output, err
		}
	}

	return exitCode, output, nil
}

// executeLocalScriptStep executes a script locally
func executeLocalScriptStep(ctx context.Context, step *BuildStep, build *FreestyleBuild, job *FreestyleJob, buildID string) (int, string, error) {
	script := expandVariables(step.Script, build, job)

	// Create temp file for script
	tmpFile, err := os.CreateTemp("", "gagos_script_*.sh")
	if err != nil {
		return -1, "", fmt.Errorf("failed to create temp script: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Add shebang if not present
	if !strings.HasPrefix(script, "#!") {
		script = "#!/bin/sh\nset -e\n" + script
	}

	if _, err := tmpFile.WriteString(script); err != nil {
		return -1, "", fmt.Errorf("failed to write script: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return -1, "", fmt.Errorf("failed to chmod script: %w", err)
	}

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Running script: %s\n", tmpFile.Name())))

	cmd := exec.CommandContext(ctx, tmpFile.Name())

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range build.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	stream := GetBuildOutputStream(buildID)
	if stream != nil {
		cmd.Stdout = &streamWriter{stream: stream, buffer: &stdout}
		cmd.Stderr = &streamWriter{stream: stream, buffer: &stderr}
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err = cmd.Run()
	output := stdout.String() + stderr.String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return -1, output, err
		}
	}

	return exitCode, output, nil
}

// streamWriter writes to both a stream and a buffer
type streamWriter struct {
	stream *BuildOutputStream
	buffer *bytes.Buffer
}

func (w *streamWriter) Write(p []byte) (n int, err error) {
	if w.buffer != nil {
		w.buffer.Write(p)
	}
	if w.stream != nil {
		w.stream.Write(p)
	}
	return len(p), nil
}

// executeShellStep executes a shell command step
func executeShellStep(ctx context.Context, session *SSHSession, step *BuildStep, build *FreestyleBuild, job *FreestyleJob, timeout time.Duration, buildID string) (int, string, error) {
	cmd := expandVariables(step.Command, build, job)

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("$ %s\n", cmd)))

	// Create a buffer to capture output for storage
	var outputBuf bytes.Buffer

	// Create a wrapper that writes to both output stream and captures output
	stream := GetBuildOutputStream(buildID)
	if stream == nil {
		// Fallback to non-streaming
		stdout, stderr, exitCode, err := session.ExecuteCommand(ctx, cmd, timeout)
		output := stdout + stderr
		return exitCode, output, err
	}

	// Use MultiWriter to write to both stream (for live updates) and buffer (for storage)
	multiWriter := io.MultiWriter(stream, &outputBuf)
	exitCode, err := session.ExecuteCommandStreaming(ctx, cmd, timeout, multiWriter)
	return exitCode, outputBuf.String(), err
}

// executeScriptStep executes a script step
func executeScriptStep(ctx context.Context, session *SSHSession, step *BuildStep, build *FreestyleBuild, job *FreestyleJob, timeout time.Duration, buildID string) (int, string, error) {
	script := expandVariables(step.Script, build, job)

	// Upload script to temp file and execute
	scriptPath := fmt.Sprintf("/tmp/gagos_script_%s.sh", build.ID)

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Uploading script to %s\n", scriptPath)))

	// Add shebang if not present
	if !strings.HasPrefix(script, "#!") {
		script = "#!/bin/bash\nset -e\n" + script
	}

	if err := session.SCPPush("", scriptPath, []byte(script)); err != nil {
		return -1, "", fmt.Errorf("failed to upload script: %w", err)
	}

	// Make executable and run
	cmd := fmt.Sprintf("chmod +x %s && %s; EXIT_CODE=$?; rm -f %s; exit $EXIT_CODE", scriptPath, scriptPath, scriptPath)

	// Create a buffer to capture output for storage
	var outputBuf bytes.Buffer

	stream := GetBuildOutputStream(buildID)
	if stream == nil {
		stdout, stderr, exitCode, err := session.ExecuteCommand(ctx, cmd, timeout)
		return exitCode, stdout + stderr, err
	}

	// Use MultiWriter to write to both stream (for live updates) and buffer (for storage)
	multiWriter := io.MultiWriter(stream, &outputBuf)
	exitCode, err := session.ExecuteCommandStreaming(ctx, cmd, timeout, multiWriter)
	return exitCode, outputBuf.String(), err
}

// executeSCPPushStep copies files to remote
func executeSCPPushStep(session *SSHSession, step *BuildStep, build *FreestyleBuild) (int, string, error) {
	// Read local file
	content, err := os.ReadFile(step.LocalPath)
	if err != nil {
		return -1, "", fmt.Errorf("failed to read local file: %w", err)
	}

	// Push to remote
	if err := session.SCPPush(step.LocalPath, step.RemotePath, content); err != nil {
		return -1, "", fmt.Errorf("failed to push file: %w", err)
	}

	return 0, fmt.Sprintf("Copied %s -> %s (%d bytes)", step.LocalPath, step.RemotePath, len(content)), nil
}

// executeSCPPullStep copies files from remote
func executeSCPPullStep(session *SSHSession, step *BuildStep, build *FreestyleBuild) (int, string, error) {
	// Pull from remote
	content, err := session.SCPPull(step.RemotePath)
	if err != nil {
		return -1, "", fmt.Errorf("failed to pull file: %w", err)
	}

	// Write to local file
	if err := os.WriteFile(step.LocalPath, content, 0644); err != nil {
		return -1, "", fmt.Errorf("failed to write local file: %w", err)
	}

	return 0, fmt.Sprintf("Copied %s -> %s (%d bytes)", step.RemotePath, step.LocalPath, len(content)), nil
}

// expandVariables replaces variables in a string
func expandVariables(s string, build *FreestyleBuild, job *FreestyleJob) string {
	result := s

	// Expand parameters
	for k, v := range build.Parameters {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
		result = strings.ReplaceAll(result, "$"+k, v)
	}

	// Expand environment
	for k, v := range build.Environment {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
		result = strings.ReplaceAll(result, "$"+k, v)
	}

	// Expand built-in variables
	builtins := map[string]string{
		"BUILD_ID":     build.ID,
		"BUILD_NUMBER": fmt.Sprintf("%d", build.BuildNumber),
		"JOB_ID":       build.JobID,
		"JOB_NAME":     build.JobName,
		"TRIGGER_TYPE": build.TriggerType,
	}

	for k, v := range builtins {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
		result = strings.ReplaceAll(result, "$"+k, v)
	}

	return result
}

// TriggerFreestyleBuild creates and executes a new build
func TriggerFreestyleBuild(jobID string, triggerType string, triggerRef string, params map[string]string) (*FreestyleBuild, error) {
	build, err := CreateFreestyleBuild(jobID, triggerType, triggerRef, params)
	if err != nil {
		return nil, err
	}

	// Execute build in background
	go ExecuteFreestyleBuild(build.ID)

	return build, nil
}
