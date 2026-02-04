package cicd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gaga951/gagos/internal/storage"
)

// SaveArtifact saves an artifact file and metadata
func SaveArtifact(runID, pipelineID, name, filename string, data io.Reader) (*ArtifactMetadata, error) {
	// Ensure artifact directory exists
	dir := filepath.Join(artifactPath, runID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Generate artifact ID
	artifactID := generateID("art")

	// Create file
	filePath := filepath.Join(dir, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create artifact file: %w", err)
	}
	defer f.Close()

	// Write data and calculate checksum
	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	size, err := io.Copy(writer, data)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to write artifact: %w", err)
	}

	// Detect mime type
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Create metadata
	artifact := &ArtifactMetadata{
		ID:         artifactID,
		RunID:      runID,
		PipelineID: pipelineID,
		Name:       name,
		Filename:   filename,
		Path:       filePath,
		Size:       size,
		MimeType:   mimeType,
		Checksum:   hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt:  time.Now(),
	}

	// Save metadata
	data2, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}

	if err := storage.SaveArtifact(artifactID, data2); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save artifact metadata: %w", err)
	}

	log.Info().
		Str("artifact_id", artifactID).
		Str("name", name).
		Int64("size", size).
		Msg("Artifact saved")

	return artifact, nil
}

// GetArtifact retrieves artifact metadata by ID
func GetArtifact(artifactID string) (*ArtifactMetadata, error) {
	data, err := storage.GetArtifact(artifactID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("artifact not found: %s", artifactID)
	}

	var artifact ArtifactMetadata
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

// GetArtifactFile returns the file reader for an artifact
func GetArtifactFile(artifactID string) (*os.File, *ArtifactMetadata, error) {
	artifact, err := GetArtifact(artifactID)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(artifact.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("artifact file not found: %w", err)
	}

	return f, artifact, nil
}

// ListArtifacts returns all artifacts, optionally filtered
func ListArtifacts(runID, pipelineID string) ([]*ArtifactMetadata, error) {
	items, err := storage.ListArtifacts()
	if err != nil {
		return nil, err
	}

	artifacts := make([]*ArtifactMetadata, 0)
	for _, data := range items {
		var a ArtifactMetadata
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}
		if runID != "" && a.RunID != runID {
			continue
		}
		if pipelineID != "" && a.PipelineID != pipelineID {
			continue
		}
		artifacts = append(artifacts, &a)
	}

	// Sort by created_at descending
	for i := 0; i < len(artifacts)-1; i++ {
		for j := i + 1; j < len(artifacts); j++ {
			if artifacts[j].CreatedAt.After(artifacts[i].CreatedAt) {
				artifacts[i], artifacts[j] = artifacts[j], artifacts[i]
			}
		}
	}

	return artifacts, nil
}

// DeleteArtifact removes an artifact
func DeleteArtifact(artifactID string) error {
	artifact, err := GetArtifact(artifactID)
	if err != nil {
		return err
	}

	// Delete file
	if err := os.Remove(artifact.Path); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Str("path", artifact.Path).Msg("Failed to delete artifact file")
	}

	// Delete metadata
	return storage.DeleteArtifact(artifactID)
}

// CleanupOldArtifacts removes artifacts older than the specified duration
func CleanupOldArtifacts(maxAge time.Duration) (int, error) {
	artifacts, err := ListArtifacts("", "")
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for _, artifact := range artifacts {
		if artifact.CreatedAt.Before(cutoff) {
			if err := DeleteArtifact(artifact.ID); err != nil {
				log.Warn().Err(err).Str("artifact_id", artifact.ID).Msg("Failed to delete old artifact")
				continue
			}
			deleted++
		}
	}

	if deleted > 0 {
		log.Info().Int("count", deleted).Msg("Cleaned up old artifacts")
	}

	return deleted, nil
}

// CleanupRunArtifacts removes all artifacts for a run
func CleanupRunArtifacts(runID string) error {
	artifacts, err := ListArtifacts(runID, "")
	if err != nil {
		return err
	}

	for _, artifact := range artifacts {
		if err := DeleteArtifact(artifact.ID); err != nil {
			log.Warn().Err(err).Str("artifact_id", artifact.ID).Msg("Failed to delete artifact")
		}
	}

	// Remove run directory
	dir := filepath.Join(artifactPath, runID)
	os.RemoveAll(dir)

	return nil
}

// GetArtifactStats returns artifact storage statistics
func GetArtifactStats() (int, int64, error) {
	artifacts, err := ListArtifacts("", "")
	if err != nil {
		return 0, 0, err
	}

	var totalSize int64
	for _, a := range artifacts {
		totalSize += a.Size
	}

	return len(artifacts), totalSize, nil
}
