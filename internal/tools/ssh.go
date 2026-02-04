package tools

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHKeyPair represents an SSH key pair
type SSHKeyPair struct {
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	Algorithm   string `json:"algorithm"`
	BitSize     int    `json:"bit_size,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SSHKeyInfo represents information about an SSH key
type SSHKeyInfo struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	BitSize     int    `json:"bit_size,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

// GenerateSSHKeyPair generates a new SSH key pair
func GenerateSSHKeyPair(algorithm string, bitSize int) SSHKeyPair {
	algorithm = strings.ToUpper(algorithm)

	switch algorithm {
	case "RSA":
		return generateRSAKey(bitSize)
	case "ECDSA":
		return generateECDSAKey(bitSize)
	case "ED25519":
		return generateEd25519Key()
	default:
		return SSHKeyPair{
			Algorithm: algorithm,
			Error:     "Unsupported algorithm. Use RSA, ECDSA, or ED25519",
		}
	}
}

func generateRSAKey(bitSize int) SSHKeyPair {
	if bitSize == 0 {
		bitSize = 4096
	}
	if bitSize < 2048 {
		bitSize = 2048
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return SSHKeyPair{Algorithm: "RSA", Error: err.Error()}
	}

	// Generate private key PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return SSHKeyPair{Algorithm: "RSA", Error: err.Error()}
	}

	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))
	fingerprint := getFingerprint(publicKey)

	return SSHKeyPair{
		PrivateKey:  string(privateKeyPEM),
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		Algorithm:   "RSA",
		BitSize:     bitSize,
	}
}

func generateECDSAKey(bitSize int) SSHKeyPair {
	var curve elliptic.Curve
	switch bitSize {
	case 256, 0:
		curve = elliptic.P256()
		bitSize = 256
	case 384:
		curve = elliptic.P384()
	case 521:
		curve = elliptic.P521()
	default:
		return SSHKeyPair{Algorithm: "ECDSA", Error: "Invalid bit size. Use 256, 384, or 521"}
	}

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return SSHKeyPair{Algorithm: "ECDSA", Error: err.Error()}
	}

	// Generate private key PEM
	ecBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return SSHKeyPair{Algorithm: "ECDSA", Error: err.Error()}
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ecBytes,
	})

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return SSHKeyPair{Algorithm: "ECDSA", Error: err.Error()}
	}

	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))
	fingerprint := getFingerprint(publicKey)

	return SSHKeyPair{
		PrivateKey:  string(privateKeyPEM),
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		Algorithm:   "ECDSA",
		BitSize:     bitSize,
	}
}

func generateEd25519Key() SSHKeyPair {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return SSHKeyPair{Algorithm: "ED25519", Error: err.Error()}
	}

	// For Ed25519, we need to use OpenSSH format
	// Generate public key first
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return SSHKeyPair{Algorithm: "ED25519", Error: err.Error()}
	}

	// Marshal private key in OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return SSHKeyPair{Algorithm: "ED25519", Error: err.Error()}
	}

	privateKeyPEM := pem.EncodeToMemory(pemBlock)
	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey)))
	fingerprint := getFingerprint(sshPublicKey)

	return SSHKeyPair{
		PrivateKey:  string(privateKeyPEM),
		PublicKey:   publicKeyStr,
		Fingerprint: fingerprint,
		Algorithm:   "ED25519",
	}
}

// GetSSHKeyInfo parses and returns info about an SSH key
func GetSSHKeyInfo(keyData string) SSHKeyInfo {
	keyData = strings.TrimSpace(keyData)

	// Try parsing as public key first
	if strings.HasPrefix(keyData, "ssh-") || strings.HasPrefix(keyData, "ecdsa-") {
		return parsePublicKey(keyData)
	}

	// Try parsing as PEM private key
	if strings.HasPrefix(keyData, "-----BEGIN") {
		return parsePrivateKey(keyData)
	}

	return SSHKeyInfo{
		Valid: false,
		Error: "Unrecognized key format",
	}
}

func parsePublicKey(keyData string) SSHKeyInfo {
	parts := strings.Fields(keyData)
	if len(parts) < 2 {
		return SSHKeyInfo{Valid: false, Error: "Invalid public key format"}
	}

	publicKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(keyData))
	if err != nil {
		return SSHKeyInfo{Valid: false, Error: err.Error()}
	}

	info := SSHKeyInfo{
		Type:        publicKey.Type(),
		Fingerprint: getFingerprint(publicKey),
		Comment:     comment,
		PublicKey:   keyData,
		Valid:       true,
	}

	// Try to determine bit size
	switch publicKey.Type() {
	case "ssh-rsa":
		info.BitSize = getRSABitSize(publicKey)
	}

	return info
}

func parsePrivateKey(keyData string) SSHKeyInfo {
	block, _ := pem.Decode([]byte(keyData))
	if block == nil {
		return SSHKeyInfo{Valid: false, Error: "Failed to parse PEM block"}
	}

	signer, err := ssh.ParsePrivateKey([]byte(keyData))
	if err != nil {
		return SSHKeyInfo{Valid: false, Error: err.Error()}
	}

	publicKey := signer.PublicKey()

	return SSHKeyInfo{
		Type:        publicKey.Type(),
		Fingerprint: getFingerprint(publicKey),
		PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))),
		Valid:       true,
	}
}

func getFingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

func getRSABitSize(key ssh.PublicKey) int {
	// Parse the public key to get bit size
	cryptoKey := key.(ssh.CryptoPublicKey).CryptoPublicKey()
	if rsaKey, ok := cryptoKey.(*rsa.PublicKey); ok {
		return rsaKey.N.BitLen()
	}
	return 0
}
