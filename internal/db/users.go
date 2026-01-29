package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"golinks/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

// UpsertUser creates or updates a user based on their OIDC subject.
func (d *DB) UpsertUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (sub, username, email, name, picture, role, organization_id)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, 'user'), $7)
		ON CONFLICT (sub) DO UPDATE SET
			username = COALESCE(EXCLUDED.username, users.username),
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			picture = EXCLUDED.picture,
			updated_at = NOW()
		RETURNING id, role, organization_id, created_at, updated_at
	`

	return d.Pool.QueryRow(ctx, query,
		user.Sub,
		nullIfEmpty(user.Username),
		user.Email,
		user.Name,
		user.Picture,
		nullIfEmpty(user.Role),
		user.OrganizationID,
	).Scan(&user.ID, &user.Role, &user.OrganizationID, &user.CreatedAt, &user.UpdatedAt)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetUserBySub retrieves a user by their OIDC subject identifier.
func (d *DB) GetUserBySub(ctx context.Context, sub string) (*models.User, error) {
	query := `
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, created_at, updated_at
		FROM users WHERE sub = $1
	`

	var user models.User
	err := d.Pool.QueryRow(ctx, query, sub).Scan(
		&user.ID,
		&user.Sub,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.Role,
		&user.OrganizationID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by their PKI username.
func (d *DB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, created_at, updated_at
		FROM users WHERE username = $1
	`

	var user models.User
	err := d.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Sub,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.Role,
		&user.OrganizationID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by their UUID.
func (d *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user models.User
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Sub,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.Role,
		&user.OrganizationID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUserRole updates a user's role (admin only).
func (d *DB) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	query := `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`
	_, err := d.Pool.Exec(ctx, query, role, userID)
	return err
}

// UpdateUserOrganization updates a user's organization membership.
func (d *DB) UpdateUserOrganization(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID) error {
	query := `UPDATE users SET organization_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := d.Pool.Exec(ctx, query, orgID, userID)
	return err
}
