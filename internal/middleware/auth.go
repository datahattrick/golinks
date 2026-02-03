package middleware

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// AuthMiddleware handles user authentication via sessions and PKI.
type AuthMiddleware struct {
	db               *db.DB
	clientCertHeader string
}

// NewAuthMiddleware creates a new auth middleware instance.
func NewAuthMiddleware(db *db.DB, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		db:               db,
		clientCertHeader: cfg.ClientCertHeader,
	}
}

// RequireAuth ensures the user is authenticated via session or PKI cert.
// Priority: 1) PKI cert (mTLS or header), 2) Session (OIDC)
func (m *AuthMiddleware) RequireAuth(c fiber.Ctx) error {
	// Try PKI authentication first (mTLS or header)
	if user, err := m.authenticateViaPKI(c); err == nil && user != nil {
		m.loadGroupMemberships(c, user)
		c.Locals("user", user)
		return c.Next()
	}

	// Fall back to session-based auth (OIDC)
	sess := session.FromContext(c)
	if sess == nil {
		return m.redirectToLogin(c, nil)
	}

	userSub := sess.Get("user_sub")
	if userSub == nil {
		return m.redirectToLogin(c, sess)
	}

	user, err := m.db.GetUserBySub(c.Context(), userSub.(string))
	if err != nil {
		sess.Destroy()
		return m.redirectToLogin(c, nil)
	}

	m.loadGroupMemberships(c, user)
	c.Locals("user", user)
	return c.Next()
}

// redirectToLogin saves the current URL and redirects to login.
// For API requests (/api/*), returns a 401 JSON error instead of redirecting.
func (m *AuthMiddleware) redirectToLogin(c fiber.Ctx, sess *session.Middleware) error {
	if strings.HasPrefix(c.Path(), "/api/") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error":  "authentication required",
		})
	}

	// Store the original URL for redirect after login
	originalURL := c.OriginalURL()
	if sess == nil {
		sess = session.FromContext(c)
	}
	if sess != nil && originalURL != "" && originalURL != "/auth/login" && originalURL != "/auth/callback" {
		sess.Set("redirect_after_login", originalURL)
	}
	return c.Redirect().To("/auth/login")
}

// authenticateViaPKI extracts username from client cert (mTLS or header) and looks up user.
func (m *AuthMiddleware) authenticateViaPKI(c fiber.Ctx) (*models.User, error) {
	username := m.extractUsernameFromCert(c)
	if username == "" {
		return nil, nil
	}

	return m.db.GetUserByUsername(c.Context(), username)
}

// extractUsernameFromCert extracts the username from client certificate CN.
// Supports both mTLS (direct cert) and header-based (ingress-terminated TLS).
// CN format: "Full Name (username)" -> extracts "username"
func (m *AuthMiddleware) extractUsernameFromCert(c fiber.Ctx) string {
	var cn string

	// Try header first (for ingress-terminated TLS)
	if m.clientCertHeader != "" {
		cn = c.Get(m.clientCertHeader)
	}

	// Try mTLS client cert if no header
	if cn == "" {
		// Access the underlying fasthttp request context for TLS state
		tlsState := c.RequestCtx().TLSConnectionState()
		if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
			cn = tlsState.PeerCertificates[0].Subject.CommonName
		}
	}

	if cn == "" {
		return ""
	}

	return extractUsernameFromCN(cn)
}

// extractUsernameFromCN parses username from CN format "Full Name (username)".
// Returns the username in parentheses, or empty string if not found.
func extractUsernameFromCN(cn string) string {
	// Match content within parentheses at the end: "Heath Taylor (heatht)" -> "heatht"
	re := regexp.MustCompile(`\(([^)]+)\)\s*$`)
	matches := re.FindStringSubmatch(cn)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// OptionalAuth loads the user if authenticated, but doesn't require authentication.
func (m *AuthMiddleware) OptionalAuth(c fiber.Ctx) error {
	// Try PKI authentication first
	if user, err := m.authenticateViaPKI(c); err == nil && user != nil {
		m.loadGroupMemberships(c, user)
		c.Locals("user", user)
		return c.Next()
	}

	// Try session-based auth
	sess := session.FromContext(c)
	if sess == nil {
		return c.Next()
	}

	userSub := sess.Get("user_sub")
	if userSub == nil {
		return c.Next()
	}

	user, err := m.db.GetUserBySub(c.Context(), userSub.(string))
	if err == nil {
		m.loadGroupMemberships(c, user)
		c.Locals("user", user)
	}

	return c.Next()
}

// loadGroupMemberships loads the user's group memberships for tier-based resolution.
func (m *AuthMiddleware) loadGroupMemberships(c fiber.Ctx, user *models.User) {
	memberships, err := m.db.GetUserMemberships(c.Context(), user.ID)
	if err == nil {
		user.GroupMemberships = memberships
	}
}
