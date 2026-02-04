package cicd

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GitCloneResult contains the result of a Git clone operation
type GitCloneResult struct {
	Workspace string // Path where code was cloned
	Commit    string // HEAD commit SHA
	Branch    string // Current branch name
}

// ExecuteGitSCM clones repositories and checks out branches based on SCM config
// This function runs on the SSH host before build steps execute
func ExecuteGitSCM(buildID string, session *SSHSession, job *FreestyleJob, build *FreestyleBuild) (*GitCloneResult, error) {
	if job.SCM == nil || job.SCM.Type != "git" {
		return nil, nil // SCM not configured
	}

	if len(job.SCM.Repositories) == 0 {
		return nil, fmt.Errorf("no repositories configured in SCM")
	}

	WriteBuildOutput(buildID, []byte("\n=== Source Code Management ===\n"))

	// Create workspace directory
	workspace := fmt.Sprintf("/tmp/gagos-builds/%s/%d", job.ID, build.BuildNumber)
	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Workspace: %s\n", workspace)))

	// Clean workspace if configured
	if job.SCM.CleanBefore {
		WriteBuildOutput(buildID, []byte("Cleaning workspace...\n"))
		cmd := fmt.Sprintf("rm -rf %s", workspace)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, _, _, err := session.ExecuteCommand(ctx, cmd, 30*time.Second)
		cancel()
		if err != nil {
			// Ignore errors, directory might not exist
		}
	}

	// Create workspace
	mkdirCmd := fmt.Sprintf("mkdir -p %s", workspace)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_, stderr, exitCode, err := session.ExecuteCommand(ctx, mkdirCmd, 30*time.Second)
	cancel()
	if err != nil || exitCode != 0 {
		return nil, fmt.Errorf("failed to create workspace: %s", stderr)
	}

	result := &GitCloneResult{
		Workspace: workspace,
	}

	// Clone each repository
	for i, repo := range job.SCM.Repositories {
		repoNum := i + 1
		WriteBuildOutput(buildID, []byte(fmt.Sprintf("\n--- Repository %d: %s ---\n", repoNum, repo.URL)))

		// Determine clone path (first repo goes directly to workspace, others get subdirs)
		clonePath := workspace
		if i > 0 {
			// Extract repo name from URL for subdir
			repoName := extractRepoName(repo.URL)
			clonePath = fmt.Sprintf("%s/%s", workspace, repoName)
		}

		// Build git clone command with authentication
		cloneCmd, err := buildGitCloneCommand(repo, job.SCM, clonePath)
		if err != nil {
			return nil, fmt.Errorf("failed to build clone command for repo %d: %w", repoNum, err)
		}

		WriteBuildOutput(buildID, []byte(fmt.Sprintf("Cloning into %s...\n", clonePath)))

		// Execute clone
		stream := GetBuildOutputStream(buildID)
		cloneTimeout := 10 * time.Minute // Git clones can take a while

		ctx, cancel := context.WithTimeout(context.Background(), cloneTimeout)
		exitCode, err := session.ExecuteCommandStreaming(ctx, cloneCmd, cloneTimeout, stream)
		cancel()

		if err != nil || exitCode != 0 {
			return nil, fmt.Errorf("git clone failed for repo %d with exit code %d", repoNum, exitCode)
		}

		// Checkout branch if specified
		if len(job.SCM.Branches) > 0 && job.SCM.Branches[0].Specifier != "" {
			branch := job.SCM.Branches[0].Specifier
			if err := checkoutBranch(buildID, session, clonePath, branch); err != nil {
				return nil, fmt.Errorf("failed to checkout branch: %w", err)
			}
		}

		// Get HEAD commit for first repo
		if i == 0 {
			commit, branch, err := getGitInfo(session, clonePath)
			if err == nil {
				result.Commit = commit
				result.Branch = branch
				WriteBuildOutput(buildID, []byte(fmt.Sprintf("Commit: %s\n", commit)))
				WriteBuildOutput(buildID, []byte(fmt.Sprintf("Branch: %s\n", branch)))
			}
		}
	}

	WriteBuildOutput(buildID, []byte("\n=== SCM Checkout Complete ===\n\n"))
	return result, nil
}

// buildGitCloneCommand constructs the git clone command with authentication
func buildGitCloneCommand(repo GitRepository, scm *GitSCMConfig, clonePath string) (string, error) {
	url := repo.URL

	// If credential is specified, get it and inject auth into URL
	if repo.CredentialID != "" {
		cred, err := GetDecryptedGitCredential(repo.CredentialID)
		if err != nil {
			return "", fmt.Errorf("failed to get credential: %w", err)
		}

		switch cred.AuthMethod {
		case GitAuthToken:
			// Inject token into HTTPS URL
			url = injectTokenIntoURL(url, cred.Token)

		case GitAuthPassword:
			// Inject username:password into HTTPS URL
			url = injectCredentialsIntoURL(url, cred.Username, cred.Password)

		case GitAuthSSHKey:
			// For SSH, we need to wrap the command with ssh-agent
			return buildSSHKeyCloneCommand(repo.URL, cred, scm, clonePath)
		}
	}

	// Build clone command with options
	cmd := "git clone"

	if scm.CloneDepth > 0 {
		cmd += fmt.Sprintf(" --depth %d", scm.CloneDepth)
	}

	if scm.Submodules {
		cmd += " --recurse-submodules"
	}

	// Use single quotes to prevent shell expansion
	cmd += fmt.Sprintf(" '%s' '%s' 2>&1", url, clonePath)

	return cmd, nil
}

