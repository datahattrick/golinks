package db

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"golinks/migrations"
)

// DB wraps a pgxpool connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool.
func New(ctx context.Context, connString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
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
