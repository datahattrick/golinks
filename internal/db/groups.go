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

// Group-related errors.
var (
	ErrGroupNotFound          = errors.New("group not found")
	ErrGroupSlugExists        = errors.New("group slug already exists")
	ErrMembershipNotFound     = errors.New("membership not found")
	ErrMembershipAlreadyExists = errors.New("membership already exists")
)

// CreateGroup creates a new group.
func (d *DB) CreateGroup(ctx context.Context, group *models.Group) error {
	query := `
		INSERT INTO groups (name, slug, tier, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := d.Pool.QueryRow(ctx, query, group.Name, group.Slug, group.Tier, group.ParentID).
		Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrGroupSlugExists
		}
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// GetGroupByID retrieves a group by ID.
func (d *DB) GetGroupByID(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	query := `
		SELECT id, name, slug, tier, parent_id, created_at, updated_at
		FROM groups
		WHERE id = $1
	`
	group := &models.Group{}
	err := d.Pool.QueryRow(ctx, query, id).Scan(
		&group.ID, &group.Name, &group.Slug, &group.Tier,
		&group.ParentID, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return group, nil
}

// GetGroupBySlug retrieves a group by slug.
func (d *DB) GetGroupBySlug(ctx context.Context, slug string) (*models.Group, error) {
	query := `
		SELECT id, name, slug, tier, parent_id, created_at, updated_at
		FROM groups
		WHERE slug = $1
	`
	group := &models.Group{}
	err := d.Pool.QueryRow(ctx, query, slug).Scan(
		&group.ID, &group.Name, &group.Slug, &group.Tier,
		&group.ParentID, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return group, nil
}

// UpdateGroup updates a group's details.
func (d *DB) UpdateGroup(ctx context.Context, group *models.Group) error {
	query := `
		UPDATE groups
		SET name = $2, slug = $3, tier = $4, parent_id = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err := d.Pool.QueryRow(ctx, query, group.ID, group.Name, group.Slug, group.Tier, group.ParentID).
		Scan(&group.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrGroupSlugExists
		}
		return fmt.Errorf("failed to update group: %w", err)
	}
	return nil
}

// DeleteGroup deletes a group by ID.
func (d *DB) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM groups WHERE id = $1`
	result, err := d.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// ListGroups lists all groups, optionally filtered by parent.
func (d *DB) ListGroups(ctx context.Context, parentID *uuid.UUID) ([]models.Group, error) {
	var query string
	var args []interface{}

	if parentID != nil {
		query = `
			SELECT id, name, slug, tier, parent_id, created_at, updated_at
			FROM groups
			WHERE parent_id = $1
			ORDER BY tier DESC, name ASC
		`
		args = []interface{}{*parentID}
	} else {
		query = `
			SELECT id, name, slug, tier, parent_id, created_at, updated_at
			FROM groups
			ORDER BY tier DESC, name ASC
		`
	}

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.Tier, &g.ParentID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// AddUserToGroup adds a user to a group.
func (d *DB) AddUserToGroup(ctx context.Context, membership *models.UserGroupMembership) error {
	query := `
		INSERT INTO user_group_memberships (user_id, group_id, is_primary, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := d.Pool.QueryRow(ctx, query, membership.UserID, membership.GroupID, membership.IsPrimary, membership.Role).
		Scan(&membership.ID, &membership.CreatedAt, &membership.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrMembershipAlreadyExists
		}
		return fmt.Errorf("failed to add user to group: %w", err)
	}

	// If this is marked as primary, unset other primary memberships
	if membership.IsPrimary {
		_, err = d.Pool.Exec(ctx, `
			UPDATE user_group_memberships
			SET is_primary = false, updated_at = NOW()
			WHERE user_id = $1 AND group_id != $2 AND is_primary = true
		`, membership.UserID, membership.GroupID)
		if err != nil {
			return fmt.Errorf("failed to update primary membership: %w", err)
		}
	}
	return nil
}

