package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"golinks/internal/models"
)

var ErrFallbackRedirectNotFound = errors.New("fallback redirect not found")

// ListFallbackRedirectsByOrg returns all fallback redirect options for an organization.
func (d *DB) ListFallbackRedirectsByOrg(ctx context.Context, orgID uuid.UUID) ([]models.FallbackRedirect, error) {
	query := `
		SELECT id, organization_id, name, url, created_at, updated_at
		FROM fallback_redirects
		WHERE organization_id = $1
		ORDER BY name ASC
	`

	rows, err := d.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirects []models.FallbackRedirect
	for rows.Next() {
		var r models.FallbackRedirect
		if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Name, &r.URL, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		redirects = append(redirects, r)
	}
	return redirects, rows.Err()
}

// GetFallbackRedirectByID retrieves a single fallback redirect by ID.
func (d *DB) GetFallbackRedirectByID(ctx context.Context, id uuid.UUID) (*models.FallbackRedirect, error) {
	query := `
		SELECT id, organization_id, name, url, created_at, updated_at
		FROM fallback_redirects WHERE id = $1
	`

	var r models.FallbackRedirect
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.OrganizationID, &r.Name, &r.URL, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFallbackRedirectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateFallbackRedirect creates a new fallback redirect option.
func (d *DB) CreateFallbackRedirect(ctx context.Context, r *models.FallbackRedirect) error {
	query := `
		INSERT INTO fallback_redirects (organization_id, name, url)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return d.Pool.QueryRow(ctx, query, r.OrganizationID, r.Name, r.URL).Scan(
		&r.ID, &r.CreatedAt, &r.UpdatedAt,
	)
}

// UpdateFallbackRedirect updates an existing fallback redirect option.
func (d *DB) UpdateFallbackRedirect(ctx context.Context, id uuid.UUID, name, url string) error {
	query := `
		UPDATE fallback_redirects SET name = $1, url = $2, updated_at = NOW()
		WHERE id = $3
	`
	tag, err := d.Pool.Exec(ctx, query, name, url, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFallbackRedirectNotFound
	}
	return nil
}

// DeleteFallbackRedirect deletes a fallback redirect option.
// Users referencing it will have their fallback_redirect_id set to NULL (ON DELETE SET NULL).
func (d *DB) DeleteFallbackRedirect(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM fallback_redirects WHERE id = $1`
	tag, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFallbackRedirectNotFound
	}
	return nil
}

// UpdateUserFallback sets or clears a user's fallback redirect preference.
func (d *DB) UpdateUserFallback(ctx context.Context, userID uuid.UUID, fallbackID *uuid.UUID) error {
	query := `UPDATE users SET fallback_redirect_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := d.Pool.Exec(ctx, query, fallbackID, userID)
	return err
}

// SyncFallbackRedirects upserts fallback redirect entries from the REDIRECT_FALLBACKS env var.
// For each org slug → URL mapping, it ensures the org exists and creates/updates a "default" fallback entry.
func (d *DB) SyncFallbackRedirects(ctx context.Context, fallbacks map[string]string) error {
	if len(fallbacks) == 0 {
		return nil
	}

	for slug, fallbackURL := range fallbacks {
		// Ensure org exists
		org, _, err := d.GetOrCreateOrganization(ctx, slug)
		if err != nil {
			return err
		}

		name := org.Name + " default"
		query := `
			INSERT INTO fallback_redirects (organization_id, name, url)
			VALUES ($1, $2, $3)
			ON CONFLICT (organization_id, name) DO UPDATE SET
				url = EXCLUDED.url,
				updated_at = NOW()
		`
		_, err = d.Pool.Exec(ctx, query, org.ID, name, fallbackURL)
		if err != nil {
			return err
		}
	}
	return nil
}
