package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)

var ErrUserLinkNotFound = errors.New("user link not found")

// CreateUserLink creates a new user-specific link override.
func (d *DB) CreateUserLink(ctx context.Context, link *models.UserLink) error {
	query := `
		INSERT INTO user_links (user_id, keyword, url, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, click_count, created_at, updated_at
	`

	err := d.Pool.QueryRow(ctx, query,
		link.UserID,
		link.Keyword,
		link.URL,
		link.Description,
	).Scan(&link.ID, &link.ClickCount, &link.CreatedAt, &link.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateKeyword
		}
		return err
	}

	return nil
}

// GetUserLinkByKeyword retrieves a user's link override for a specific keyword.
func (d *DB) GetUserLinkByKeyword(ctx context.Context, userID uuid.UUID, keyword string) (*models.UserLink, error) {
	query := `
		SELECT id, user_id, keyword, url, description, click_count, created_at, updated_at,
		       health_status, health_checked_at, health_error
		FROM user_links WHERE user_id = $1 AND keyword = $2
	`

	var link models.UserLink
	err := d.Pool.QueryRow(ctx, query, userID, keyword).Scan(
		&link.ID,
		&link.UserID,
		&link.Keyword,
		&link.URL,
		&link.Description,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
		&link.HealthStatus,
		&link.HealthCheckedAt,
		&link.HealthError,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	return &link, nil
}

// GetUserLinkByID retrieves a user's link override by ID, scoped to the user.
func (d *DB) GetUserLinkByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.UserLink, error) {
	query := `
		SELECT id, user_id, keyword, url, description, click_count, created_at, updated_at,
		       health_status, health_checked_at, health_error
		FROM user_links WHERE id = $1 AND user_id = $2
	`

	var link models.UserLink
	err := d.Pool.QueryRow(ctx, query, id, userID).Scan(
		&link.ID,
		&link.UserID,
		&link.Keyword,
		&link.URL,
		&link.Description,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
		&link.HealthStatus,
		&link.HealthCheckedAt,
		&link.HealthError,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	return &link, nil
}

// GetUserLinks retrieves all link overrides for a user.
func (d *DB) GetUserLinks(ctx context.Context, userID uuid.UUID) ([]models.UserLink, error) {
	query := `
		SELECT id, user_id, keyword, url, description, click_count, created_at, updated_at,
		       health_status, health_checked_at, health_error
		FROM user_links WHERE user_id = $1
		ORDER BY keyword ASC
	`

	rows, err := d.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.UserLink
	for rows.Next() {
		var link models.UserLink
		if err := rows.Scan(
			&link.ID,
			&link.UserID,
			&link.Keyword,
			&link.URL,
			&link.Description,
			&link.ClickCount,
			&link.CreatedAt,
			&link.UpdatedAt,
			&link.HealthStatus,
			&link.HealthCheckedAt,
			&link.HealthError,
		); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// UpdateUserLink updates a user's link override.
func (d *DB) UpdateUserLink(ctx context.Context, link *models.UserLink) error {
	query := `
		UPDATE user_links
		SET url = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	err := d.Pool.QueryRow(ctx, query,
		link.URL,
		link.Description,
		link.ID,
		link.UserID,
	).Scan(&link.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserLinkNotFound
	}
	return err
}

// DeleteUserLink deletes a user's link override.
func (d *DB) DeleteUserLink(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM user_links WHERE id = $1 AND user_id = $2`
	result, err := d.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserLinkNotFound
	}
	return nil
}

// IncrementUserLinkClickCount increments the click count for a user link.
func (d *DB) IncrementUserLinkClickCount(ctx context.Context, userID uuid.UUID, keyword string) error {
	query := `UPDATE user_links SET click_count = click_count + 1 WHERE user_id = $1 AND keyword = $2`
	_, err := d.Pool.Exec(ctx, query, userID, keyword)
	return err
}
