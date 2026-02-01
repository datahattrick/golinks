package server

import (
	"context"
	"log"

	"golinks/internal/db"
	"golinks/internal/handlers"
	"golinks/internal/middleware"
)

// RegisterRoutes registers all application routes.
func (s *Server) RegisterRoutes(ctx context.Context, database *db.DB) error {
	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(database, s.Cfg)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(database, s.Cfg)
	redirectHandler := handlers.NewRedirectHandler(database, s.Cfg)
	profileHandler := handlers.NewProfileHandler(database, s.Cfg)
	userLinkHandler := handlers.NewUserLinkHandler(database, s.Cfg)
	moderationHandler := handlers.NewModerationHandler(database, s.Cfg)
	manageHandler := handlers.NewManageHandler(database, s.Cfg)
	healthHandler := handlers.NewHealthHandler(database)
	userHandler := handlers.NewUserHandler(database, s.Cfg)

	// Auth routes - OIDC is always required for frontend access
	if s.Cfg.OIDCIssuer == "" {
		log.Fatal("OIDC_ISSUER is required. All users must be authenticated.")
	}

	authHandler, err := handlers.NewAuthHandler(ctx, s.Cfg, database)
	if err != nil {
		return err
	}

	s.App.Get("/auth/login", authHandler.Login)
	s.App.Get("/auth/callback", authHandler.Callback)
	s.App.Get("/auth/logout", authHandler.Logout)

	// Frontend routes - always require authentication
	s.App.Get("/", authMiddleware.RequireAuth, linkHandler.Index)
	s.App.Get("/search", authMiddleware.RequireAuth, linkHandler.Search)
	s.App.Get("/suggest", authMiddleware.RequireAuth, linkHandler.Suggest)
	s.App.Get("/browse", authMiddleware.RequireAuth, linkHandler.Browse)
	s.App.Get("/new", authMiddleware.RequireAuth, linkHandler.New)
	s.App.Get("/links/check", authMiddleware.RequireAuth, linkHandler.CheckKeyword)
	s.App.Post("/links", authMiddleware.RequireAuth, linkHandler.Create)
	s.App.Delete("/links/:id", authMiddleware.RequireAuth, linkHandler.Delete)
	s.App.Get("/profile", authMiddleware.RequireAuth, profileHandler.Show)

	// User link override routes (only if personal links enabled)
	if s.Cfg.EnablePersonalLinks {
		s.App.Get("/my-links", authMiddleware.RequireAuth, userLinkHandler.List)
		s.App.Post("/my-links", authMiddleware.RequireAuth, userLinkHandler.Create)
		s.App.Delete("/my-links/:id", authMiddleware.RequireAuth, userLinkHandler.Delete)
	}

	// Moderation routes (moderators only)
	s.App.Get("/moderation", authMiddleware.RequireAuth, moderationHandler.Index)
	s.App.Post("/moderation/:id/approve", authMiddleware.RequireAuth, moderationHandler.Approve)
	s.App.Post("/moderation/:id/reject", authMiddleware.RequireAuth, moderationHandler.Reject)

	// Management routes (moderators only)
	s.App.Get("/manage", authMiddleware.RequireAuth, manageHandler.Index)
	s.App.Get("/manage/:id/edit", authMiddleware.RequireAuth, manageHandler.Edit)
	s.App.Put("/manage/:id", authMiddleware.RequireAuth, manageHandler.Update)
	s.App.Post("/health/:id", authMiddleware.RequireAuth, healthHandler.CheckLink)

	// Admin routes (admin only)
	s.App.Get("/admin/users", authMiddleware.RequireAuth, userHandler.ListUsers)
	s.App.Post("/admin/users/:id/role", authMiddleware.RequireAuth, userHandler.UpdateUserRole)
	s.App.Post("/admin/users/:id/org", authMiddleware.RequireAuth, userHandler.UpdateUserOrg)
	s.App.Delete("/admin/users/:id", authMiddleware.RequireAuth, userHandler.DeleteUser)

	// Random link route ("I'm Feeling Lucky")
	s.App.Get("/random", authMiddleware.RequireAuth, redirectHandler.Random)

	// Redirect API routes - auth depends on mode
	// In simple mode (no personal/org links), redirect API doesn't require auth
	if s.Cfg.IsSimpleMode() {
		log.Println("Running in simple mode (personal and org links disabled)")
		log.Println("Redirect API (/go/:keyword) does not require authentication")
		s.App.Get("/go/:keyword", authMiddleware.OptionalAuth, redirectHandler.Redirect)
		s.App.Get("/:keyword", authMiddleware.OptionalAuth, redirectHandler.Redirect)
	} else {
		// Full mode - redirect routes require auth for personal/org resolution
		s.App.Get("/go/:keyword", authMiddleware.RequireAuth, redirectHandler.Redirect)
		s.App.Get("/:keyword", authMiddleware.RequireAuth, redirectHandler.Redirect)
	}

	return nil
}
