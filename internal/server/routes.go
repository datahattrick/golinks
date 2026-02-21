package server

import (
	"context"
	"log/slog"
	"os"

	"golinks/internal/db"
	"golinks/internal/email"
	"golinks/internal/handlers"
	"golinks/internal/handlers/api"
	"golinks/internal/metrics"
	"golinks/internal/middleware"
)

// RegisterRoutes registers all application routes.
func (s *Server) RegisterRoutes(ctx context.Context, database *db.DB) error {
	// Initialize Prometheus metrics collector
	metrics.Init(database)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(database, s.Cfg)

	// Initialize email notifier
	notifier := email.NewNotifier(s.Cfg, database)
	handlers.SetNotifier(notifier)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(database, s.Cfg)
	redirectHandler := handlers.NewRedirectHandler(database, s.Cfg)
	profileHandler := handlers.NewProfileHandler(database, s.Cfg)
	userLinkHandler := handlers.NewUserLinkHandler(database, s.Cfg)
	moderationHandler := handlers.NewModerationHandler(database, s.Cfg, notifier)
	manageHandler := handlers.NewManageHandler(database, s.Cfg)
	healthHandler := handlers.NewHealthHandler(database)
	userHandler := handlers.NewUserHandler(database, s.Cfg)

	// Kubernetes probe endpoints (no auth required)
	probeHandler := handlers.NewProbeHandler(database)
	s.App.Get("/healthz", probeHandler.Liveness)
	s.App.Get("/readyz", probeHandler.Readiness)

	// Auth routes - OIDC is always required for frontend access
	if s.Cfg.OIDCIssuer == "" {
		slog.Error("OIDC_ISSUER is required, all users must be authenticated")
		os.Exit(1)
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
	s.App.Get("/links/:id/suggest-edit", authMiddleware.RequireAuth, linkHandler.SuggestEdit)
	s.App.Post("/links/:id/suggest-edit", authMiddleware.RequireAuth, linkHandler.SubmitSuggestEdit)
	s.App.Delete("/links/:id", authMiddleware.RequireAuth, linkHandler.Delete)
	s.App.Get("/profile", authMiddleware.RequireAuth, profileHandler.Show)
	s.App.Patch("/profile/fallback", authMiddleware.RequireAuth, profileHandler.UpdateFallbackPreference)

	// Pending submissions count badge (available regardless of personal links setting)
	s.App.Get("/my-links/pending-count", authMiddleware.RequireAuth, userLinkHandler.PendingCount)

	// User link override routes (only if personal links enabled)
	if s.Cfg.EnablePersonalLinks {
		s.App.Get("/my-links", authMiddleware.RequireAuth, userLinkHandler.List)
		s.App.Post("/my-links", authMiddleware.RequireAuth, userLinkHandler.Create)

		// Shared link routes (must be before /my-links/:id to avoid parameter capture)
		sharedLinkHandler := handlers.NewSharedLinkHandler(database, s.Cfg)
		s.App.Get("/my-links/users/search", authMiddleware.RequireAuth, sharedLinkHandler.SearchUsers)
		s.App.Post("/my-links/share", authMiddleware.RequireAuth, sharedLinkHandler.Create)
		s.App.Post("/my-links/share/:id/accept", authMiddleware.RequireAuth, sharedLinkHandler.Accept)
		s.App.Delete("/my-links/share/:id", authMiddleware.RequireAuth, sharedLinkHandler.Decline)
		s.App.Delete("/my-links/share/:id/withdraw", authMiddleware.RequireAuth, sharedLinkHandler.Withdraw)

		s.App.Get("/my-links/:id/edit", authMiddleware.RequireAuth, userLinkHandler.Edit)
		s.App.Put("/my-links/:id", authMiddleware.RequireAuth, userLinkHandler.Update)
		s.App.Delete("/my-links/:id", authMiddleware.RequireAuth, userLinkHandler.Delete)
	}

	// Moderation routes (moderators only — role checks in handlers)
	s.App.Get("/moderation", authMiddleware.RequireAuth, moderationHandler.Index)
	s.App.Post("/moderation/:id/approve", authMiddleware.RequireAuth, moderationHandler.Approve)
	s.App.Post("/moderation/:id/reject", authMiddleware.RequireAuth, moderationHandler.Reject)
	s.App.Post("/moderation/:id/approve-deletion", authMiddleware.RequireAuth, moderationHandler.ApproveDeletion)
	s.App.Post("/moderation/:id/reject-deletion", authMiddleware.RequireAuth, moderationHandler.RejectDeletion)
	s.App.Post("/moderation/edit/:id/approve", authMiddleware.RequireAuth, moderationHandler.ApproveEdit)
	s.App.Post("/moderation/edit/:id/reject", authMiddleware.RequireAuth, moderationHandler.RejectEdit)

	// Management routes (all authenticated users — role checks in handlers)
	s.App.Get("/manage", authMiddleware.RequireAuth, manageHandler.Index)
	s.App.Get("/manage/:id/edit", authMiddleware.RequireAuth, manageHandler.Edit)
	s.App.Put("/manage/:id", authMiddleware.RequireAuth, manageHandler.Update)
	s.App.Post("/manage/:id/edit-request", authMiddleware.RequireAuth, manageHandler.RequestEdit)
	s.App.Post("/manage/:id/request-deletion", authMiddleware.RequireAuth, manageHandler.RequestDeletion)
	s.App.Post("/health/:id", authMiddleware.RequireAuth, healthHandler.CheckLink)

	// Admin routes (admin only)
	s.App.Get("/admin/users", authMiddleware.RequireAuth, userHandler.ListUsers)
	s.App.Post("/admin/users/:id/role", authMiddleware.RequireAuth, userHandler.UpdateUserRole)
	s.App.Post("/admin/users/:id/org", authMiddleware.RequireAuth, userHandler.UpdateUserOrg)
	s.App.Delete("/admin/users/:id", authMiddleware.RequireAuth, userHandler.DeleteUser)

	// Admin fallback redirect management
	fallbackHandler := handlers.NewFallbackRedirectHandler(database, s.Cfg)
	s.App.Get("/admin/fallback-redirects", authMiddleware.RequireAuth, fallbackHandler.List)
	s.App.Post("/admin/fallback-redirects", authMiddleware.RequireAuth, fallbackHandler.Create)
	s.App.Put("/admin/fallback-redirects/:id", authMiddleware.RequireAuth, fallbackHandler.Update)
	s.App.Delete("/admin/fallback-redirects/:id", authMiddleware.RequireAuth, fallbackHandler.Delete)

	// Random link route ("I'm Feeling Lucky")
	s.App.Get("/random", authMiddleware.RequireAuth, redirectHandler.Random)

	// Redirect API routes - auth depends on mode
	// Only /go/:keyword is used; the old /:keyword catch-all was removed because
	// it shadowed real endpoints (any route name became an unreachable keyword).
	if s.Cfg.IsSimpleMode() {
		slog.Info("running in simple mode, redirect API does not require authentication")
		s.App.Get("/go/:keyword", authMiddleware.OptionalAuth, redirectHandler.Redirect)
	} else {
		// Full mode - redirect routes require auth for personal/org resolution
		s.App.Get("/go/:keyword", authMiddleware.RequireAuth, redirectHandler.Redirect)
	}

	// --- JSON API v1 routes ---
	apiLinkHandler := api.NewLinkHandler(database, s.Cfg, notifier)
	apiResolveHandler := api.NewResolveHandler(database, s.Cfg)
	apiUserHandler := api.NewUserHandler(database, s.Cfg)
	apiModerationHandler := api.NewModerationHandler(database, s.Cfg, notifier)
	apiHealthHandler := api.NewHealthHandler(database)

	// Link management API
	s.App.Get("/api/v1/links", authMiddleware.RequireAuth, apiLinkHandler.List)
	s.App.Post("/api/v1/links", authMiddleware.RequireAuth, apiLinkHandler.Create)
	s.App.Get("/api/v1/links/check/:keyword", authMiddleware.RequireAuth, apiLinkHandler.CheckKeyword)
	s.App.Get("/api/v1/links/:id", authMiddleware.RequireAuth, apiLinkHandler.Get)
	s.App.Put("/api/v1/links/:id", authMiddleware.RequireAuth, apiLinkHandler.Update)
	s.App.Delete("/api/v1/links/:id", authMiddleware.RequireAuth, apiLinkHandler.Delete)

	// Keyword resolution API - auth depends on mode
	if s.Cfg.IsSimpleMode() {
		s.App.Get("/api/v1/resolve/:keyword", authMiddleware.OptionalAuth, apiResolveHandler.Resolve)
	} else {
		s.App.Get("/api/v1/resolve/:keyword", authMiddleware.RequireAuth, apiResolveHandler.Resolve)
	}

	// User management API (admin checks enforced in handlers)
	s.App.Get("/api/v1/users", authMiddleware.RequireAuth, apiUserHandler.List)
	s.App.Put("/api/v1/users/:id/role", authMiddleware.RequireAuth, apiUserHandler.UpdateRole)
	s.App.Put("/api/v1/users/:id/org", authMiddleware.RequireAuth, apiUserHandler.UpdateOrg)
	s.App.Delete("/api/v1/users/:id", authMiddleware.RequireAuth, apiUserHandler.Delete)

	// Moderation API (moderator checks enforced in handlers)
	s.App.Get("/api/v1/moderation/pending", authMiddleware.RequireAuth, apiModerationHandler.ListPending)
	s.App.Post("/api/v1/moderation/:id/approve", authMiddleware.RequireAuth, apiModerationHandler.Approve)
	s.App.Post("/api/v1/moderation/:id/reject", authMiddleware.RequireAuth, apiModerationHandler.Reject)

	// Health check API (moderator checks enforced in handler)
	s.App.Post("/api/v1/health/:id", authMiddleware.RequireAuth, apiHealthHandler.CheckLink)

	return nil
}
