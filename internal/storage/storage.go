package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

var (
	backend StorageBackend
	once    sync.Once
)

// Legacy constants for backwards compatibility
const (
	notepadBucket   = BucketNotepad
	pipelinesBucket = BucketPipelines
	runsBucket      = BucketRuns
	artifactsBucket = BucketArtifacts
	defaultDBPath   = "/data/gagos.db"
)

// NotepadData represents the notepad content
type NotepadData struct {
	Content   string `json:"content"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Init initializes the storage backend based on environment variables
// Environment variables:
//   - GAGOS_STORAGE_TYPE: bbolt (default), postgres, redis
//   - GAGOS_DB_PATH: path for BBolt database (default: /data/gagos.db)
//   - GAGOS_POSTGRES_URL: PostgreSQL connection URL (e.g., postgres://user:pass@host:5432/dbname?sslmode=disable)
//   - GAGOS_REDIS_URL: Redis connection URL (e.g., redis://localhost:6379/0)
func Init() error {
	var initErr error
	once.Do(func() {
		storageType := os.Getenv("GAGOS_STORAGE_TYPE")
		if storageType == "" {
			storageType = StorageTypeBBolt
		}

		log.Info().Str("storage_type", storageType).Msg("Initializing storage")

		switch storageType {
		case StorageTypePostgres:
			url := os.Getenv("GAGOS_POSTGRES_URL")
			if url == "" {
				log.Warn().Msg("GAGOS_POSTGRES_URL not set, falling back to BBolt")
				backend = NewBBoltBackend(os.Getenv("GAGOS_DB_PATH"))
			} else {
				backend = NewPostgresBackend(url)
			}

		case StorageTypeRedis:
			url := os.Getenv("GAGOS_REDIS_URL")
			if url == "" {
				log.Warn().Msg("GAGOS_REDIS_URL not set, falling back to BBolt")
				backend = NewBBoltBackend(os.Getenv("GAGOS_DB_PATH"))
			} else {
				backend = NewRedisBackend(url)
			}

		case StorageTypeBBolt:
			fallthrough
		default:
			backend = NewBBoltBackend(os.Getenv("GAGOS_DB_PATH"))
		}

		if err := backend.Init(); err != nil {
			log.Error().Err(err).Msg("Failed to initialize storage backend")
			initErr = err
			return
		}

		log.Info().
			Str("type", backend.Type()).
			Msg("Storage backend initialized successfully")
	})
	return initErr
}

// Close closes the storage connection
func Close() error {
	if backend != nil {
		return backend.Close()
	}
	return nil
}

// GetBackend returns the current storage backend
func GetBackend() StorageBackend {
	return backend
}

// GetDB returns the BBolt database if using BBolt backend (for backwards compatibility)
func GetDB() *bolt.DB {
	if b, ok := backend.(*BBoltBackend); ok {
		return b.GetDB()
	}
	return nil
}

// ========== Notepad Functions ==========

// SaveNotepad saves notepad content
func SaveNotepad(key string, data *NotepadData) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return backend.Set(notepadBucket, key, encoded)
}

// GetNotepad retrieves notepad content
func GetNotepad(key string) (*NotepadData, error) {
	data, err := backend.Get(notepadBucket, key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return &NotepadData{}, nil
	}
	var notepad NotepadData
	if err := json.Unmarshal(data, &notepad); err != nil {
		return nil, err
	}
	return &notepad, nil
}

// DeleteNotepad removes notepad content
func DeleteNotepad(key string) error {
	return backend.Delete(notepadBucket, key)
}

// ListNotepads returns all notepad keys
func ListNotepads() ([]string, error) {
	return backend.ListKeys(notepadBucket)
}

// ========== CI/CD Pipeline Storage Functions ==========

// SavePipeline stores a pipeline
func SavePipeline(id string, data []byte) error {
	return backend.Set(pipelinesBucket, id, data)
}

// GetPipeline retrieves a pipeline by ID
func GetPipeline(id string) ([]byte, error) {
	return backend.Get(pipelinesBucket, id)
}

// DeletePipeline removes a pipeline
func DeletePipeline(id string) error {
	return backend.Delete(pipelinesBucket, id)
}

// ListPipelines returns all pipeline data
func ListPipelines() ([][]byte, error) {
	return backend.List(pipelinesBucket)
}

// ========== CI/CD Run Storage Functions ==========

// SaveRun stores a pipeline run
func SaveRun(id string, data []byte) error {
	return backend.Set(runsBucket, id, data)
}

// GetRun retrieves a run by ID
func GetRun(id string) ([]byte, error) {
	return backend.Get(runsBucket, id)
}

// DeleteRun removes a run
func DeleteRun(id string) error {
	return backend.Delete(runsBucket, id)
}

// ListRuns returns all run data
func ListRuns() ([][]byte, error) {
	return backend.List(runsBucket)
}

// ========== CI/CD Artifact Storage Functions ==========

// SaveArtifact stores artifact metadata
func SaveArtifact(id string, data []byte) error {
	return backend.Set(artifactsBucket, id, data)
}

// GetArtifact retrieves artifact metadata by ID
func GetArtifact(id string) ([]byte, error) {
	return backend.Get(artifactsBucket, id)
}

// DeleteArtifact removes artifact metadata
func DeleteArtifact(id string) error {
	return backend.Delete(artifactsBucket, id)
}

// ListArtifacts returns all artifact metadata
func ListArtifacts() ([][]byte, error) {
	return backend.List(artifactsBucket)
}

// ========== Preferences Storage Functions ==========

// DesktopPreferences represents the desktop icon preferences
type DesktopPreferences struct {
	Slots     []*string `json:"slots,omitempty"`     // New format: 24 slots, each icon id or null
	IconOrder []string  `json:"icon_order,omitempty"` // Legacy format
	Hidden    []string  `json:"hidden,omitempty"`     // Legacy format
	UpdatedAt int64     `json:"updated_at"`
}

// SavePreference stores a preference value
func SavePreference(key string, data []byte) error {
	return backend.Set(BucketPreferences, key, data)
}

// GetPreference retrieves a preference value
func GetPreference(key string) ([]byte, error) {
	return backend.Get(BucketPreferences, key)
}

// DeletePreference removes a preference
func DeletePreference(key string) error {
	return backend.Delete(BucketPreferences, key)
}

// SaveDesktopPreferences saves desktop icon preferences
func SaveDesktopPreferences(prefs *DesktopPreferences) error {
	encoded, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	return backend.Set(BucketPreferences, "desktop", encoded)
}

// GetDesktopPreferences retrieves desktop icon preferences
func GetDesktopPreferences() (*DesktopPreferences, error) {
	data, err := backend.Get(BucketPreferences, "desktop")
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var prefs DesktopPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, err
	}
	return &prefs, nil
}
