package tools

import (
	"encoding/base64"
	"strings"
)

// Base64Result represents the result of a base64 operation
type Base64Result struct {
	Input   string `json:"input,omitempty"`
	Output  string `json:"output"`
	URLSafe bool   `json:"url_safe,omitempty"`
	Error   string `json:"error,omitempty"`
}

// EncodeBase64 encodes text to base64
func EncodeBase64(input string, urlSafe bool) Base64Result {
	var encoded string
	if urlSafe {
		encoded = base64.URLEncoding.EncodeToString([]byte(input))
	} else {
		encoded = base64.StdEncoding.EncodeToString([]byte(input))
	}
	return Base64Result{
		Output:  encoded,
		URLSafe: urlSafe,
	}
}

// DecodeBase64 decodes base64 to text
func DecodeBase64(input string, urlSafe bool) Base64Result {
	// Try to auto-detect if it's URL-safe encoded
	input = strings.TrimSpace(input)

	var decoded []byte
	var err error

	if urlSafe {
		decoded, err = base64.URLEncoding.DecodeString(input)
	} else {
		// Try standard first, then URL-safe
		decoded, err = base64.StdEncoding.DecodeString(input)
		if err != nil {
			decoded, err = base64.URLEncoding.DecodeString(input)
			urlSafe = true
		}
	}

	if err != nil {
		return Base64Result{
			Error: "Invalid base64 input: " + err.Error(),
		}
	}

	return Base64Result{
		Output:  string(decoded),
		URLSafe: urlSafe,
	}
}

// DecodeK8sSecret decodes all base64 values in a Kubernetes secret data map
func DecodeK8sSecret(data map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range data {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			result[key] = value + " (decode error)"
		} else {
			result[key] = string(decoded)
		}
	}
	return result
}
