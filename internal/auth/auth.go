// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

var (
	password     string
	sessions     = make(map[string]time.Time)
	sessionMutex sync.RWMutex
	sessionTTL   = 24 * time.Hour
)

// Init initializes the auth package with password from environment
func Init() {
	password = os.Getenv("GAGOS_PASSWORD")
	if password == "" {
		log.Warn().Msg("GAGOS_PASSWORD not set - authentication disabled")
	} else {
		log.Info().Msg("Authentication enabled")
	}
}

// IsEnabled returns true if authentication is configured
func IsEnabled() bool {
	return password != ""
}

// GenerateToken creates a cryptographically secure random token
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidatePassword checks if the provided password matches using constant-time comparison
func ValidatePassword(input string) bool {
	if password == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(input), []byte(password)) == 1
}

// CreateSession creates a new session and returns the token
func CreateSession() string {
	token := GenerateToken()
	sessionMutex.Lock()
	sessions[token] = time.Now().Add(sessionTTL)
	sessionMutex.Unlock()
	log.Debug().Str("token_prefix", token[:8]).Msg("Session created")
	return token
}

// ValidateSession checks if a session token is valid and not expired
func ValidateSession(token string) bool {
	if token == "" {
		return false
	}
	sessionMutex.RLock()
	expiry, exists := sessions[token]
	sessionMutex.RUnlock()
	return exists && time.Now().Before(expiry)
}

// DeleteSession removes a session
func DeleteSession(token string) {
	sessionMutex.Lock()
	delete(sessions, token)
	sessionMutex.Unlock()
}

// Middleware returns a Fiber middleware that enforces authentication
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If auth is disabled, allow all requests
		if !IsEnabled() {
			return c.Next()
		}

		path := c.Path()

		// Allow public endpoints (health checks, login, runtime info)
		publicPaths := []string{
			"/api/health",
			"/api/version",
			"/api/runtime",
			"/login",
			"/api/auth/login",
		}

		for _, p := range publicPaths {
			if path == p {
				return c.Next()
			}
		}

		// Allow static assets for login page
		if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".png") {
			return c.Next()
		}

		// Check session cookie
		token := c.Cookies("gagos_session")
		if ValidateSession(token) {
			return c.Next()
		}

		// For API requests, return 401 JSON
		if strings.HasPrefix(path, "/api/") || c.Get("Accept") == "application/json" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// For browser requests, redirect to login
		return c.Redirect("/login")
	}
}

// CleanupExpiredSessions removes expired sessions (call periodically)
func CleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	now := time.Now()
	for token, expiry := range sessions {
		if now.After(expiry) {
			delete(sessions, token)
		}
	}
}
