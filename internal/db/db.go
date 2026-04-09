package db

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"golinks/migrations"
)

// DB wraps a pgxpool connection pool.
type DB struct {
	Pool *pgxpool.Pool
	buf  *writeBuffer
}

// New creates a new database connection pool with explicit sizing and lifecycle settings.
func New(ctx context.Context, connString string, maxConns, minConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	cfg.MaxConns          = maxConns
	cfg.MinConns          = minConns
	cfg.MaxConnLifetime   = 30 * time.Minute
	cfg.MaxConnIdleTime   = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool, buf: newWriteBuffer()}, nil
}

// RunMigrations runs all embedded SQL migrations.
func (d *DB) RunMigrations(connString string) error {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, connString)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// Close closes the connection pool.
func (d *DB) Close() {
	d.Pool.Close()
}

// Ping checks database connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}

// SeedDevLinks inserts test links for development. Skips links that already exist.
func (d *DB) SeedDevLinks(ctx context.Context) error {
	links := []struct {
		keyword     string
		url         string
		description string
	}{
		{"google", "https://www.google.com", "Google Search"},
		{"example", "https://example.org", "Example Domain"},
		{"github", "https://github.com", "GitHub"},
		{"go", "https://go.dev", "Go Programming Language"},
		{"docs", "https://pkg.go.dev", "Go Package Documentation"},
	}

	query := `
		INSERT INTO links (keyword, url, description, scope, status)
		VALUES ($1, $2, $3, 'global', 'approved')
		ON CONFLICT (keyword) WHERE scope = 'global' DO NOTHING
	`

	for _, link := range links {
		if _, err := d.Pool.Exec(ctx, query, link.keyword, link.url, link.description); err != nil {
			return fmt.Errorf("failed to seed link %s: %w", link.keyword, err)
		}
	}

	return nil
}
