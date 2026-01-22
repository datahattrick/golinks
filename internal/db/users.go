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
		INSERT INTO users (sub, email, name, picture)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (sub) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			picture = EXCLUDED.picture,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return d.Pool.QueryRow(ctx, query,
		user.Sub,
		user.Email,
		user.Name,
		user.Picture,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetUserBySub retrieves a user by their OIDC subject identifier.
func (d *DB) GetUserBySub(ctx context.Context, sub string) (*models.User, error) {
	query := `
		SELECT id, sub, email, name, picture, created_at, updated_at
		FROM users WHERE sub = $1
	`

	var user models.User
	err := d.Pool.QueryRow(ctx, query, sub).Scan(
		&user.ID,
		&user.Sub,
		&user.Email,
		&user.Name,
		&user.Picture,
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
		SELECT id, sub, email, name, picture, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user models.User
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Sub,
		&user.Email,
		&user.Name,
		&user.Picture,
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
