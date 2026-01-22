package config

import (
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	ServerAddr string
	BaseURL    string

	// Database
	DatabaseURL string

	// OIDC
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string

	// Session
	SessionSecret string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerAddr:       getEnv("SERVER_ADDR", ":3000"),
		BaseURL:          getEnv("BASE_URL", "http://localhost:3000"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://localhost:5432/golinks?sslmode=disable"),
		OIDCIssuer:       getEnv("OIDC_ISSUER", ""),
		OIDCClientID:     getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  getEnv("OIDC_REDIRECT_URL", "http://localhost:3000/auth/callback"),
		SessionSecret:    getEnv("SESSION_SECRET", "change-me-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
