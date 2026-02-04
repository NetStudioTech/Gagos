package cicd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gaga951/gagos/internal/storage"

	"github.com/rs/zerolog/log"
)

// generateSSHHostID generates a unique ID for an SSH host
func generateSSHHostID() string {
	return generateID("ssh")
}

// CreateSSHHost creates a new SSH host
func CreateSSHHost(req *CreateSSHHostRequest) (*SSHHost, error) {
	// Initialize crypto if not already done
	if err := InitCrypto(); err != nil {
		return nil, fmt.Errorf("failed to initialize crypto: %w", err)
	}

	// Encrypt credentials
	var encPassword, encKey, encPassphrase string
	var err error

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

	port := req.Port
	if port == 0 {
		port = 22
	}

	host := &SSHHost{
		ID:          generateSSHHostID(),
		Name:        req.Name,
		Host:        req.Host,
		Port:        port,
		Username:    req.Username,
		AuthMethod:  req.AuthMethod,
		Password:    encPassword,
		PrivateKey:  encKey,
		Passphrase:  encPassphrase,
		HostGroups:  req.HostGroups,
		Description: req.Description,
		TestStatus:  "untested",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to storage
	data, err := json.Marshal(host)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketSSHHosts, host.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save host: %w", err)
	}

	log.Info().Str("id", host.ID).Str("name", host.Name).Msg("SSH host created")
	return host, nil
}

// GetSSHHost retrieves an SSH host by ID
func GetSSHHost(id string) (*SSHHost, error) {
	data, err := storage.GetBackend().Get(storage.BucketSSHHosts, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("SSH host not found: %s", id)
	}

	var host SSHHost
	if err := json.Unmarshal(data, &host); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host: %w", err)
	}

	return &host, nil
}

// ListSSHHosts returns all SSH hosts
func ListSSHHosts() ([]*SSHHost, error) {
	dataList, err := storage.GetBackend().List(storage.BucketSSHHosts)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}

	hosts := make([]*SSHHost, 0, len(dataList))
	for _, data := range dataList {
		var host SSHHost
		if err := json.Unmarshal(data, &host); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal SSH host")
			continue
		}
		hosts = append(hosts, &host)
	}

	// Sort by name
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].Name < hosts[j].Name
	})

	return hosts, nil
}

// ListSSHHostsSafe returns all SSH hosts without sensitive data
func ListSSHHostsSafe() ([]SSHHostSafe, error) {
	hosts, err := ListSSHHosts()
	if err != nil {
		return nil, err
	}

	safeHosts := make([]SSHHostSafe, len(hosts))
	for i, h := range hosts {
		safeHosts[i] = h.ToSafe()
	}

	return safeHosts, nil
}

// UpdateSSHHost updates an existing SSH host
func UpdateSSHHost(id string, req *UpdateSSHHostRequest) (*SSHHost, error) {
	host, err := GetSSHHost(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != "" {
		host.Name = req.Name
	}
	if req.Host != "" {
		host.Host = req.Host
	}
	if req.Port != 0 {
		host.Port = req.Port
	}
	if req.Username != "" {
		host.Username = req.Username
	}
	if req.AuthMethod != "" {
		host.AuthMethod = req.AuthMethod
	}
	if req.HostGroups != nil {
		host.HostGroups = req.HostGroups
	}
	if req.Description != "" {
		host.Description = req.Description
	}

	// Update credentials if provided (re-encrypt)
	if req.Password != "" {
		encPassword, err := Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		host.Password = encPassword
	}

	if req.PrivateKey != "" {
		encKey, err := Encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		host.PrivateKey = encKey
	}

	if req.Passphrase != "" {
		encPassphrase, err := Encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		host.Passphrase = encPassphrase
	}

	host.UpdatedAt = time.Now()

	// Save to storage
	data, err := json.Marshal(host)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketSSHHosts, host.ID, data); err != nil {
		return nil, fmt.Errorf("failed to save host: %w", err)
	}

	log.Info().Str("id", host.ID).Str("name", host.Name).Msg("SSH host updated")
	return host, nil
}

// DeleteSSHHost deletes an SSH host
func DeleteSSHHost(id string) error {
	// Check if host exists
	_, err := GetSSHHost(id)
	if err != nil {
		return err
	}

	// TODO: Check if host is used by any freestyle jobs

	if err := storage.GetBackend().Delete(storage.BucketSSHHosts, id); err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}

	log.Info().Str("id", id).Msg("SSH host deleted")
	return nil
}

// TestSSHHostConnection tests the connection to an SSH host
func TestSSHHostConnection(id string) error {
	host, err := GetSSHHost(id)
	if err != nil {
		return err
	}

	// Test the connection
	testErr := TestSSHConnection(host)

	// Update test status
	now := time.Now()
	host.LastTested = &now
	if testErr != nil {
		host.TestStatus = "failed"
		host.TestError = testErr.Error()
	} else {
		host.TestStatus = "success"
		host.TestError = ""
	}
	host.UpdatedAt = now

	// Save updated status
	data, err := json.Marshal(host)
	if err != nil {
		return fmt.Errorf("failed to marshal host: %w", err)
	}

	if err := storage.GetBackend().Set(storage.BucketSSHHosts, host.ID, data); err != nil {
		return fmt.Errorf("failed to save host status: %w", err)
	}

	if testErr != nil {
		return fmt.Errorf("connection test failed: %w", testErr)
	}

	log.Info().Str("id", id).Str("name", host.Name).Msg("SSH host connection test passed")
	return nil
}

// GetSSHHostGroups returns all unique host groups
func GetSSHHostGroups() ([]string, error) {
	hosts, err := ListSSHHosts()
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]bool)
	for _, h := range hosts {
		for _, g := range h.HostGroups {
			groupMap[g] = true
		}
	}

	groups := make([]string, 0, len(groupMap))
	for g := range groupMap {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	return groups, nil
}
