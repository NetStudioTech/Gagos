package tools

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// HashResult represents the result of a hash operation
type HashResult struct {
	Algorithm string `json:"algorithm"`
	Hash      string `json:"hash"`
	Input     string `json:"input,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HashCompareResult represents the result of comparing hashes
type HashCompareResult struct {
	Hash1   string `json:"hash1"`
	Hash2   string `json:"hash2"`
	Match   bool   `json:"match"`
	Message string `json:"message"`
}

// AllHashesResult contains all hash types for input
type AllHashesResult struct {
	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
}

// HashMD5 generates MD5 hash
func HashMD5(input string) HashResult {
	hash := md5.Sum([]byte(input))
	return HashResult{
		Algorithm: "MD5",
		Hash:      hex.EncodeToString(hash[:]),
	}
}

// HashSHA1 generates SHA-1 hash
func HashSHA1(input string) HashResult {
	hash := sha1.Sum([]byte(input))
	return HashResult{
		Algorithm: "SHA-1",
		Hash:      hex.EncodeToString(hash[:]),
	}
}

// HashSHA256 generates SHA-256 hash
func HashSHA256(input string) HashResult {
	hash := sha256.Sum256([]byte(input))
	return HashResult{
		Algorithm: "SHA-256",
		Hash:      hex.EncodeToString(hash[:]),
	}
}

// HashSHA512 generates SHA-512 hash
func HashSHA512(input string) HashResult {
	hash := sha512.Sum512([]byte(input))
	return HashResult{
		Algorithm: "SHA-512",
		Hash:      hex.EncodeToString(hash[:]),
	}
}

// HashAll generates all hash types for input
func HashAll(input string) AllHashesResult {
	md5Hash := md5.Sum([]byte(input))
	sha1Hash := sha1.Sum([]byte(input))
	sha256Hash := sha256.Sum256([]byte(input))
	sha512Hash := sha512.Sum512([]byte(input))

	return AllHashesResult{
		MD5:    hex.EncodeToString(md5Hash[:]),
		SHA1:   hex.EncodeToString(sha1Hash[:]),
		SHA256: hex.EncodeToString(sha256Hash[:]),
		SHA512: hex.EncodeToString(sha512Hash[:]),
	}
}

// CompareHashes compares two hash strings
func CompareHashes(hash1, hash2 string) HashCompareResult {
	h1 := strings.ToLower(strings.TrimSpace(hash1))
	h2 := strings.ToLower(strings.TrimSpace(hash2))
	match := h1 == h2

	message := "Hashes do not match"
	if match {
		message = "Hashes match"
	}

	return HashCompareResult{
		Hash1:   h1,
		Hash2:   h2,
		Match:   match,
		Message: message,
	}
}

// HashFile generates hash for a file
func HashFile(filepath string, algorithm string) HashResult {
	file, err := os.Open(filepath)
	if err != nil {
		return HashResult{
			Algorithm: algorithm,
			Error:     "Failed to open file: " + err.Error(),
		}
	}
	defer file.Close()

	var hashStr string

	switch strings.ToUpper(algorithm) {
	case "MD5":
		h := md5.New()
		if _, err := io.Copy(h, file); err != nil {
			return HashResult{Algorithm: algorithm, Error: err.Error()}
		}
		hashStr = hex.EncodeToString(h.Sum(nil))
	case "SHA1", "SHA-1":
		h := sha1.New()
		if _, err := io.Copy(h, file); err != nil {
			return HashResult{Algorithm: algorithm, Error: err.Error()}
		}
		hashStr = hex.EncodeToString(h.Sum(nil))
	case "SHA256", "SHA-256":
		h := sha256.New()
		if _, err := io.Copy(h, file); err != nil {
			return HashResult{Algorithm: algorithm, Error: err.Error()}
		}
		hashStr = hex.EncodeToString(h.Sum(nil))
	case "SHA512", "SHA-512":
		h := sha512.New()
		if _, err := io.Copy(h, file); err != nil {
			return HashResult{Algorithm: algorithm, Error: err.Error()}
		}
		hashStr = hex.EncodeToString(h.Sum(nil))
	default:
		return HashResult{Algorithm: algorithm, Error: "Unknown algorithm: " + algorithm}
	}

	return HashResult{
		Algorithm: algorithm,
		Hash:      hashStr,
	}
}