// buildSSHKeyCloneCommand creates a clone command that uses ssh-agent with the provided key
func buildSSHKeyCloneCommand(url string, cred *GitCredential, scm *GitSCMConfig, clonePath string) (string, error) {
	// Build the base git clone command
	cloneOpts := ""
	if scm.CloneDepth > 0 {
		cloneOpts += fmt.Sprintf(" --depth %d", scm.CloneDepth)
	}
	if scm.Submodules {
		cloneOpts += " --recurse-submodules"
	}

	gitCmd := fmt.Sprintf("git clone%s '%s' '%s'", cloneOpts, url, clonePath)

	// Wrap with SSH agent setup
	// Note: We escape the key content and passphrase carefully
	keyEscaped := strings.ReplaceAll(cred.PrivateKey, "'", "'\"'\"'")

	cmd := fmt.Sprintf(`
KEYFILE=$(mktemp)
trap "rm -f $KEYFILE; ssh-agent -k >/dev/null 2>&1" EXIT
cat > "$KEYFILE" << 'GAGOS_SSH_KEY_EOF'
%s
GAGOS_SSH_KEY_EOF
chmod 600 "$KEYFILE"
eval $(ssh-agent -s) >/dev/null 2>&1
`, keyEscaped)

	if cred.Passphrase != "" {
		// Use expect or sshpass for passphrase if available, otherwise use ssh-add -p
		// For simplicity, we'll try without expect first
		cmd += `ssh-add "$KEYFILE" 2>&1 || echo "Warning: Could not add key (may need passphrase)"
`
	} else {
		cmd += `ssh-add "$KEYFILE" 2>&1
`
	}

	cmd += fmt.Sprintf(`GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" %s 2>&1
`, gitCmd)

	return cmd, nil
}

// injectTokenIntoURL injects a token into an HTTPS URL
func injectTokenIntoURL(url, token string) string {
	// Handle various URL formats
	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", "https://"+token+"@", 1)
	}
	if strings.HasPrefix(url, "http://") {
		return strings.Replace(url, "http://", "http://"+token+"@", 1)
	}
	return url
}

// injectCredentialsIntoURL injects username:password into an HTTPS URL
func injectCredentialsIntoURL(url, username, password string) string {
	// URL encode special characters in password
	escapedPassword := strings.ReplaceAll(password, "@", "%40")
	escapedPassword = strings.ReplaceAll(escapedPassword, ":", "%3A")

	auth := username + ":" + escapedPassword

	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", "https://"+auth+"@", 1)
	}
	if strings.HasPrefix(url, "http://") {
		return strings.Replace(url, "http://", "http://"+auth+"@", 1)
	}
	return url
}

// extractRepoName extracts the repository name from a Git URL
func extractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:user/repo
	if idx := strings.LastIndex(url, ":"); idx > 0 && strings.Contains(url[:idx], "@") {
		path := url[idx+1:]
		if lastSlash := strings.LastIndex(path, "/"); lastSlash >= 0 {
			return path[lastSlash+1:]
		}
		return path
	}

	// Handle HTTPS format: https://github.com/user/repo
	if lastSlash := strings.LastIndex(url, "/"); lastSlash >= 0 {
		return url[lastSlash+1:]
	}

	return "repo"
}

// checkoutBranch checks out a specific branch in the cloned repository
func checkoutBranch(buildID string, session *SSHSession, repoPath, branchSpec string) error {
	// Parse branch specifier
	// "*/main" -> "main"
	// "refs/heads/main" -> "main"
	// "origin/main" -> "main"
	branch := branchSpec
	if strings.HasPrefix(branch, "*/") {
		branch = branch[2:]
	}
	if strings.HasPrefix(branch, "refs/heads/") {
		branch = branch[11:]
	}
	if strings.HasPrefix(branch, "origin/") {
		branch = branch[7:]
	}

	WriteBuildOutput(buildID, []byte(fmt.Sprintf("Checking out branch: %s\n", branch)))

	// Fetch and checkout
	cmd := fmt.Sprintf("cd '%s' && git checkout '%s' 2>&1", repoPath, branch)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream := GetBuildOutputStream(buildID)
	exitCode, err := session.ExecuteCommandStreaming(ctx, cmd, 2*time.Minute, stream)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("checkout failed with exit code %d", exitCode)
	}

	return nil
}

// getGitInfo retrieves current commit SHA and branch name
func getGitInfo(session *SSHSession, repoPath string) (commit, branch string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get commit SHA
	commitCmd := fmt.Sprintf("cd '%s' && git rev-parse HEAD 2>/dev/null", repoPath)
	stdout, _, exitCode, _ := session.ExecuteCommand(ctx, commitCmd, 30*time.Second)
	if exitCode == 0 {
		commit = strings.TrimSpace(stdout)
	}

	// Get branch name
	branchCmd := fmt.Sprintf("cd '%s' && git rev-parse --abbrev-ref HEAD 2>/dev/null", repoPath)
	stdout, _, exitCode, _ = session.ExecuteCommand(ctx, branchCmd, 30*time.Second)
	if exitCode == 0 {
		branch = strings.TrimSpace(stdout)
	}

	return commit, branch, nil
}

// SetGitEnvironmentVariables returns environment variables for Git builds
func SetGitEnvironmentVariables(result *GitCloneResult) map[string]string {
	if result == nil {
		return nil
	}

	return map[string]string{
		"WORKSPACE":  result.Workspace,
		"GIT_COMMIT": result.Commit,
		"GIT_BRANCH": result.Branch,
	}
}
