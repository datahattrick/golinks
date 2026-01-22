package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"golinks/internal/db"
)

// AuthMiddleware handles user authentication via sessions.
type AuthMiddleware struct {
	store *session.Store
	db    *db.DB
}

// NewAuthMiddleware creates a new auth middleware instance.
func NewAuthMiddleware(store *session.Store, db *db.DB) *AuthMiddleware {
	return &AuthMiddleware{store: store, db: db}
}

// RequireAuth ensures the user is authenticated, redirecting to /login if not.
func (m *AuthMiddleware) RequireAuth(c fiber.Ctx) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	userSub := sess.Get("user_sub")
	if userSub == nil {
		return c.Redirect().To("/login")
	}

	user, err := m.db.GetUserBySub(c.Context(), userSub.(string))
	if err != nil {
		sess.Destroy()
		return c.Redirect().To("/login")
	}

	c.Locals("user", user)
	return c.Next()
}

// OptionalAuth loads the user if authenticated, but doesn't require authentication.
func (m *AuthMiddleware) OptionalAuth(c fiber.Ctx) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return c.Next()
	}

	userSub := sess.Get("user_sub")
	if userSub == nil {
		return c.Next()
	}

	user, err := m.db.GetUserBySub(c.Context(), userSub.(string))
	if err == nil {
		c.Locals("user", user)
	}

	return c.Next()
}
