package cicd

import "time"

// ============ SSH Host Types ============

// SSHAuthMethod defines how to authenticate
type SSHAuthMethod string

const (
	SSHAuthPassword SSHAuthMethod = "password"
	SSHAuthKey      SSHAuthMethod = "key"
)

// SSHHost represents a remote SSH host configuration
type SSHHost struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Host             string        `json:"host"`
	Port             int           `json:"port"`
	Username         string        `json:"username"`
	AuthMethod       SSHAuthMethod `json:"auth_method"`
	Password         string        `json:"password,omitempty"`          // Encrypted
	PrivateKey       string        `json:"private_key,omitempty"`       // Encrypted
	Passphrase       string        `json:"passphrase,omitempty"`        // Encrypted (for key)
	VerifyHostKey    bool          `json:"verify_host_key,omitempty"`   // Enable host key verification
	HostKeyType      string        `json:"host_key_type,omitempty"`     // ssh-rsa, ssh-ed25519, etc.
	HostFingerprint  string        `json:"host_fingerprint,omitempty"`  // SHA256 fingerprint
	HostGroups       []string      `json:"host_groups,omitempty"`
	Description      string        `json:"description,omitempty"`
	LastTested       *time.Time    `json:"last_tested,omitempty"`
	TestStatus       string        `json:"test_status,omitempty"`       // success, failed, untested
	TestError        string        `json:"test_error,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// SSHHostSafe is SSHHost without sensitive data for API responses
type SSHHostSafe struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Username        string        `json:"username"`
	AuthMethod      SSHAuthMethod `json:"auth_method"`
	HasPassword     bool          `json:"has_password,omitempty"`
	HasKey          bool          `json:"has_key,omitempty"`
	VerifyHostKey   bool          `json:"verify_host_key,omitempty"`
	HostKeyType     string        `json:"host_key_type,omitempty"`
	HostFingerprint string        `json:"host_fingerprint,omitempty"`
	HostGroups      []string      `json:"host_groups,omitempty"`
	Description     string        `json:"description,omitempty"`
	LastTested      *time.Time    `json:"last_tested,omitempty"`
	TestStatus      string        `json:"test_status,omitempty"`
	TestError       string        `json:"test_error,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// ToSafe converts SSHHost to SSHHostSafe (without credentials)
func (h *SSHHost) ToSafe() SSHHostSafe {
	return SSHHostSafe{
		ID:              h.ID,
		Name:            h.Name,
		Host:            h.Host,
		Port:            h.Port,
		Username:        h.Username,
		AuthMethod:      h.AuthMethod,
		HasPassword:     h.Password != "",
		HasKey:          h.PrivateKey != "",
		VerifyHostKey:   h.VerifyHostKey,
		HostKeyType:     h.HostKeyType,
		HostFingerprint: h.HostFingerprint,
		HostGroups:      h.HostGroups,
		Description:     h.Description,
		LastTested:      h.LastTested,
		TestStatus:      h.TestStatus,
		TestError:       h.TestError,
		CreatedAt:       h.CreatedAt,
		UpdatedAt:       h.UpdatedAt,
	}
}

// ============ Git Credential Types ============

// GitAuthMethod defines how to authenticate to Git
type GitAuthMethod string

const (
	GitAuthToken    GitAuthMethod = "token"    // Personal access token (HTTPS)
	GitAuthPassword GitAuthMethod = "password" // Username + password (HTTPS)
	GitAuthSSHKey   GitAuthMethod = "ssh_key"  // SSH private key
)

// GitCredential represents stored Git credentials
type GitCredential struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	AuthMethod  GitAuthMethod `json:"auth_method"`

	// For token auth (encrypted)
	Token string `json:"token,omitempty"`

	// For password auth (encrypted)
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// For SSH key auth (encrypted)
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`

	TestStatus string     `json:"test_status,omitempty"` // success, failed, untested
	TestError  string     `json:"test_error,omitempty"`
	LastTested *time.Time `json:"last_tested,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// GitCredentialSafe is GitCredential without sensitive data for API responses
