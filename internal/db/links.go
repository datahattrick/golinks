package db

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)

var (
	ErrDuplicateKeyword = errors.New("keyword already exists")
	ErrLinkNotFound     = errors.New("link not found")
)

// linkColumns is the standard column list for link queries.
const linkColumns = `id, keyword, url, description, scope, organization_id, status,
	created_by, submitted_by, reviewed_by, reviewed_at, click_count, created_at, updated_at,
	health_status, health_checked_at, health_error`

// scanLink scans a row into a Link struct.
func scanLink(row pgx.Row) (*models.Link, error) {
	var link models.Link
	err := row.Scan(
		&link.ID,
		&link.Keyword,
		&link.URL,
		&link.Description,
		&link.Scope,
		&link.OrganizationID,
		&link.Status,
		&link.CreatedBy,
		&link.SubmittedBy,
		&link.ReviewedBy,
		&link.ReviewedAt,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
		&link.HealthStatus,
		&link.HealthCheckedAt,
		&link.HealthError,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// scanLinks scans multiple rows into a slice of Links.
func scanLinks(rows pgx.Rows) ([]models.Link, error) {
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(
			&link.ID,
			&link.Keyword,
			&link.URL,
			&link.Description,
			&link.Scope,
			&link.OrganizationID,
			&link.Status,
			&link.CreatedBy,
			&link.SubmittedBy,
			&link.ReviewedBy,
			&link.ReviewedAt,
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

// CreateLink creates a new link (for moderators creating approved links directly).
func (d *DB) CreateLink(ctx context.Context, link *models.Link) error {
	query := `
		INSERT INTO links (keyword, url, description, scope, organization_id, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, click_count, created_at, updated_at
	`

	// Default status to approved for direct creation (by moderators)
	status := link.Status
	if status == "" {
		status = models.StatusApproved
	}

	err := d.Pool.QueryRow(ctx, query,
		link.Keyword,
		link.URL,
		link.Description,
		link.Scope,
		link.OrganizationID,
		status,
		link.CreatedBy,
	).Scan(&link.ID, &link.ClickCount, &link.CreatedAt, &link.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateKeyword
		}
		return err
	}

	link.Status = status
	return nil
}

// SubmitLinkForApproval creates a new link with pending status for moderator review.
func (d *DB) SubmitLinkForApproval(ctx context.Context, link *models.Link) error {
	query := `
		INSERT INTO links (keyword, url, description, scope, organization_id, status, submitted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, click_count, created_at, updated_at
	`

	err := d.Pool.QueryRow(ctx, query,
		link.Keyword,
		link.URL,
		link.Description,
		link.Scope,
		link.OrganizationID,
		models.StatusPending,
		link.SubmittedBy,
	).Scan(&link.ID, &link.ClickCount, &link.CreatedAt, &link.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateKeyword
		}
		return err
	}

	link.Status = models.StatusPending
	return nil
}

// ApproveLink approves a pending link.
func (d *DB) ApproveLink(ctx context.Context, linkID uuid.UUID, reviewerID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE links
		SET status = $1, reviewed_by = $2, reviewed_at = $3, created_by = submitted_by, updated_at = NOW()
		WHERE id = $4 AND status = $5
	`
	result, err := d.Pool.Exec(ctx, query,
		models.StatusApproved,
		reviewerID,
		now,
		linkID,
		models.StatusPending,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}

// RejectLink rejects a pending link.
func (d *DB) RejectLink(ctx context.Context, linkID uuid.UUID, reviewerID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE links
		SET status = $1, reviewed_by = $2, reviewed_at = $3, updated_at = NOW()
		WHERE id = $4 AND status = $5
	`
	result, err := d.Pool.Exec(ctx, query,
		models.StatusRejected,
		reviewerID,
		now,
		linkID,
		models.StatusPending,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}

// GetLinkByID retrieves a link by its ID.
func (d *DB) GetLinkByID(ctx context.Context, id uuid.UUID) (*models.Link, error) {
	query := `SELECT ` + linkColumns + ` FROM links WHERE id = $1`
	return scanLink(d.Pool.QueryRow(ctx, query, id))
}

// GetApprovedGlobalLinkByKeyword retrieves an approved global link by keyword.
func (d *DB) GetApprovedGlobalLinkByKeyword(ctx context.Context, keyword string) (*models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE keyword = $1 AND scope = $2 AND status = $3
	`
	return scanLink(d.Pool.QueryRow(ctx, query, keyword, models.ScopeGlobal, models.StatusApproved))
}

// GetApprovedOrgLinkByKeyword retrieves an approved org link by keyword and org ID.
func (d *DB) GetApprovedOrgLinkByKeyword(ctx context.Context, keyword string, orgID uuid.UUID) (*models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE keyword = $1 AND scope = $2 AND organization_id = $3 AND status = $4
	`
	return scanLink(d.Pool.QueryRow(ctx, query, keyword, models.ScopeOrg, orgID, models.StatusApproved))
}

// GetLinkByKeyword retrieves any approved link by keyword (for backwards compatibility).
func (d *DB) GetLinkByKeyword(ctx context.Context, keyword string) (*models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE keyword = $1 AND status = $2
		ORDER BY CASE scope WHEN 'global' THEN 1 ELSE 2 END
		LIMIT 1
	`
	return scanLink(d.Pool.QueryRow(ctx, query, keyword, models.StatusApproved))
}

// IncrementClickCount increments the click count for a link.
func (d *DB) IncrementClickCount(ctx context.Context, linkID uuid.UUID) error {
	query := `UPDATE links SET click_count = click_count + 1 WHERE id = $1`
	_, err := d.Pool.Exec(ctx, query, linkID)
	return err
}

// GetPendingGlobalLinks retrieves all pending global links for moderation.
func (d *DB) GetPendingGlobalLinks(ctx context.Context) ([]models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE scope = $1 AND status = $2
		ORDER BY created_at ASC
	`
	rows, err := d.Pool.Query(ctx, query, models.ScopeGlobal, models.StatusPending)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// GetPendingOrgLinks retrieves all pending org links for a specific organization.
func (d *DB) GetPendingOrgLinks(ctx context.Context, orgID uuid.UUID) ([]models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE scope = $1 AND organization_id = $2 AND status = $3
		ORDER BY created_at ASC
	`
	rows, err := d.Pool.Query(ctx, query, models.ScopeOrg, orgID, models.StatusPending)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// GetApprovedGlobalLinks retrieves all approved global links.
func (d *DB) GetApprovedGlobalLinks(ctx context.Context) ([]models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE scope = $1 AND status = $2
		ORDER BY click_count DESC, keyword ASC
	`
	rows, err := d.Pool.Query(ctx, query, models.ScopeGlobal, models.StatusApproved)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// GetApprovedOrgLinks retrieves all approved links for an organization.
func (d *DB) GetApprovedOrgLinks(ctx context.Context, orgID uuid.UUID) ([]models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE scope = $1 AND organization_id = $2 AND status = $3
		ORDER BY click_count DESC, keyword ASC
	`
	rows, err := d.Pool.Query(ctx, query, models.ScopeOrg, orgID, models.StatusApproved)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// SearchApprovedLinks searches for approved links by keyword, URL, or description.
// If orgID is provided, includes org-scoped links for that organization.
func (d *DB) SearchApprovedLinks(ctx context.Context, queryStr string, orgID *uuid.UUID, limit int) ([]models.Link, error) {
	var sql string
	var args []any

	if strings.TrimSpace(queryStr) == "" {
		if orgID != nil {
			sql = `
				SELECT ` + linkColumns + `
				FROM links
				WHERE status = $1 AND (scope = $2 OR (scope = $3 AND organization_id = $4))
				ORDER BY click_count DESC, keyword ASC
				LIMIT $5
			`
			args = []any{models.StatusApproved, models.ScopeGlobal, models.ScopeOrg, *orgID, limit}
		} else {
			sql = `
				SELECT ` + linkColumns + `
				FROM links
				WHERE status = $1 AND scope = $2
				ORDER BY click_count DESC, keyword ASC
				LIMIT $3
			`
			args = []any{models.StatusApproved, models.ScopeGlobal, limit}
		}
	} else {
		pattern := "%" + queryStr + "%"
		if orgID != nil {
			sql = `
				SELECT ` + linkColumns + `
				FROM links
				WHERE status = $1
					AND (scope = $2 OR (scope = $3 AND organization_id = $4))
					AND (keyword ILIKE $5 OR url ILIKE $5 OR description ILIKE $5)
				ORDER BY click_count DESC, keyword ASC
				LIMIT $6
			`
			args = []any{models.StatusApproved, models.ScopeGlobal, models.ScopeOrg, *orgID, pattern, limit}
		} else {
			sql = `
				SELECT ` + linkColumns + `
				FROM links
				WHERE status = $1 AND scope = $2
					AND (keyword ILIKE $3 OR url ILIKE $3 OR description ILIKE $3)
				ORDER BY click_count DESC, keyword ASC
				LIMIT $4
			`
			args = []any{models.StatusApproved, models.ScopeGlobal, pattern, limit}
		}
	}

	rows, err := d.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// SearchLinks is kept for backwards compatibility - searches approved global links.
func (d *DB) SearchLinks(ctx context.Context, query string, limit int) ([]models.Link, error) {
	return d.SearchApprovedLinks(ctx, query, nil, limit)
}

// GetLinksByUser retrieves all links created/submitted by a specific user.
func (d *DB) GetLinksByUser(ctx context.Context, userID uuid.UUID) ([]models.Link, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE created_by = $1 OR submitted_by = $1
		ORDER BY created_at DESC
	`

	rows, err := d.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// DeleteLink deletes a link by ID. For moderators, no ownership check is done.
func (d *DB) DeleteLink(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM links WHERE id = $1`
	result, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}

// DeleteLinkByUser deletes a link by ID, but only if submitted by the specified user and still pending.
func (d *DB) DeleteLinkByUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM links WHERE id = $1 AND submitted_by = $2 AND status = $3`
	result, err := d.Pool.Exec(ctx, query, id, userID, models.StatusPending)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}

// UpdateLink updates a link's URL and description.
func (d *DB) UpdateLink(ctx context.Context, link *models.Link) error {
	query := `
		UPDATE links
		SET url = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`
	err := d.Pool.QueryRow(ctx, query, link.URL, link.Description, link.ID).Scan(&link.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLinkNotFound
	}
	return err
}

// UpdateLinkAndResetHealth updates a link's URL and description and resets health status.
func (d *DB) UpdateLinkAndResetHealth(ctx context.Context, link *models.Link) error {
	query := `
		UPDATE links
		SET url = $1, description = $2, health_status = $3, health_checked_at = NULL, health_error = NULL, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`
	err := d.Pool.QueryRow(ctx, query, link.URL, link.Description, models.HealthUnknown, link.ID).Scan(&link.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLinkNotFound
	}
	link.HealthStatus = models.HealthUnknown
	link.HealthCheckedAt = nil
	link.HealthError = nil
	return err
}

// UpdateLinkHealthStatus updates the health status for a link.
func (d *DB) UpdateLinkHealthStatus(ctx context.Context, linkID uuid.UUID, status string, errorMsg *string) error {
	query := `
		UPDATE links
		SET health_status = $1, health_checked_at = NOW(), health_error = $2
		WHERE id = $3
	`
	result, err := d.Pool.Exec(ctx, query, status, errorMsg, linkID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}

// GetLinksForManagement retrieves links for the management page based on user role and filter.
func (d *DB) GetLinksForManagement(ctx context.Context, user *models.User, healthFilter string, limit int) ([]models.Link, error) {
	var sql string
	var args []any

	// Build the base query based on user role
	if user.IsGlobalMod() {
		// Global mods and admins can see all approved links
		sql = `
			SELECT ` + linkColumns + `
			FROM links
			WHERE status = $1
		`
		args = []any{models.StatusApproved}
	} else if user.IsOrgMod() && user.OrganizationID != nil {
		// Org mods can only see their org's links
		sql = `
			SELECT ` + linkColumns + `
			FROM links
			WHERE status = $1 AND scope = $2 AND organization_id = $3
		`
		args = []any{models.StatusApproved, models.ScopeOrg, *user.OrganizationID}
	} else {
		// Regular users shouldn't reach here, but return empty
		return []models.Link{}, nil
	}

	// Apply health filter
	if healthFilter != "" && healthFilter != "all" {
		sql += ` AND health_status = $` + strconv.Itoa(len(args)+1)
		args = append(args, healthFilter)
	}

	// Order and limit
	sql += ` ORDER BY keyword ASC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := d.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// GetLinksNeedingHealthCheck retrieves links that need a health check.
func (d *DB) GetLinksNeedingHealthCheck(ctx context.Context, maxAge time.Duration, limit int) ([]models.Link, error) {
	cutoff := time.Now().Add(-maxAge)
	query := `
		SELECT ` + linkColumns + `
		FROM links
		WHERE status = $1 AND (health_checked_at IS NULL OR health_checked_at < $2)
		ORDER BY health_checked_at NULLS FIRST
		LIMIT $3
	`

	rows, err := d.Pool.Query(ctx, query, models.StatusApproved, cutoff, limit)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}
