package cicd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gaga951/gagos/internal/storage"

	"github.com/rs/zerolog/log"
)

// generateGitCredentialID generates a unique ID for a Git credential
func generateGitCredentialID() string {
	return generateID("git")
}

// CreateGitCredential creates a new Git credential
func CreateGitCredential(req *CreateGitCredentialRequest) (*GitCredential, error) {
	// Initialize crypto if not already done
	if err := InitCrypto(); err != nil {
		return nil, fmt.Errorf("failed to initialize crypto: %w", err)
	}

	// Encrypt credentials based on auth method
	var encToken, encUsername, encPassword, encKey, encPassphrase string
	var err error

	if req.Token != "" {
		encToken, err = Encrypt(req.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
	}

	if req.Username != "" {
		encUsername, err = Encrypt(req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt username: %w", err)
		}
	}

	if req.Password != "" {
		encPassword, err = Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
	}

	if req.PrivateKey != "" {
		encKey, err = Encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
	}

	if req.Passphrase != "" {
		encPassphrase, err = Encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
	}

	cred := &GitCredential{
		ID:          generateGitCredentialID(),
		Name:        req.Name,
		Description: req.Description,
		AuthMethod:  req.AuthMethod,
		Token:       encToken,
		Username:    encUsername,
		Password:    encPassword,
		PrivateKey:  encKey,
		Passphrase:  encPassphrase,
		TestStatus:  "untested",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to storage
	data, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credential: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketGitCredentials, cred.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save credential: %w", err)
	}

	log.Info().Str("id", cred.ID).Str("name", cred.Name).Msg("Git credential created")
	return cred, nil
}

// GetGitCredential retrieves a Git credential by ID
func GetGitCredential(id string) (*GitCredential, error) {
	data, err := storage.GetBackend().Get(storage.BucketGitCredentials, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("Git credential not found: %s", id)
	}

	var cred GitCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential: %w", err)
	}

	return &cred, nil
}

// ListGitCredentials returns all Git credentials
func ListGitCredentials() ([]*GitCredential, error) {
	dataList, err := storage.GetBackend().List(storage.BucketGitCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	creds := make([]*GitCredential, 0, len(dataList))
	for _, data := range dataList {
		var cred GitCredential
		if err := json.Unmarshal(data, &cred); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal Git credential")
			continue
		}
		creds = append(creds, &cred)
	}

	// Sort by name
	sort.Slice(creds, func(i, j int) bool {
		return creds[i].Name < creds[j].Name
	})

	return creds, nil
}

// ListGitCredentialsSafe returns all Git credentials without sensitive data
func ListGitCredentialsSafe() ([]GitCredentialSafe, error) {
	creds, err := ListGitCredentials()
	if err != nil {
		return nil, err
	}

	safeCreds := make([]GitCredentialSafe, len(creds))
	for i, c := range creds {
		safeCreds[i] = c.ToSafe()
	}

	return safeCreds, nil
}

// UpdateGitCredential updates an existing Git credential
func UpdateGitCredential(id string, req *UpdateGitCredentialRequest) (*GitCredential, error) {
	cred, err := GetGitCredential(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != "" {
		cred.Name = req.Name
	}
	if req.Description != "" {
		cred.Description = req.Description
	}
	if req.AuthMethod != "" {
		cred.AuthMethod = req.AuthMethod
	}

	// Update credentials if provided (re-encrypt)
	if req.Token != "" {
		encToken, err := Encrypt(req.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		cred.Token = encToken
	}

	if req.Username != "" {
		encUsername, err := Encrypt(req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt username: %w", err)
		}
		cred.Username = encUsername
	}

	if req.Password != "" {
		encPassword, err := Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		cred.Password = encPassword
	}

	if req.PrivateKey != "" {
		encKey, err := Encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		cred.PrivateKey = encKey
	}

	if req.Passphrase != "" {
		encPassphrase, err := Encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		cred.Passphrase = encPassphrase
	}

	cred.UpdatedAt = time.Now()

	// Save to storage
	data, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credential: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketGitCredentials, cred.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save credential: %w", err)
	}

	log.Info().Str("id", cred.ID).Str("name", cred.Name).Msg("Git credential updated")
	return cred, nil
}

// DeleteGitCredential deletes a Git credential
func DeleteGitCredential(id string) error {
	// Check if credential exists
	_, err := GetGitCredential(id)
	if err != nil {
		return err
	}

	// TODO: Check if credential is used by any freestyle jobs

	if err := storage.GetBackend().Delete(storage.BucketGitCredentials, id); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	log.Info().Str("id", id).Msg("Git credential deleted")
	return nil
}

// GetDecryptedGitCredential retrieves a Git credential with decrypted values
func GetDecryptedGitCredential(id string) (*GitCredential, error) {
	cred, err := GetGitCredential(id)
	if err != nil {
		return nil, err
	}

	// Decrypt credentials
	decrypted := &GitCredential{
		ID:          cred.ID,
		Name:        cred.Name,
		Description: cred.Description,
		AuthMethod:  cred.AuthMethod,
		TestStatus:  cred.TestStatus,
		TestError:   cred.TestError,
		LastTested:  cred.LastTested,
		CreatedAt:   cred.CreatedAt,
		UpdatedAt:   cred.UpdatedAt,
	}

	if cred.Token != "" {
		decrypted.Token, err = Decrypt(cred.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}
	}

	if cred.Username != "" {
		decrypted.Username, err = Decrypt(cred.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt username: %w", err)
		}
	}

	if cred.Password != "" {
		decrypted.Password, err = Decrypt(cred.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
	}

	if cred.PrivateKey != "" {
		decrypted.PrivateKey, err = Decrypt(cred.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
	}

	if cred.Passphrase != "" {
		decrypted.Passphrase, err = Decrypt(cred.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
		}
	}

	return decrypted, nil
}

// TestGitCredential tests a Git credential against a repository URL
func TestGitCredential(id string, repoURL string) error {
	cred, err := GetGitCredential(id)
	if err != nil {
		return err
	}

	// Get decrypted credentials for testing
	decrypted, err := GetDecryptedGitCredential(id)
	if err != nil {
		return err
	}

	// Test the credential
	testErr := testGitCredentialConnection(decrypted, repoURL)

	// Update test status
	now := time.Now()
	cred.LastTested = &now
	if testErr != nil {
		cred.TestStatus = "failed"
		cred.TestError = testErr.Error()
	} else {
		cred.TestStatus = "success"
		cred.TestError = ""
	}
	cred.UpdatedAt = now

	// Save updated status
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketGitCredentials, cred.ID, data); err != nil {
		return fmt.Errorf("failed to save credential status: %w", err)
	}

	if testErr != nil {
		return fmt.Errorf("credential test failed: %w", testErr)
	}

	log.Info().Str("id", id).Str("name", cred.Name).Msg("Git credential test passed")
	return nil
}

// testGitCredentialConnection tests the Git credential against a repo
func testGitCredentialConnection(cred *GitCredential, repoURL string) error {
	// This will be implemented in git_executor.go
	// For now, just validate the credential has required fields
	switch cred.AuthMethod {
	case GitAuthToken:
		if cred.Token == "" {
			return fmt.Errorf("token is required for token authentication")
		}
	case GitAuthPassword:
		if cred.Username == "" || cred.Password == "" {
			return fmt.Errorf("username and password are required for password authentication")
		}
	case GitAuthSSHKey:
		if cred.PrivateKey == "" {
			return fmt.Errorf("private key is required for SSH key authentication")
		}
	default:
		return fmt.Errorf("unknown auth method: %s", cred.AuthMethod)
	}
	return nil
}
