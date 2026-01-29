package config

import (
	"os"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Environment
	Env string // "development", "production", etc.

	// Server
	ServerAddr string
	BaseURL    string

	// Database
	DatabaseURL string

	// TLS/mTLS
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string // CA for verifying client certs (mTLS)

	// Client cert via header (for ingress-terminated TLS)
	ClientCertHeader string // Header name containing client cert CN, e.g. "X-Client-CN"

	// OIDC
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string

	// Session
	SessionSecret string // Used for signing cookies (min 32 chars)

	// CORS
	CORSOrigins string // Comma-separated allowed origins, e.g. "https://example.com,https://app.example.com"

	// Features
	EnableRandomKeywords bool // Enable random keywords section and "I'm Feeling Lucky" feature

	// Site Branding
	SiteTitle   string // env: SITE_TITLE, default: "GoLinks"
	SiteTagline string // env: SITE_TAGLINE, default: "Fast URL shortcuts for your team"
	SiteFooter  string // env: SITE_FOOTER, default: "GoLinks - Fast URL shortcuts for your team"
	SiteLogoURL string // env: SITE_LOGO_URL, default: "" (no logo, text only)
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Env:              getEnv("ENV", "development"),
		ServerAddr:       getEnv("SERVER_ADDR", ":3000"),
		BaseURL:          getEnv("BASE_URL", "http://localhost:3000"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://localhost:5432/golinks?sslmode=disable"),
		TLSEnabled:       getEnv("TLS_ENABLED", "") != "",
		TLSCertFile:      getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:       getEnv("TLS_KEY_FILE", ""),
		TLSCAFile:        getEnv("TLS_CA_FILE", ""),
		ClientCertHeader: getEnv("CLIENT_CERT_HEADER", ""),
		OIDCIssuer:       getEnv("OIDC_ISSUER", ""),
		OIDCClientID:     getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  getEnv("OIDC_REDIRECT_URL", "http://localhost:3000/auth/callback"),
		SessionSecret:    getEnv("SESSION_SECRET", "change-me-in-production-min-32-chars"),
		CORSOrigins:          getEnv("CORS_ORIGINS", ""),
		EnableRandomKeywords: getEnv("ENABLE_RANDOM_KEYWORDS", "") != "",

		SiteTitle:   getEnv("SITE_TITLE", "GoLinks"),
		SiteTagline: getEnv("SITE_TAGLINE", "Fast URL shortcuts for your team"),
		SiteFooter:  getEnv("SITE_FOOTER", "GoLinks - Fast URL shortcuts for your team"),
		SiteLogoURL: getEnv("SITE_LOGO_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// IsDev returns true if the environment is set to development.
func (c *Config) IsDev() bool {
	return c.Env == "development" || c.Env == "dev"
}

// IsMTLSEnabled returns true if mTLS is configured with a CA file.
func (c *Config) IsMTLSEnabled() bool {
	return c.TLSEnabled && c.TLSCAFile != ""
}
