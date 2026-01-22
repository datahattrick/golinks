package db

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)

var (
	ErrDuplicateKeyword = errors.New("keyword already exists")
	ErrLinkNotFound     = errors.New("link not found")
)

// CreateLink creates a new link. Returns ErrDuplicateKeyword if keyword exists.
func (d *DB) CreateLink(ctx context.Context, link *models.Link) error {
	query := `
		INSERT INTO links (keyword, url, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, click_count, created_at, updated_at
	`

	err := d.Pool.QueryRow(ctx, query,
		link.Keyword,
		link.URL,
		link.Description,
		link.CreatedBy,
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

// GetLinkByKeyword retrieves a link by its keyword.
func (d *DB) GetLinkByKeyword(ctx context.Context, keyword string) (*models.Link, error) {
	query := `
		SELECT id, keyword, url, description, created_by, click_count, created_at, updated_at
		FROM links WHERE keyword = $1
	`

	var link models.Link
	err := d.Pool.QueryRow(ctx, query, keyword).Scan(
		&link.ID,
		&link.Keyword,
		&link.URL,
		&link.Description,
		&link.CreatedBy,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	return &link, nil
}

// IncrementClickCount increments the click count for a link.
func (d *DB) IncrementClickCount(ctx context.Context, keyword string) error {
	query := `UPDATE links SET click_count = click_count + 1 WHERE keyword = $1`
	_, err := d.Pool.Exec(ctx, query, keyword)
	return err
}

// SearchLinks searches for links by keyword, URL, or description using trigram matching.
func (d *DB) SearchLinks(ctx context.Context, query string, limit int) ([]models.Link, error) {
	var sql string
	var args []any

	if strings.TrimSpace(query) == "" {
		sql = `
			SELECT id, keyword, url, description, created_by, click_count, created_at, updated_at
			FROM links
			ORDER BY click_count DESC, keyword ASC
			LIMIT $1
		`
		args = []any{limit}
	} else {
		sql = `
			SELECT id, keyword, url, description, created_by, click_count, created_at, updated_at
			FROM links
			WHERE keyword ILIKE $1 OR url ILIKE $1 OR description ILIKE $1
			ORDER BY click_count DESC, keyword ASC
			LIMIT $2
		`
		args = []any{"%" + query + "%", limit}
	}

	rows, err := d.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(
			&link.ID,
			&link.Keyword,
			&link.URL,
			&link.Description,
			&link.CreatedBy,
			&link.ClickCount,
			&link.CreatedAt,
			&link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// GetLinksByUser retrieves all links created by a specific user.
func (d *DB) GetLinksByUser(ctx context.Context, userID uuid.UUID) ([]models.Link, error) {
	query := `
		SELECT id, keyword, url, description, created_by, click_count, created_at, updated_at
		FROM links WHERE created_by = $1
		ORDER BY created_at DESC
	`

	rows, err := d.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(
			&link.ID,
			&link.Keyword,
			&link.URL,
			&link.Description,
			&link.CreatedBy,
			&link.ClickCount,
			&link.CreatedAt,
			&link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// DeleteLink deletes a link by ID, but only if it belongs to the specified user.
func (d *DB) DeleteLink(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM links WHERE id = $1 AND created_by = $2`
	result, err := d.Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}
