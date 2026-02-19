package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)

var (
	ErrEditRequestNotFound = errors.New("edit request not found")
	ErrPendingRequestLimit = errors.New("you have reached the maximum number of pending requests (5)")
	ErrDuplicateEditRequest = errors.New("you already have a pending edit request for this link")
)

// CreateEditRequest inserts a new edit request after checking limits.
func (d *DB) CreateEditRequest(ctx context.Context, req *models.LinkEditRequest) error {
	// Check pending request limit
	count, err := d.CountPendingRequestsByUser(ctx, req.UserID)
	if err != nil {
		return err
	}
	if count >= 5 {
		return ErrPendingRequestLimit
	}

	query := `
		INSERT INTO link_edit_requests (link_id, user_id, url, description, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at
	`
	err = d.Pool.QueryRow(ctx, query,
		req.LinkID,
		req.UserID,
		req.URL,
		req.Description,
		req.Reason,
	).Scan(&req.ID, &req.Status, &req.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEditRequest
		}
		return err
	}
	return nil
}

// GetEditRequestByID retrieves an edit request with link keyword and author info.
func (d *DB) GetEditRequestByID(ctx context.Context, id uuid.UUID) (*models.LinkEditRequest, error) {
	query := `
		SELECT r.id, r.link_id, r.user_id, r.url, r.description, r.reason, r.status,
			r.reviewed_by, r.reviewed_at, r.created_at,
			l.keyword, COALESCE(u.name, ''), COALESCE(u.email, '')
		FROM link_edit_requests r
		JOIN links l ON l.id = r.link_id
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1
	`
	var req models.LinkEditRequest
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.LinkID, &req.UserID, &req.URL, &req.Description, &req.Reason, &req.Status,
		&req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt,
		&req.Keyword, &req.AuthorName, &req.AuthorEmail,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEditRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetPendingEditRequests returns pending edit requests scoped by user role.
func (d *DB) GetPendingEditRequests(ctx context.Context, user *models.User) ([]models.LinkEditRequest, error) {
	var sql string
	var args []any

	if user.IsGlobalMod() {
		sql = `
			SELECT r.id, r.link_id, r.user_id, r.url, r.description, r.reason, r.status,
				r.reviewed_by, r.reviewed_at, r.created_at,
				l.keyword, COALESCE(u.name, ''), COALESCE(u.email, '')
			FROM link_edit_requests r
			JOIN links l ON l.id = r.link_id
			JOIN users u ON u.id = r.user_id
			WHERE r.status = $1
			ORDER BY r.created_at ASC
		`
		args = []any{models.StatusPending}
	} else if user.IsOrgMod() && user.OrganizationID != nil {
		sql = `
			SELECT r.id, r.link_id, r.user_id, r.url, r.description, r.reason, r.status,
				r.reviewed_by, r.reviewed_at, r.created_at,
				l.keyword, COALESCE(u.name, ''), COALESCE(u.email, '')
			FROM link_edit_requests r
			JOIN links l ON l.id = r.link_id
			JOIN users u ON u.id = r.user_id
			WHERE r.status = $1 AND l.scope = $2 AND l.organization_id = $3
			ORDER BY r.created_at ASC
		`
		args = []any{models.StatusPending, models.ScopeOrg, *user.OrganizationID}
	} else {
		return []models.LinkEditRequest{}, nil
	}

	rows, err := d.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.LinkEditRequest
	for rows.Next() {
		var req models.LinkEditRequest
		if err := rows.Scan(
			&req.ID, &req.LinkID, &req.UserID, &req.URL, &req.Description, &req.Reason, &req.Status,
			&req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt,
			&req.Keyword, &req.AuthorName, &req.AuthorEmail,
		); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// ApproveEditRequest approves an edit request and applies changes to the link.
func (d *DB) ApproveEditRequest(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get the edit request
	var req models.LinkEditRequest
	err = tx.QueryRow(ctx, `
		SELECT id, link_id, url, description FROM link_edit_requests
		WHERE id = $1 AND status = $2
	`, id, models.StatusPending).Scan(&req.ID, &req.LinkID, &req.URL, &req.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEditRequestNotFound
	}
	if err != nil {
		return err
	}

	// Apply changes to the link and reset health
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE links
		SET url = $1, description = $2, health_status = $3, health_checked_at = NULL, health_error = NULL, updated_at = NOW()
		WHERE id = $4
	`, req.URL, req.Description, models.HealthUnknown, req.LinkID)
	if err != nil {
		return err
	}

	// Mark edit request as approved
	_, err = tx.Exec(ctx, `
		UPDATE link_edit_requests
		SET status = $1, reviewed_by = $2, reviewed_at = $3
		WHERE id = $4
	`, models.StatusApproved, reviewerID, now, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RejectEditRequest rejects an edit request.
func (d *DB) RejectEditRequest(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error {
	now := time.Now()
	result, err := d.Pool.Exec(ctx, `
		UPDATE link_edit_requests
		SET status = $1, reviewed_by = $2, reviewed_at = $3
		WHERE id = $4 AND status = $5
	`, models.StatusRejected, reviewerID, now, id, models.StatusPending)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEditRequestNotFound
	}
	return nil
}

// GetLinkIDsWithPendingEdits returns a set of link IDs that have at least one pending edit request.
func (d *DB) GetLinkIDsWithPendingEdits(ctx context.Context, linkIDs []uuid.UUID) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(linkIDs) == 0 {
		return result, nil
	}

	rows, err := d.Pool.Query(ctx, `
		SELECT DISTINCT link_id FROM link_edit_requests
		WHERE status = $1 AND link_id = ANY($2)
	`, models.StatusPending, linkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id.String()] = true
	}
	return result, rows.Err()
}

// GetPendingEditRequestForLink checks if a link has a pending edit request from a user.
func (d *DB) GetPendingEditRequestForLink(ctx context.Context, linkID uuid.UUID, userID uuid.UUID) (*models.LinkEditRequest, error) {
	var req models.LinkEditRequest
	err := d.Pool.QueryRow(ctx, `
		SELECT id, link_id, user_id, url, description, reason, status, reviewed_by, reviewed_at, created_at
		FROM link_edit_requests
		WHERE link_id = $1 AND user_id = $2 AND status = $3
	`, linkID, userID, models.StatusPending).Scan(
		&req.ID, &req.LinkID, &req.UserID, &req.URL, &req.Description, &req.Reason, &req.Status,
		&req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEditRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}
