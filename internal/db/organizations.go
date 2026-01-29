package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"golinks/internal/models"
)

var ErrOrgNotFound = errors.New("organization not found")

// CreateOrganization creates a new organization.
func (d *DB) CreateOrganization(ctx context.Context, org *models.Organization) error {
	query := `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`
	return d.Pool.QueryRow(ctx, query, org.Name, org.Slug).Scan(
		&org.ID, &org.CreatedAt, &org.UpdatedAt,
	)
}

// GetOrganizationByID retrieves an organization by ID.
func (d *DB) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE id = $1
	`

	var org models.Organization
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrgNotFound
	}
	if err != nil {
		return nil, err
	}

	return &org, nil
}

// GetOrganizationBySlug retrieves an organization by its slug.
func (d *DB) GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations WHERE slug = $1
	`

	var org models.Organization
	err := d.Pool.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrgNotFound
	}
	if err != nil {
		return nil, err
	}

	return &org, nil
}

// GetAllOrganizations retrieves all organizations.
func (d *DB) GetAllOrganizations(ctx context.Context) ([]models.Organization, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations ORDER BY name ASC
	`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}
