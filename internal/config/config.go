package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"time"
)

// Errors
var (
	ErrMissingOIDCIssuer       = errors.New("OIDC_ISSUER_URL is required")
	ErrMissingOIDCClientID     = errors.New("OIDC_CLIENT_ID is required")
	ErrMissingOIDCClientSecret = errors.New("OIDC_CLIENT_SECRET is required")
	ErrMissingOIDCRedirectURI  = errors.New("OIDC_REDIRECT_URI is required")
	ErrMissingBackendURL       = errors.New("BACKEND_URL is required")
	ErrMissingFrontendURL      = errors.New("FRONTEND_URL is required")
)

// Config holds all configuration for the application
type Config struct {
	// Server
	ListenAddress string
	Port          string

	// Database
	DatabasePath string

	// OIDC
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
	OIDCScope        string

	// Backend
	BackendURL string

	// Frontend
	FrontendURL string

	// Session
	SessionDuration time.Duration

	// Cookie
	CookieSecret string

	// UI Accent Colors
	UIAccentColorStart string
	UIAccentColorEnd   string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		ListenAddress:      getEnv("LISTEN_ADDRESS", "127.0.0.1"),
		Port:               getEnv("PORT", "8080"),
		DatabasePath:       getEnv("DATABASE_PATH", "./data/languagetool.db"),
		OIDCIssuerURL:      getEnv("OIDC_ISSUER_URL", ""),
		OIDCClientID:       getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:   getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURI:    getEnv("OIDC_REDIRECT_URI", ""),
		OIDCScope:          getEnv("OIDC_SCOPE", "openid profile email"),
		BackendURL:         getEnv("BACKEND_URL", "http://localhost:8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "https://languagetool.example.com"),
		SessionDuration:    time.Duration(getEnvDuration("SESSION_DURATION_HOURS", 24)) * time.Hour,
		CookieSecret:       getEnv("COOKIE_SECRET", generateRandomSecret()),
		UIAccentColorStart: getEnv("UI_ACCENT_COLOR_START", "#667eea"),
		UIAccentColorEnd:   getEnv("UI_ACCENT_COLOR_END", "#764ba2"),
	}
}

// Validate checks if required configuration is present
func (c *Config) Validate() error {
	if c.OIDCIssuerURL == "" {
		return ErrMissingOIDCIssuer
	}
	if c.OIDCClientID == "" {
		return ErrMissingOIDCClientID
	}
	if c.OIDCClientSecret == "" {
		return ErrMissingOIDCClientSecret
	}
	if c.OIDCRedirectURI == "" {
		return ErrMissingOIDCRedirectURI
	}
	if c.BackendURL == "" {
		return ErrMissingBackendURL
	}
	if c.FrontendURL == "" {
		return ErrMissingFrontendURL
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if hours, err := strconv.Atoi(value); err == nil {
			return hours
		}
	}
	return defaultValue
}

func generateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
