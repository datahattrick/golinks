package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/handlers"
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
				"Title":   "Error",
				"Message": message,
			})
		},
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Session middleware and store
	sessionMiddleware, sessionStore := session.NewWithStore(session.Config{
		CookieSecure:   false, // Set true in production with HTTPS
		CookieHTTPOnly: true,
	})
	app.Use(sessionMiddleware)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(sessionStore, database)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(database)
	redirectHandler := handlers.NewRedirectHandler(database)
	profileHandler := handlers.NewProfileHandler(database)

	// Static files
	app.Get("/static/*", static.New("./static"))

	// Auth routes - only initialize if OIDC is configured
	if cfg.OIDCIssuer != "" {
		authHandler, err := handlers.NewAuthHandler(ctx, cfg, sessionStore, database)
		if err != nil {
			log.Printf("Warning: Failed to initialize OIDC auth: %v", err)
			log.Println("OIDC authentication is disabled. Set OIDC_* environment variables to enable.")
		} else {
			app.Get("/auth/login", authHandler.Login)
			app.Get("/auth/callback", authHandler.Callback)
			app.Get("/auth/logout", authHandler.Logout)
		}
	} else {
		log.Println("OIDC authentication is disabled. Set OIDC_ISSUER to enable.")
	}

	// Login page (always available)
	app.Get("/login", func(c fiber.Ctx) error {
		return c.Render("login", fiber.Map{})
	})

	// Protected routes
	app.Post("/links", authMiddleware.RequireAuth, linkHandler.Create)
	app.Delete("/links/:id", authMiddleware.RequireAuth, linkHandler.Delete)
	app.Get("/profile", authMiddleware.RequireAuth, profileHandler.Show)

	// Public routes with optional auth
	app.Get("/", authMiddleware.OptionalAuth, linkHandler.Index)
	app.Get("/search", authMiddleware.OptionalAuth, linkHandler.Search)

	// Redirect routes - must be last (catch-all for keywords)
	app.Get("/go/:keyword", redirectHandler.Redirect)
	app.Get("/:keyword", redirectHandler.Redirect)

	// Graceful shutdown
	go func() {
		if err := app.Listen(cfg.ServerAddr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("Server started on %s", cfg.ServerAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
