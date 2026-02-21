package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golinks/internal/models"
)


// CreateSharedLink inserts a share offer after checking anti-spam limits.
func (d *DB) CreateSharedLink(ctx context.Context, link *models.SharedLink) error {
	// Check sender outgoing limit
	var senderCount int
	if err := d.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM shared_links WHERE sender_id = $1`, link.SenderID,
	).Scan(&senderCount); err != nil {
		return err
	}
	if senderCount >= 5 {
		return ErrShareLimitReached
	}

	// Check recipient incoming limit
	var recipientCount int
	if err := d.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM shared_links WHERE recipient_id = $1`, link.RecipientID,
	).Scan(&recipientCount); err != nil {
		return err
	}
	if recipientCount >= 5 {
		return ErrRecipientLimitReached
	}

	query := `
		INSERT INTO shared_links (sender_id, recipient_id, keyword, url, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := d.Pool.QueryRow(ctx, query,
		link.SenderID,
		link.RecipientID,
		link.Keyword,
		link.URL,
		link.Description,
	).Scan(&link.ID, &link.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrDuplicateShare
			}
			if pgErr.Code == "23514" && pgErr.ConstraintName == "no_self_share" {
				return errors.New("you cannot share a link with yourself")
			}
		}
		return err
	}

	return nil
}

// GetIncomingShares returns pending shares for a recipient, with sender info.
func (d *DB) GetIncomingShares(ctx context.Context, recipientID uuid.UUID) ([]models.SharedLinkWithUser, error) {
	query := `
		SELECT sl.id, sl.sender_id, sl.recipient_id, sl.keyword, sl.url, sl.description, sl.created_at,
		       COALESCE(NULLIF(u.name, ''), NULLIF(u.username, ''), u.sub),
		       COALESCE(NULLIF(u.email, ''), u.sub)
		FROM shared_links sl
		JOIN users u ON u.id = sl.sender_id
		WHERE sl.recipient_id = $1
		ORDER BY sl.created_at DESC
	`

	rows, err := d.Pool.Query(ctx, query, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []models.SharedLinkWithUser
	for rows.Next() {
		var s models.SharedLinkWithUser
		if err := rows.Scan(
			&s.ID, &s.SenderID, &s.RecipientID, &s.Keyword, &s.URL, &s.Description, &s.CreatedAt,
			&s.UserName, &s.UserEmail,
		); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}

	return shares, rows.Err()
}

// GetOutgoingShares returns pending shares by a sender, with recipient info.
func (d *DB) GetOutgoingShares(ctx context.Context, senderID uuid.UUID) ([]models.SharedLinkWithUser, error) {
	query := `
		SELECT sl.id, sl.sender_id, sl.recipient_id, sl.keyword, sl.url, sl.description, sl.created_at,
		       COALESCE(NULLIF(u.name, ''), NULLIF(u.username, ''), u.sub),
		       COALESCE(NULLIF(u.email, ''), u.sub)
		FROM shared_links sl
		JOIN users u ON u.id = sl.recipient_id
		WHERE sl.sender_id = $1
		ORDER BY sl.created_at DESC
	`

	rows, err := d.Pool.Query(ctx, query, senderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []models.SharedLinkWithUser
	for rows.Next() {
		var s models.SharedLinkWithUser
		if err := rows.Scan(
			&s.ID, &s.SenderID, &s.RecipientID, &s.Keyword, &s.URL, &s.Description, &s.CreatedAt,
			&s.UserName, &s.UserEmail,
		); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}

	return shares, rows.Err()
}

// GetSharedLinkByID returns a single shared link by ID.
func (d *DB) GetSharedLinkByID(ctx context.Context, id uuid.UUID) (*models.SharedLink, error) {
	query := `
		SELECT id, sender_id, recipient_id, keyword, url, description, created_at
		FROM shared_links WHERE id = $1
	`

	var link models.SharedLink
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&link.ID, &link.SenderID, &link.RecipientID, &link.Keyword, &link.URL, &link.Description, &link.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSharedLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	return &link, nil
}

// DeleteSharedLink removes a shared link by ID.
func (d *DB) DeleteSharedLink(ctx context.Context, id uuid.UUID) error {
	result, err := d.Pool.Exec(ctx, `DELETE FROM shared_links WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrSharedLinkNotFound
	}
	return nil
}
