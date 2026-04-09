package main

import (
	"context"
	"fmt"
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

	// Wait for the database to become available before proceeding.
	// This handles the common Kubernetes race where the app pod starts before
	// the database pod is ready to accept connections.
	database, err := waitForDB(ctx, cfg.DatabaseURL, 60*time.Second, cfg.DBPoolMaxConns, cfg.DBPoolMinConns)
	if err != nil {
		slog.Error("database never became available", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations completed successfully")

	// Sync fallback redirect options from config
	if len(cfg.RedirectFallbacks) > 0 {
		if err := database.SyncFallbackRedirects(ctx, cfg.RedirectFallbacks); err != nil {
			slog.Warn("failed to sync fallback redirects", "error", err)
		} else {
			slog.Info("synced fallback redirects", "count", len(cfg.RedirectFallbacks))
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

	// Start write buffer — batches click counts and keyword lookups to reduce WAL writes
	database.StartWriteBuffer(ctx, 5*time.Second)

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited")
	database.FlushWriteBuffer(shutdownCtx)
}

// waitForDB retries connecting to the database until it succeeds or the timeout
// elapses. It logs each failed attempt so progress is visible in pod logs.
func waitForDB(ctx context.Context, connString string, timeout time.Duration, maxConns, minConns int32) (*db.DB, error) {
	deadline := time.Now().Add(timeout)
	attempt := 0
	for {
		attempt++
		database, err := db.New(ctx, connString, maxConns, minConns)
		if err == nil {
			if attempt > 1 {
				slog.Info("database connection established", "attempt", attempt)
			}
			return database, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out after %s waiting for database: %w", timeout, err)
		}

		slog.Warn("database not ready, retrying in 5s", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
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