type GitCredentialSafe struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	AuthMethod  GitAuthMethod `json:"auth_method"`
	HasToken    bool          `json:"has_token,omitempty"`
	HasUsername bool          `json:"has_username,omitempty"`
	HasPassword bool          `json:"has_password,omitempty"`
	HasKey      bool          `json:"has_key,omitempty"`
	TestStatus  string        `json:"test_status,omitempty"`
	TestError   string        `json:"test_error,omitempty"`
	LastTested  *time.Time    `json:"last_tested,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ToSafe converts GitCredential to GitCredentialSafe (without credentials)
func (c *GitCredential) ToSafe() GitCredentialSafe {
	return GitCredentialSafe{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		AuthMethod:  c.AuthMethod,
		HasToken:    c.Token != "",
		HasUsername: c.Username != "",
		HasPassword: c.Password != "",
		HasKey:      c.PrivateKey != "",
		TestStatus:  c.TestStatus,
		TestError:   c.TestError,
		LastTested:  c.LastTested,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// ============ Git SCM Types ============

// GitRepository represents a Git repository configuration
type GitRepository struct {
	URL          string `json:"url"`           // Repository URL
	CredentialID string `json:"credential_id"` // Reference to GitCredential (optional for public repos)
}

// GitBranch represents a branch specifier
type GitBranch struct {
	Specifier string `json:"specifier"` // "*/main", "refs/heads/*", blank for any
}

// GitSCMConfig holds Source Code Management configuration
type GitSCMConfig struct {
	Type         string          `json:"type"` // "none" or "git"
	Repositories []GitRepository `json:"repositories,omitempty"`
	Branches     []GitBranch     `json:"branches,omitempty"`
	CloneDepth   int             `json:"clone_depth,omitempty"`   // 0 = full clone
	Submodules   bool            `json:"submodules,omitempty"`    // Clone submodules
	CleanBefore  bool            `json:"clean_before,omitempty"`  // Clean workspace before clone
}

// ============ Freestyle Job Types ============

// BuildStepType defines the type of build step
type BuildStepType string

const (
	StepTypeShell    BuildStepType = "shell"    // Execute shell command
	StepTypeScript   BuildStepType = "script"   // Execute script content
	StepTypeSCPPush  BuildStepType = "scp_push" // Copy files TO remote
	StepTypeSCPPull  BuildStepType = "scp_pull" // Copy files FROM remote
)

// BuildStep represents a single step in a freestyle job
type BuildStep struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Type            BuildStepType `json:"type"`
	Order           int           `json:"order"`
	HostID          string        `json:"host_id"`                     // SSH host to execute on
	Command         string        `json:"command,omitempty"`           // For shell type
	Script          string        `json:"script,omitempty"`            // For script type
	LocalPath       string        `json:"local_path,omitempty"`        // For SCP
	RemotePath      string        `json:"remote_path,omitempty"`       // For SCP
	Timeout         int           `json:"timeout,omitempty"`           // Seconds, default 300
	ContinueOnError bool          `json:"continue_on_error,omitempty"`
}

// BuildParameter defines a user-input parameter for job runs
type BuildParameter struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"` // string, boolean, choice
	Description  string   `json:"description,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	Choices      []string `json:"choices,omitempty"` // For choice type
	Required     bool     `json:"required,omitempty"`
}

// FreestyleTrigger defines how a freestyle job can be triggered
type FreestyleTrigger struct {
	Type     string `json:"type"`               // manual, cron, webhook
	Schedule string `json:"schedule,omitempty"` // Cron expression
	Enabled  bool   `json:"enabled"`
}

// FreestyleJob represents a UI-configured job
type FreestyleJob struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Enabled      bool               `json:"enabled"`
	SCM          *GitSCMConfig      `json:"scm,omitempty"` // Source Code Management
	Parameters   []BuildParameter   `json:"parameters,omitempty"`
	Environment  map[string]string  `json:"environment,omitempty"`
	BuildSteps   []BuildStep        `json:"build_steps"`
	Triggers     []FreestyleTrigger `json:"triggers,omitempty"`
	Status       FreestyleJobStatus `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// FreestyleJobStatus holds runtime status
type FreestyleJobStatus struct {
	WebhookURL    string     `json:"webhook_url,omitempty"`
	WebhookToken  string     `json:"webhook_token,omitempty"`
	WebhookSecret string     `json:"webhook_secret,omitempty"` // For HMAC signature verification
	LastBuildID   string     `json:"last_build_id,omitempty"`
	LastBuildAt   *time.Time `json:"last_build_at,omitempty"`
	LastStatus    string     `json:"last_status,omitempty"`
	TotalBuilds   int        `json:"total_builds"`
}

// ============ Freestyle Build Types ============

