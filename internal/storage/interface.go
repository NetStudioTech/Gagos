package storage

// StorageBackend defines the interface for all storage backends
type StorageBackend interface {
	// Initialize the storage backend
	Init() error
	// Close the storage connection
	Close() error
	// Get the backend type name
	Type() string

	// Generic key-value operations
	Set(bucket, key string, value []byte) error
	Get(bucket, key string) ([]byte, error)
	Delete(bucket, key string) error
	List(bucket string) ([][]byte, error)
	ListKeys(bucket string) ([]string, error)
}

// Supported storage types
const (
	StorageTypeBBolt    = "bbolt"
	StorageTypePostgres = "postgres"
	StorageTypeRedis    = "redis"
	StorageTypeMemory   = "memory" // For testing
)

// Bucket names
const (
	BucketNotepad         = "notepad"
	BucketPipelines       = "cicd_pipelines"
	BucketRuns            = "cicd_runs"
	BucketArtifacts       = "cicd_artifacts"
	BucketPreferences     = "preferences"
	BucketSSHHosts        = "ssh_hosts"
	BucketFreestyleJobs   = "freestyle_jobs"
	BucketFreestyleBuilds = "freestyle_builds"
	BucketNotifications   = "notifications"
	BucketGitCredentials  = "git_credentials"
)

// AllBuckets returns all bucket names
func AllBuckets() []string {
	return []string{
		BucketNotepad, BucketPipelines, BucketRuns, BucketArtifacts, BucketPreferences,
		BucketSSHHosts, BucketFreestyleJobs, BucketFreestyleBuilds, BucketNotifications,
		BucketGitCredentials,
	}
}
