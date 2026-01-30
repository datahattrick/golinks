package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/handlers"
	"golinks/internal/jobs"
	"golinks/internal/middleware"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	// Initialize database
	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// Sync organization fallback URLs from config
	if len(cfg.OrgFallbacks) > 0 {
		if err := database.SyncOrgFallbackURLs(ctx, cfg.OrgFallbacks); err != nil {
			log.Printf("Warning: Failed to sync org fallback URLs: %v", err)
		} else {
			log.Printf("Synced %d organization fallback URL(s)", len(cfg.OrgFallbacks))
		}
	}

	// Seed dev links in development mode
	if cfg.IsDev() {
		if err := database.SeedDevLinks(ctx); err != nil {
			log.Printf("Warning: Failed to seed dev links: %v", err)
		} else {
			log.Println("Development seed links loaded")
		}
	}

	// Setup template engine
	engine := html.New("./views", ".html")
	engine.Reload(true) // Set to false in production

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Internal Server Error"

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}

			return c.Status(code).Render("error", fiber.Map{
				"Title":       "Error",
				"Message":     message,
				"SiteTitle":   cfg.SiteTitle,
				"SiteTagline": cfg.SiteTagline,
				"SiteFooter":  cfg.SiteFooter,
				"SiteLogoURL": cfg.SiteLogoURL,
			})
		},
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// CORS middleware
	corsOrigins := cfg.BaseURL // Default to same origin
	if cfg.CORSOrigins != "" {
		corsOrigins = cfg.CORSOrigins
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "HX-Request", "HX-Current-URL", "HX-Target"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Cookie encryption middleware (encrypt all cookies)
	encryptionKey := deriveEncryptionKey(cfg.SessionSecret)
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: encryptionKey,
	}))

	// Session middleware with signed and secure cookies
	sessionMiddleware, _ := session.NewWithStore(session.Config{
		CookieSecure:   cfg.TLSEnabled || !cfg.IsDev(),
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	app.Use(sessionMiddleware)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(database, cfg)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(database, cfg)
	redirectHandler := handlers.NewRedirectHandler(database, cfg)
	profileHandler := handlers.NewProfileHandler(database, cfg)
	userLinkHandler := handlers.NewUserLinkHandler(database, cfg)
	moderationHandler := handlers.NewModerationHandler(database, cfg)
	manageHandler := handlers.NewManageHandler(database, cfg)
	healthHandler := handlers.NewHealthHandler(database)
	userHandler := handlers.NewUserHandler(database, cfg)

	// Start background health checker
	healthChecker := jobs.NewHealthChecker(database, 1*time.Hour, 24*time.Hour)
	go healthChecker.Start(ctx)

	// Static files
	app.Get("/static/*", static.New("./static"))

	// Auth routes - OIDC must be configured for this app
	if cfg.OIDCIssuer == "" {
		log.Fatal("OIDC_ISSUER is required. All users must be authenticated.")
	}

	authHandler, err := handlers.NewAuthHandler(ctx, cfg, database)
	if err != nil {
		log.Fatalf("Failed to initialize OIDC auth: %v", err)
	}

	app.Get("/auth/login", authHandler.Login)
	app.Get("/auth/callback", authHandler.Callback)
	app.Get("/auth/logout", authHandler.Logout)

	// All routes require authentication
	app.Get("/", authMiddleware.RequireAuth, linkHandler.Index)
	app.Get("/search", authMiddleware.RequireAuth, linkHandler.Search)
	app.Get("/suggest", authMiddleware.RequireAuth, linkHandler.Suggest)
	app.Get("/browse", authMiddleware.RequireAuth, linkHandler.Browse)
	app.Get("/new", authMiddleware.RequireAuth, linkHandler.New)
	app.Get("/links/check", authMiddleware.RequireAuth, linkHandler.CheckKeyword)
	app.Post("/links", authMiddleware.RequireAuth, linkHandler.Create)
	app.Delete("/links/:id", authMiddleware.RequireAuth, linkHandler.Delete)
	app.Get("/profile", authMiddleware.RequireAuth, profileHandler.Show)

	// User link override routes
	app.Get("/my-links", authMiddleware.RequireAuth, userLinkHandler.List)
	app.Post("/my-links", authMiddleware.RequireAuth, userLinkHandler.Create)
	app.Delete("/my-links/:id", authMiddleware.RequireAuth, userLinkHandler.Delete)

	// Moderation routes (moderators only)
	app.Get("/moderation", authMiddleware.RequireAuth, moderationHandler.Index)
	app.Post("/moderation/:id/approve", authMiddleware.RequireAuth, moderationHandler.Approve)
	app.Post("/moderation/:id/reject", authMiddleware.RequireAuth, moderationHandler.Reject)

	// Management routes (moderators only)
	app.Get("/manage", authMiddleware.RequireAuth, manageHandler.Index)
	app.Get("/manage/:id/edit", authMiddleware.RequireAuth, manageHandler.Edit)
	app.Put("/manage/:id", authMiddleware.RequireAuth, manageHandler.Update)
	app.Post("/health/:id", authMiddleware.RequireAuth, healthHandler.CheckLink)

	// Admin routes (admin only)
	app.Get("/admin/users", authMiddleware.RequireAuth, userHandler.ListUsers)
	app.Post("/admin/users/:id/role", authMiddleware.RequireAuth, userHandler.UpdateUserRole)
	app.Post("/admin/users/:id/org", authMiddleware.RequireAuth, userHandler.UpdateUserOrg)
	app.Delete("/admin/users/:id", authMiddleware.RequireAuth, userHandler.DeleteUser)

	// Random link route ("I'm Feeling Lucky")
	app.Get("/random", authMiddleware.RequireAuth, redirectHandler.Random)

	// Redirect routes - also require auth (catch-all for keywords)
	app.Get("/go/:keyword", authMiddleware.RequireAuth, redirectHandler.Redirect)
	app.Get("/:keyword", authMiddleware.RequireAuth, redirectHandler.Redirect)

	// Start server
	go func() {
		var listenErr error
		if cfg.TLSEnabled {
			tlsConfig := buildTLSConfig(cfg)
			listenConfig := fiber.ListenConfig{
				CertFile:      cfg.TLSCertFile,
				CertKeyFile:   cfg.TLSKeyFile,
				TLSConfigFunc: func(tc *tls.Config) { *tc = *tlsConfig },
			}
			if cfg.TLSCAFile != "" {
				log.Printf("Starting server with mTLS on %s", cfg.ServerAddr)
			} else {
				log.Printf("Starting server with TLS on %s", cfg.ServerAddr)
			}
			listenErr = app.Listen(cfg.ServerAddr, listenConfig)
		} else {
			listenErr = app.Listen(cfg.ServerAddr)
		}
		if listenErr != nil {
			log.Printf("Server error: %v", listenErr)
		}
	}()

	log.Printf("Server started on %s", cfg.ServerAddr)
	if cfg.ClientCertHeader != "" {
		log.Printf("Accepting client cert via header: %s", cfg.ClientCertHeader)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

// deriveEncryptionKey derives a 32-byte encryption key from the session secret.
// Uses SHA-256 to ensure consistent key length for AES-256.
// Returns base64-encoded string as required by encryptcookie middleware.
func deriveEncryptionKey(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// buildTLSConfig creates a TLS config for mTLS if CA file is provided.
func buildTLSConfig(cfg *config.Config) *tls.Config {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.TLSCAFile != "" {
		caCert, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			log.Fatalf("Failed to read CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			log.Fatal("Failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
