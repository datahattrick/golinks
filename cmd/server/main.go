package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/email"
	"golinks/internal/handlers"
	"golinks/internal/jobs"
	"golinks/internal/server"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	// Initialize structured logger
	initLogger(cfg.LogLevel)

	// Validate configuration
	cfg.Validate()

	// Log startup configuration (omit secrets)
	slog.Info("configuration loaded",
		"env", cfg.Env,
		"addr", cfg.ServerAddr,
		"base_url", cfg.BaseURL,
		"tls_enabled", cfg.TLSEnabled,
		"oidc_issuer", cfg.OIDCIssuer,
		"personal_links", cfg.EnablePersonalLinks,
		"org_links", cfg.EnableOrgLinks,
		"simple_mode", cfg.IsSimpleMode(),
		"smtp_enabled", cfg.IsEmailEnabled(),
		"log_level", cfg.LogLevel,
	)

	// Initialize database
	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations completed successfully")

	// Sync organization fallback URLs from config
	if len(cfg.OrgFallbacks) > 0 {
		if err := database.SyncOrgFallbackURLs(ctx, cfg.OrgFallbacks); err != nil {
			slog.Warn("failed to sync org fallback URLs", "error", err)
		} else {
			slog.Info("synced organization fallback URLs", "count", len(cfg.OrgFallbacks))
		}
	}

	// Seed dev links in development mode
	if cfg.IsDev() {
		if err := database.SeedDevLinks(ctx); err != nil {
			slog.Warn("failed to seed dev links", "error", err)
		} else {
			slog.Info("development seed links loaded")
		}
	}

	// Initialize email notifier
	notifier := email.NewNotifier(cfg, database)
	handlers.SetNotifier(notifier)

	// Create server
	srv := server.New(cfg)

	// Register routes
	if err := srv.RegisterRoutes(ctx, database); err != nil {
		slog.Error("failed to register routes", "error", err)
		os.Exit(1)
	}

	// Start background health checker
	healthChecker := jobs.NewHealthChecker(database, 1*time.Hour, 24*time.Hour)
	go healthChecker.Start(ctx)

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	slog.Info("server started", "addr", cfg.ServerAddr)
	if cfg.ClientCertHeader != "" {
		slog.Info("accepting client cert via header", "header", cfg.ClientCertHeader)
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	if err := srv.Shutdown(); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited")
}

// initLogger configures the default slog logger with the given level.
func initLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
