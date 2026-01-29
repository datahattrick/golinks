package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)

// Group link related errors.
var (
	ErrGroupLinkNotFound      = errors.New("group link not found")
	ErrGroupLinkDuplicateKeyword = errors.New("group link keyword already exists")
)

// CreateGroupLink creates a new group link.
func (d *DB) CreateGroupLink(ctx context.Context, link *models.GroupLink) error {
	query := `
		INSERT INTO group_links (group_id, keyword, url, description, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, click_count, health_status, created_at, updated_at
	`
	err := d.Pool.QueryRow(ctx, query,
		link.GroupID, link.Keyword, link.URL, link.Description, link.Status, link.CreatedBy,
	).Scan(&link.ID, &link.ClickCount, &link.HealthStatus, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrGroupLinkDuplicateKeyword
		}
		return fmt.Errorf("failed to create group link: %w", err)
	}
	return nil
}

// GetGroupLinkByID retrieves a group link by ID.
func (d *DB) GetGroupLinkByID(ctx context.Context, id uuid.UUID) (*models.GroupLink, error) {
	query := `
		SELECT id, group_id, keyword, url, description, status, click_count,
			created_by, submitted_by, reviewed_by, reviewed_at,
			health_status, health_checked_at, health_error, created_at, updated_at
		FROM group_links
		WHERE id = $1
	`
	link := &models.GroupLink{}
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&link.ID, &link.GroupID, &link.Keyword, &link.URL, &link.Description,
		&link.Status, &link.ClickCount, &link.CreatedBy, &link.SubmittedBy,
		&link.ReviewedBy, &link.ReviewedAt, &link.HealthStatus,
		&link.HealthCheckedAt, &link.HealthError, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupLinkNotFound
		}
		return nil, fmt.Errorf("failed to get group link: %w", err)
	}
	return link, nil
}

// GetGroupLinkByKeyword retrieves a group link by group ID and keyword.
func (d *DB) GetGroupLinkByKeyword(ctx context.Context, groupID uuid.UUID, keyword string) (*models.GroupLink, error) {
	query := `
		SELECT id, group_id, keyword, url, description, status, click_count,
			created_by, submitted_by, reviewed_by, reviewed_at,
			health_status, health_checked_at, health_error, created_at, updated_at
		FROM group_links
		WHERE group_id = $1 AND keyword = $2
	`
	link := &models.GroupLink{}
	err := d.Pool.QueryRow(ctx, query, groupID, keyword).Scan(
		&link.ID, &link.GroupID, &link.Keyword, &link.URL, &link.Description,
		&link.Status, &link.ClickCount, &link.CreatedBy, &link.SubmittedBy,
		&link.ReviewedBy, &link.ReviewedAt, &link.HealthStatus,
		&link.HealthCheckedAt, &link.HealthError, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupLinkNotFound
		}
		return nil, fmt.Errorf("failed to get group link: %w", err)
	}
	return link, nil
}

// UpdateGroupLink updates a group link.
func (d *DB) UpdateGroupLink(ctx context.Context, link *models.GroupLink) error {
	query := `
		UPDATE group_links
		SET url = $2, description = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err := d.Pool.QueryRow(ctx, query, link.ID, link.URL, link.Description).Scan(&link.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupLinkNotFound
		}
		return fmt.Errorf("failed to update group link: %w", err)
	}
	return nil
}

// DeleteGroupLink deletes a group link.
func (d *DB) DeleteGroupLink(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM group_links WHERE id = $1`
	result, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group link: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupLinkNotFound
	}
	return nil
}

