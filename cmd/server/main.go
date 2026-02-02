package main

import (
	"context"
	"log"
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

	// Initialize email notifier
	notifier := email.NewNotifier(cfg, database)
	handlers.SetNotifier(notifier)

	// Create server
	srv := server.New(cfg)

	// Register routes
	if err := srv.RegisterRoutes(ctx, database); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	// Start background health checker
	healthChecker := jobs.NewHealthChecker(database, 1*time.Hour, 24*time.Hour)
	go healthChecker.Start(ctx)

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("Server started on %s", cfg.ServerAddr)
	if cfg.ClientCertHeader != "" {
		log.Printf("Accepting client cert via header: %s", cfg.ClientCertHeader)
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := srv.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
