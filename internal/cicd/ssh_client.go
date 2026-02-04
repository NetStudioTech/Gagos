package cicd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHSession wraps an SSH client with helper methods
type SSHSession struct {
	client *ssh.Client
	host   *SSHHost
}

// NewSSHSession creates a new SSH session to the host
func NewSSHSession(host *SSHHost) (*SSHSession, error) {
	var authMethods []ssh.AuthMethod

	switch host.AuthMethod {
	case SSHAuthPassword:
		password, err := Decrypt(host.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(password))

	case SSHAuthKey:
		keyData, err := Decrypt(host.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key: %w", err)
		}

		var signer ssh.Signer
		if host.Passphrase != "" {
			passphrase, _ := Decrypt(host.Passphrase)
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(keyData), []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(keyData))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		return nil, fmt.Errorf("unsupported auth method: %s", host.AuthMethod)
	}

	// Set up host key callback
	var hostKeyCallback ssh.HostKeyCallback
	if host.VerifyHostKey && host.HostFingerprint != "" {
		// Verify against stored fingerprint
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			fingerprint := FingerprintSHA256(key)
			if fingerprint != host.HostFingerprint {
				return fmt.Errorf("host key mismatch: expected %s, got %s", host.HostFingerprint, fingerprint)
			}
			return nil
		}
	} else {
		// Skip verification (but log warning)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	port := host.Port
	if port == 0 {
		port = 22
	}

	addr := fmt.Sprintf("%s:%d", host.Host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &SSHSession{client: client, host: host}, nil
}

// Close closes the SSH connection
func (s *SSHSession) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// ExecuteCommand runs a command and returns output
func (s *SSHSession) ExecuteCommand(ctx context.Context, cmd string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return "", "", -1, ctx.Err()
	case <-time.After(timeout):
		session.Signal(ssh.SIGKILL)
		return stdoutBuf.String(), stderrBuf.String(), -1, fmt.Errorf("command timeout after %v", timeout)
	case runErr := <-done:
		exitCode = 0
		if runErr != nil {
			if exitErr, ok := runErr.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				return "", "", -1, runErr
			}
		}
		return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
	}
}

// ExecuteCommandStreaming runs a command and streams output to a writer
func (s *SSHSession) ExecuteCommandStreaming(ctx context.Context, cmd string, timeout time.Duration, output io.Writer) (exitCode int, err error) {
	session, err := s.client.NewSession()
	if err != nil {
		return -1, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Create pipes for stdout and stderr
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Multiplex stdout and stderr to output
	go io.Copy(output, stdoutPipe)
	go io.Copy(output, stderrPipe)

	// Start command
	if err := session.Start(cmd); err != nil {
		return -1, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return -1, ctx.Err()
	case <-time.After(timeout):
		session.Signal(ssh.SIGKILL)
		return -1, fmt.Errorf("command timeout after %v", timeout)
	case waitErr := <-done:
		exitCode = 0
		if waitErr != nil {
			if exitErr, ok := waitErr.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				return -1, waitErr
			}
		}
		return exitCode, nil
	}
}

// TestConnection verifies the SSH connection works
func TestSSHConnection(host *SSHHost) error {
	session, err := NewSSHSession(host)
	if err != nil {
		return err
	}
	defer session.Close()

	// Run a simple command to verify
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stdout, _, exitCode, err := session.ExecuteCommand(ctx, "echo 'GAGOS SSH Test OK'", 10*time.Second)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("test command failed with exit code %d", exitCode)
	}
	if len(stdout) == 0 {
		return fmt.Errorf("test command returned empty output")
	}

	return nil
}

// SCPPush copies a local file to the remote host
func (s *SSHSession) SCPPush(localPath, remotePath string, content []byte) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use cat with stdin for simple file transfer
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		w.Write(content)
	}()

	cmd := fmt.Sprintf("cat > %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// SCPPull reads a file from the remote host
func (s *SSHSession) SCPPull(remotePath string) ([]byte, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf

	cmd := fmt.Sprintf("cat %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return buf.Bytes(), nil
}

// FingerprintSHA256 returns the SHA256 fingerprint of an SSH public key
func FingerprintSHA256(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(hash[:]), "=")
}

// HostKeyInfo contains information about a host's SSH key
type HostKeyInfo struct {
	KeyType     string `json:"key_type"`
	Fingerprint string `json:"fingerprint"`
}

// GetHostFingerprint returns the host key fingerprint and type
func GetHostFingerprint(host string, port int) (*HostKeyInfo, error) {
	if port == 0 {
		port = 22
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	var hostKeyInfo *HostKeyInfo

	config := &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			hostKeyInfo = &HostKeyInfo{
				KeyType:     key.Type(),
				Fingerprint: FingerprintSHA256(key),
			}
			// Return an error to stop the handshake after getting the key
			return fmt.Errorf("got host key")
		},
		Timeout: 10 * time.Second,
	}

	// This will fail with our error, but we'll have captured the host key
	_, _, _, _ = ssh.NewClientConn(conn, addr, config)

	if hostKeyInfo == nil {
		return nil, fmt.Errorf("failed to get host key")
	}

	return hostKeyInfo, nil
}