// RemoveUserFromGroup removes a user from a group.
func (d *DB) RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	query := `DELETE FROM user_group_memberships WHERE user_id = $1 AND group_id = $2`
	result, err := d.Pool.Exec(ctx, query, userID, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

// GetUserMemberships retrieves all group memberships for a user, with group details.
func (d *DB) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]models.UserGroupMembership, error) {
	query := `
		SELECT
			ugm.id, ugm.user_id, ugm.group_id, ugm.is_primary, ugm.role,
			ugm.created_at, ugm.updated_at,
			g.id, g.name, g.slug, g.tier, g.parent_id, g.created_at, g.updated_at
		FROM user_group_memberships ugm
		JOIN groups g ON g.id = ugm.group_id
		WHERE ugm.user_id = $1
		ORDER BY g.tier DESC, ugm.is_primary DESC, g.name ASC
	`
	rows, err := d.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user memberships: %w", err)
	}
	defer rows.Close()

	var memberships []models.UserGroupMembership
	for rows.Next() {
		var m models.UserGroupMembership
		var g models.Group
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.GroupID, &m.IsPrimary, &m.Role,
			&m.CreatedAt, &m.UpdatedAt,
			&g.ID, &g.Name, &g.Slug, &g.Tier, &g.ParentID, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan membership: %w", err)
		}
		m.Group = &g
		memberships = append(memberships, m)
	}
	return memberships, nil
}

// GetGroupMembers retrieves all members of a group.
func (d *DB) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]models.UserGroupMembership, error) {
	query := `
		SELECT id, user_id, group_id, is_primary, role, created_at, updated_at
		FROM user_group_memberships
		WHERE group_id = $1
		ORDER BY role DESC, created_at ASC
	`
	rows, err := d.Pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []models.UserGroupMembership
	for rows.Next() {
		var m models.UserGroupMembership
		if err := rows.Scan(&m.ID, &m.UserID, &m.GroupID, &m.IsPrimary, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan membership: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

// SetPrimaryGroup sets a group as the user's primary group.
func (d *DB) SetPrimaryGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	// Start a transaction
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify membership exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_group_memberships WHERE user_id = $1 AND group_id = $2)
	`, userID, groupID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !exists {
		return ErrMembershipNotFound
	}

	// Unset all primary flags for this user
	_, err = tx.Exec(ctx, `
		UPDATE user_group_memberships
		SET is_primary = false, updated_at = NOW()
		WHERE user_id = $1 AND is_primary = true
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to unset primary: %w", err)
	}

	// Set the new primary
	_, err = tx.Exec(ctx, `
		UPDATE user_group_memberships
		SET is_primary = true, updated_at = NOW()
		WHERE user_id = $1 AND group_id = $2
	`, userID, groupID)
	if err != nil {
		return fmt.Errorf("failed to set primary: %w", err)
	}

	return tx.Commit(ctx)
}

// UpdateMembershipRole updates a user's role in a group.
func (d *DB) UpdateMembershipRole(ctx context.Context, userID, groupID uuid.UUID, role string) error {
	query := `
		UPDATE user_group_memberships
		SET role = $3, updated_at = NOW()
		WHERE user_id = $1 AND group_id = $2
	`
	result, err := d.Pool.Exec(ctx, query, userID, groupID, role)
	if err != nil {
		return fmt.Errorf("failed to update membership role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

// GetUserMembership retrieves a specific membership.
func (d *DB) GetUserMembership(ctx context.Context, userID, groupID uuid.UUID) (*models.UserGroupMembership, error) {
	query := `
		SELECT
			ugm.id, ugm.user_id, ugm.group_id, ugm.is_primary, ugm.role,
			ugm.created_at, ugm.updated_at,
			g.id, g.name, g.slug, g.tier, g.parent_id, g.created_at, g.updated_at
		FROM user_group_memberships ugm
		JOIN groups g ON g.id = ugm.group_id
		WHERE ugm.user_id = $1 AND ugm.group_id = $2
	`
	var m models.UserGroupMembership
	var g models.Group
	err := d.Pool.QueryRow(ctx, query, userID, groupID).Scan(
		&m.ID, &m.UserID, &m.GroupID, &m.IsPrimary, &m.Role,
		&m.CreatedAt, &m.UpdatedAt,
		&g.ID, &g.Name, &g.Slug, &g.Tier, &g.ParentID, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMembershipNotFound
		}
		return nil, fmt.Errorf("failed to get membership: %w", err)
	}
	m.Group = &g
	return &m, nil
}
