package cicd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"

	"github.com/gaga951/gagos/internal/k8s"
)

// GetJobLogs retrieves logs for a specific job in a run
func GetJobLogs(ctx context.Context, runID, jobName string, tailLines int64) (string, error) {
	run, err := GetRun(runID)
	if err != nil {
		return "", err
	}

	// Find the job
	var jobRun *JobRun
	for i := range run.Jobs {
		if run.Jobs[i].Name == jobName {
			jobRun = &run.Jobs[i]
			break
		}
	}

	if jobRun == nil {
		return "", fmt.Errorf("job not found: %s", jobName)
	}

	if jobRun.K8sPodName == "" {
		return "", fmt.Errorf("job has not started yet")
	}

	clientset := k8s.GetClient()
	if clientset == nil {
		return "", fmt.Errorf("kubernetes client not initialized")
	}

	// Get logs from pod
	opts := &corev1.PodLogOptions{
		Container: "runner",
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := clientset.CoreV1().Pods(cicdNamespace).GetLogs(jobRun.K8sPodName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer stream.Close()

	// Read all logs
	buf := make([]byte, 64*1024)
	var logs string
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			logs += string(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return logs, err
		}
	}

	return logs, nil
}

// StreamJobLogs streams logs for a job via WebSocket
func StreamJobLogs(c *websocket.Conn, runID, jobName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().
		Str("run_id", runID).
		Str("job", jobName).
		Msg("Starting log stream")

	// Get the run and job info
	run, err := GetRun(runID)
	if err != nil {
		sendWsError(c, fmt.Sprintf("Run not found: %s", err))
		return
	}

	// Find the job
	var jobRun *JobRun
	for i := range run.Jobs {
		if run.Jobs[i].Name == jobName {
			jobRun = &run.Jobs[i]
			break
		}
	}

	if jobRun == nil {
		sendWsError(c, fmt.Sprintf("Job not found: %s", jobName))
		return
	}

	// Send initial status
	sendWsStatus(c, string(jobRun.Status))

	// Wait for pod to be ready
	if jobRun.K8sPodName == "" {
		// Poll for pod name
		for i := 0; i < 30; i++ {
			run, err = GetRun(runID)
			if err != nil {
				sendWsError(c, err.Error())
				return
			}
			for j := range run.Jobs {
				if run.Jobs[j].Name == jobName {
					jobRun = &run.Jobs[j]
					break
				}
			}
			if jobRun.K8sPodName != "" {
				break
			}
			time.Sleep(time.Second)
		}

		if jobRun.K8sPodName == "" {
			sendWsError(c, "Timeout waiting for pod")
			return
		}
	}

	clientset := k8s.GetClient()
	if clientset == nil {
		sendWsError(c, "Kubernetes client not initialized")
		return
	}

	// Stream logs
	opts := &corev1.PodLogOptions{
		Container: "runner",
		Follow:    true,
	}

	req := clientset.CoreV1().Pods(cicdNamespace).GetLogs(jobRun.K8sPodName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		sendWsError(c, fmt.Sprintf("Failed to stream logs: %s", err))
		return
	}
	defer stream.Close()

	// Read and forward logs
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		msg := WsMessage{
			Type:      "log",
			Line:      line,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := c.WriteJSON(msg); err != nil {
			log.Warn().Err(err).Msg("Failed to send log line")
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Warn().Err(err).Msg("Scanner error")
	}

	// Check final status
	run, _ = GetRun(runID)
	for i := range run.Jobs {
		if run.Jobs[i].Name == jobName {
			jobRun = &run.Jobs[i]
			break
		}
	}

	// Send completion message
	msg := WsMessage{
		Type:     "complete",
		Status:   string(jobRun.Status),
		ExitCode: jobRun.ExitCode,
	}
	c.WriteJSON(msg)

	log.Info().
		Str("run_id", runID).
		Str("job", jobName).
		Str("status", string(jobRun.Status)).
		Msg("Log stream completed")
}

func sendWsError(c *websocket.Conn, errMsg string) {
	msg := WsMessage{
		Type:  "error",
		Error: errMsg,
	}
	c.WriteJSON(msg)
}

func sendWsStatus(c *websocket.Conn, status string) {
	msg := WsMessage{
		Type:   "status",
		Status: status,
	}
	c.WriteJSON(msg)
}