// FreestyleBuild represents an execution of a freestyle job
type FreestyleBuild struct {
	ID           string               `json:"id"`
	JobID        string               `json:"job_id"`
	JobName      string               `json:"job_name"`
	BuildNumber  int                  `json:"build_number"`
	Status       RunStatus            `json:"status"` // Reuse from existing
	TriggerType  string               `json:"trigger_type"`
	TriggerRef   string               `json:"trigger_ref,omitempty"`
	Parameters   map[string]string    `json:"parameters,omitempty"`
	Environment  map[string]string    `json:"environment,omitempty"`
	Steps        []FreestyleBuildStep `json:"steps"`
	StartedAt    *time.Time           `json:"started_at,omitempty"`
	FinishedAt   *time.Time           `json:"finished_at,omitempty"`
	Duration     int64                `json:"duration_ms,omitempty"`
	Error        string               `json:"error,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
}

// FreestyleBuildStep represents execution of a single build step
type FreestyleBuildStep struct {
	StepID     string        `json:"step_id"`
	Name       string        `json:"name"`
	Type       BuildStepType `json:"type"`
	HostID     string        `json:"host_id"`
	HostName   string        `json:"host_name"`
	Status     RunStatus     `json:"status"`
	StartedAt  *time.Time    `json:"started_at,omitempty"`
	FinishedAt *time.Time    `json:"finished_at,omitempty"`
	Duration   int64         `json:"duration_ms,omitempty"`
	ExitCode   int           `json:"exit_code"`
	Output     string        `json:"output,omitempty"` // stdout+stderr
	Error      string        `json:"error,omitempty"`
}

// ============ Request/Response Types ============

// CreateSSHHostRequest is the request body for creating an SSH host
type CreateSSHHostRequest struct {
	Name        string        `json:"name"`
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	Username    string        `json:"username"`
	AuthMethod  SSHAuthMethod `json:"auth_method"`
	Password    string        `json:"password,omitempty"`
	PrivateKey  string        `json:"private_key,omitempty"`
	Passphrase  string        `json:"passphrase,omitempty"`
	HostGroups  []string      `json:"host_groups,omitempty"`
	Description string        `json:"description,omitempty"`
}

// UpdateSSHHostRequest is the request body for updating an SSH host
type UpdateSSHHostRequest struct {
	Name        string        `json:"name,omitempty"`
	Host        string        `json:"host,omitempty"`
	Port        int           `json:"port,omitempty"`
	Username    string        `json:"username,omitempty"`
	AuthMethod  SSHAuthMethod `json:"auth_method,omitempty"`
	Password    string        `json:"password,omitempty"`
	PrivateKey  string        `json:"private_key,omitempty"`
	Passphrase  string        `json:"passphrase,omitempty"`
	HostGroups  []string      `json:"host_groups,omitempty"`
	Description string        `json:"description,omitempty"`
}

// CreateFreestyleJobRequest is the request body for creating a freestyle job
type CreateFreestyleJobRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Enabled     bool               `json:"enabled"`
	SCM         *GitSCMConfig      `json:"scm,omitempty"`
	Parameters  []BuildParameter   `json:"parameters,omitempty"`
	Environment map[string]string  `json:"environment,omitempty"`
	BuildSteps  []BuildStep        `json:"build_steps"`
	Triggers    []FreestyleTrigger `json:"triggers,omitempty"`
}

// TriggerFreestyleBuildRequest is the request body for triggering a build
type TriggerFreestyleBuildRequest struct {
	Parameters map[string]string `json:"parameters,omitempty"`
}

// CreateGitCredentialRequest is the request body for creating a Git credential
type CreateGitCredentialRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	AuthMethod  GitAuthMethod `json:"auth_method"`
	Token       string        `json:"token,omitempty"`
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`
	PrivateKey  string        `json:"private_key,omitempty"`
	Passphrase  string        `json:"passphrase,omitempty"`
}

// UpdateGitCredentialRequest is the request body for updating a Git credential
type UpdateGitCredentialRequest struct {
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	AuthMethod  GitAuthMethod `json:"auth_method,omitempty"`
	Token       string        `json:"token,omitempty"`
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`
	PrivateKey  string        `json:"private_key,omitempty"`
	Passphrase  string        `json:"passphrase,omitempty"`
}

// TestGitCredentialRequest is the request body for testing a Git credential
type TestGitCredentialRequest struct {
	URL string `json:"url"` // Repository URL to test against
}
