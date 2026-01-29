// Package testutil provides test utilities and helpers.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"golinks/internal/db"
)

// TestDB creates a test database connection and returns a cleanup function.
// Uses TEST_DATABASE_URL environment variable or defaults to a test database.
func TestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://golinks:golinks@localhost:5432/golinks_test?sslmode=disable"
	}

	ctx := context.Background()
	database, err := db.New(ctx, connString)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(connString); err != nil {
		database.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		cleanupTestData(ctx, database.Pool)
		database.Close()
	}

	return database, cleanup
}

// cleanupTestData removes all test data from the database.
func cleanupTestData(ctx context.Context, pool *pgxpool.Pool) {
	// Delete in order to respect foreign keys
	pool.Exec(ctx, "DELETE FROM user_links")
	pool.Exec(ctx, "DELETE FROM links")
	pool.Exec(ctx, "DELETE FROM users")
	pool.Exec(ctx, "DELETE FROM organizations")
}

// CreateTestOrg creates a test organization and returns it.
func CreateTestOrg(t *testing.T, database *db.DB, name, slug string) *db.DB {
	t.Helper()
	ctx := context.Background()

	_, err := database.Pool.Exec(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		ON CONFLICT (slug) DO NOTHING
	`, name, slug)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	return database
}

// CreateTestUser creates a test user and returns the user ID.
func CreateTestUser(t *testing.T, database *db.DB, sub, email, role string) string {
	t.Helper()
	ctx := context.Background()

	var id string
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO users (sub, email, name, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, sub, email, fmt.Sprintf("Test User %s", sub), role).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return id
}

// CreateTestLink creates a test link and returns the link ID.
func CreateTestLink(t *testing.T, database *db.DB, keyword, url, scope, status string) string {
	t.Helper()
	ctx := context.Background()

	var id string
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO links (keyword, url, description, scope, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (keyword) DO UPDATE SET url = EXCLUDED.url
		RETURNING id
	`, keyword, url, "Test link", scope, status).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test link: %v", err)
	}

	return id
}
