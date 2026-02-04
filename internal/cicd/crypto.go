package cicd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

var (
	encryptionKey []byte
	cryptoOnce    sync.Once
	cryptoErr     error
)

// InitCrypto initializes the encryption key from environment or derives one
func InitCrypto() error {
	cryptoOnce.Do(func() {
		keyStr := os.Getenv("GAGOS_ENCRYPTION_KEY")
		if keyStr == "" {
			// Derive key from stable identifier (namespace in K8s, or DB path)
			// Namespace is stable across pod restarts, unique per deployment
			stableID := ""

			// Try reading Kubernetes namespace (stable across restarts)
			if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				stableID = string(data)
			}

			// Fallback to DB path which is typically configured per deployment
			if stableID == "" {
				stableID = os.Getenv("GAGOS_DB_PATH")
			}

			// Final fallback
			if stableID == "" {
				stableID = "gagos-default"
			}

			keyStr = stableID + "-gagos-encryption-key-v1"
		}
		hash := sha256.Sum256([]byte(keyStr))
		encryptionKey = hash[:]
	})
	return cryptoErr
}

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plaintext string) (string, error) {
	if encryptionKey == nil {
		if err := InitCrypto(); err != nil {
			return "", err
		}
	}

	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt
func Decrypt(encrypted string) (string, error) {
	if encryptionKey == nil {
		if err := InitCrypto(); err != nil {
			return "", err
		}
	}

	if encrypted == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// MaskCredential returns a masked version of a credential for display
func MaskCredential(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