// ListGroupLinks lists all links for a group.
func (d *DB) ListGroupLinks(ctx context.Context, groupID uuid.UUID, statusFilter string) ([]models.GroupLink, error) {
	var query string
	var args []any

	if statusFilter != "" {
		query = `
			SELECT id, group_id, keyword, url, description, status, click_count,
				created_by, submitted_by, reviewed_by, reviewed_at,
				health_status, health_checked_at, health_error, created_at, updated_at
			FROM group_links
			WHERE group_id = $1 AND status = $2
			ORDER BY keyword ASC
		`
		args = []any{groupID, statusFilter}
	} else {
		query = `
			SELECT id, group_id, keyword, url, description, status, click_count,
				created_by, submitted_by, reviewed_by, reviewed_at,
				health_status, health_checked_at, health_error, created_at, updated_at
			FROM group_links
			WHERE group_id = $1
			ORDER BY keyword ASC
		`
		args = []any{groupID}
	}

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list group links: %w", err)
	}
	defer rows.Close()

	var links []models.GroupLink
	for rows.Next() {
		var link models.GroupLink
		if err := rows.Scan(
			&link.ID, &link.GroupID, &link.Keyword, &link.URL, &link.Description,
			&link.Status, &link.ClickCount, &link.CreatedBy, &link.SubmittedBy,
			&link.ReviewedBy, &link.ReviewedAt, &link.HealthStatus,
			&link.HealthCheckedAt, &link.HealthError, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group link: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
}

// IncrementGroupLinkClickCount increments the click count for a group link.
func (d *DB) IncrementGroupLinkClickCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE group_links SET click_count = click_count + 1 WHERE id = $1`
	_, err := d.Pool.Exec(ctx, query, id)
	return err
}

// ResolveKeywordForUser resolves a keyword to a URL using the tier-based hierarchy.
// Priority: personal (tier 100) > group links (tier 1-99, ordered by tier desc, primary first) > global (tier 0)
// Returns the resolved link with source information, or ErrLinkNotFound if not found.
func (d *DB) ResolveKeywordForUser(ctx context.Context, userID *uuid.UUID, keyword string) (*models.ResolvedLink, error) {
	query := `
		WITH user_groups AS (
			SELECT g.id, g.tier, ugm.is_primary
			FROM user_group_memberships ugm
			JOIN groups g ON g.id = ugm.group_id
			WHERE ugm.user_id = $1
		),
		personal_link AS (
			SELECT id, url, 100 as tier, true as is_primary, 'personal'::text as source
			FROM user_links
			WHERE user_id = $1 AND keyword = $2
		),
		group_links_match AS (
			SELECT gl.id, gl.url, g.tier, ug.is_primary, 'group'::text as source
			FROM group_links gl
			JOIN groups g ON g.id = gl.group_id
			JOIN user_groups ug ON ug.id = g.id
			WHERE gl.keyword = $2 AND gl.status = 'approved'
		),
		global_link AS (
			SELECT id, url, 0 as tier, true as is_primary, 'global'::text as source
			FROM links
			WHERE keyword = $2 AND scope = 'global' AND status = 'approved'
		)
		SELECT id, url, tier, is_primary, source FROM (
			SELECT * FROM personal_link
			UNION ALL SELECT * FROM group_links_match
			UNION ALL SELECT * FROM global_link
		) combined
		ORDER BY tier DESC, is_primary DESC
		LIMIT 1
	`

	resolved := &models.ResolvedLink{}

	// If no user, only check global links
	if userID == nil {
		globalQuery := `
			SELECT id, url, 0 as tier, true as is_primary, 'global'::text as source
			FROM links
			WHERE keyword = $1 AND scope = 'global' AND status = 'approved'
			LIMIT 1
		`
		err := d.Pool.QueryRow(ctx, globalQuery, keyword).Scan(
			&resolved.ID, &resolved.URL, &resolved.Tier, &resolved.IsPrimary, &resolved.Source,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrLinkNotFound
			}
			return nil, fmt.Errorf("failed to resolve keyword: %w", err)
		}
		return resolved, nil
	}

	err := d.Pool.QueryRow(ctx, query, userID, keyword).Scan(
		&resolved.ID, &resolved.URL, &resolved.Tier, &resolved.IsPrimary, &resolved.Source,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("failed to resolve keyword: %w", err)
	}
	return resolved, nil
}

// IncrementResolvedLinkClickCount increments the click count for a resolved link based on source.
func (d *DB) IncrementResolvedLinkClickCount(ctx context.Context, resolved *models.ResolvedLink, userID *uuid.UUID) error {
	switch resolved.Source {
	case "personal":
		if userID != nil {
			// For user links, we need the keyword - but we have the ID
			// Let's update by ID
			query := `UPDATE user_links SET click_count = click_count + 1 WHERE id = $1`
			_, err := d.Pool.Exec(ctx, query, resolved.ID)
			return err
		}
		return nil
	case "group":
		return d.IncrementGroupLinkClickCount(ctx, resolved.ID)
	case "global":
		return d.IncrementClickCount(ctx, resolved.ID)
	default:
		return nil
	}
}

// ApproveGroupLink approves a pending group link.
func (d *DB) ApproveGroupLink(ctx context.Context, id, reviewerID uuid.UUID) error {
	query := `
		UPDATE group_links
		SET status = 'approved', reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`
	result, err := d.Pool.Exec(ctx, query, id, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to approve group link: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupLinkNotFound
	}
	return nil
}

// RejectGroupLink rejects a pending group link.
func (d *DB) RejectGroupLink(ctx context.Context, id, reviewerID uuid.UUID) error {
	query := `
		UPDATE group_links
		SET status = 'rejected', reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`
	result, err := d.Pool.Exec(ctx, query, id, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to reject group link: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupLinkNotFound
	}
	return nil
}

// SubmitGroupLinkForApproval creates a group link in pending status.
func (d *DB) SubmitGroupLinkForApproval(ctx context.Context, link *models.GroupLink) error {
	link.Status = "pending"
	query := `
		INSERT INTO group_links (group_id, keyword, url, description, status, submitted_by)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, click_count, health_status, created_at, updated_at
	`
	err := d.Pool.QueryRow(ctx, query,
		link.GroupID, link.Keyword, link.URL, link.Description, link.SubmittedBy,
	).Scan(&link.ID, &link.ClickCount, &link.HealthStatus, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrGroupLinkDuplicateKeyword
		}
		return fmt.Errorf("failed to submit group link for approval: %w", err)
	}
	return nil
}
