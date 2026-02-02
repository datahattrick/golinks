package config

import (
	"os"
	"strconv"
	"strings"
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
	OIDCOrgClaim     string // OIDC claim name for organization, e.g. "org", "organization", "tenant"

	// Session
	SessionSecret string // Used for signing cookies (min 32 chars)

	// CORS
	CORSOrigins string // Comma-separated allowed origins, e.g. "https://example.com,https://app.example.com"

	// Features
	EnableRandomKeywords bool // Enable random keywords section and "I'm Feeling Lucky" feature
	EnablePersonalLinks  bool // Enable personal link scopes (requires auth)
	EnableOrgLinks       bool // Enable organization link scopes (requires auth)

	// Organizations
	OrgFallbacks map[string]string // Map of org slug to fallback redirect URL, e.g. {"org1": "https://other.com/go/"}

	// Site Branding
	SiteTitle   string // env: SITE_TITLE, default: "GoLinks"
	SiteTagline string // env: SITE_TAGLINE, default: "Fast URL shortcuts for your team"
	SiteFooter  string // env: SITE_FOOTER, default: "GoLinks - Fast URL shortcuts for your team"
	SiteLogoURL string // env: SITE_LOGO_URL, default: "" (no logo, text only)

	// SMTP Email Configuration
	SMTPEnabled  bool   // Enable email notifications
	SMTPHost     string // SMTP server hostname
	SMTPPort     int    // SMTP server port (25, 465, 587)
	SMTPUsername string // SMTP authentication username
	SMTPPassword string // SMTP authentication password
	SMTPFrom     string // From email address
	SMTPFromName string // From display name
	SMTPTLS      string // TLS mode: "none", "starttls", "tls"

	// Email Notification Settings
	EmailNotifyModeratorsOnSubmit  bool // Notify moderators when a link is submitted for review
	EmailNotifyUserOnApproval      bool // Notify user when their link is approved
	EmailNotifyUserOnRejection     bool // Notify user when their link is rejected
	EmailNotifyUserOnDeletion      bool // Notify user when their link is deleted
	EmailNotifyModsOnHealthFailure bool // Notify moderators when health checks fail
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
		OIDCOrgClaim:     getEnv("OIDC_ORG_CLAIM", "organisation"), // OIDC claim name for organization
		SessionSecret:    getEnv("SESSION_SECRET", "change-me-in-production-min-32-chars"),
		CORSOrigins:          getEnv("CORS_ORIGINS", ""),
		EnableRandomKeywords: getEnv("ENABLE_RANDOM_KEYWORDS", "") != "",
		EnablePersonalLinks:  getEnv("ENABLE_PERSONAL_LINKS", "true") != "false",
		EnableOrgLinks:       getEnv("ENABLE_ORG_LINKS", "true") != "false",
		OrgFallbacks:         parseOrgFallbacks(getEnv("ORG_FALLBACKS", "")),

		SiteTitle:   getEnv("SITE_TITLE", "GoLinks"),
		SiteTagline: getEnv("SITE_TAGLINE", "Fast URL shortcuts for your team"),
		SiteFooter:  getEnv("SITE_FOOTER", "GoLinks - Fast URL shortcuts for your team"),
		SiteLogoURL: getEnv("SITE_LOGO_URL", ""),

		// SMTP Configuration
		SMTPEnabled:  getEnv("SMTP_ENABLED", "") != "",
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "GoLinks"),
		SMTPTLS:      getEnv("SMTP_TLS", "starttls"), // none, starttls, tls

		// Email Notification Settings (all enabled by default when SMTP is configured)
		EmailNotifyModeratorsOnSubmit:  getEnv("EMAIL_NOTIFY_MODS_ON_SUBMIT", "true") != "false",
		EmailNotifyUserOnApproval:      getEnv("EMAIL_NOTIFY_USER_ON_APPROVAL", "true") != "false",
		EmailNotifyUserOnRejection:     getEnv("EMAIL_NOTIFY_USER_ON_REJECTION", "true") != "false",
		EmailNotifyUserOnDeletion:      getEnv("EMAIL_NOTIFY_USER_ON_DELETION", "true") != "false",
		EmailNotifyModsOnHealthFailure: getEnv("EMAIL_NOTIFY_MODS_ON_HEALTH_FAILURE", "true") != "false",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
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

// IsSimpleMode returns true if both personal and org links are disabled.
// In simple mode, only global links are used and the redirect API doesn't require authentication.
func (c *Config) IsSimpleMode() bool {
	return !c.EnablePersonalLinks && !c.EnableOrgLinks
}

// IsEmailEnabled returns true if SMTP is configured and enabled.
func (c *Config) IsEmailEnabled() bool {
	return c.SMTPEnabled && c.SMTPHost != "" && c.SMTPFrom != ""
}

// parseOrgFallbacks parses ORG_FALLBACKS env var format: "org1=https://url1/go/,org2=https://url2/"
func parseOrgFallbacks(val string) map[string]string {
	result := make(map[string]string)
	if val == "" {
		return result
	}

	pairs := strings.Split(val, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			slug := strings.TrimSpace(parts[0])
			url := strings.TrimSpace(parts[1])
			if slug != "" && url != "" {
				result[slug] = url
			}
		}
	}
	return result
}
