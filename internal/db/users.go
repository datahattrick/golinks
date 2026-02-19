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
		RETURNING id, role, organization_id, fallback_redirect_id, created_at, updated_at
	`

	return d.Pool.QueryRow(ctx, query,
		user.Sub,
		nullIfEmpty(user.Username),
		user.Email,
		user.Name,
		user.Picture,
		nullIfEmpty(user.Role),
		user.OrganizationID,
	).Scan(&user.ID, &user.Role, &user.OrganizationID, &user.FallbackRedirectID, &user.CreatedAt, &user.UpdatedAt)
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
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, fallback_redirect_id, created_at, updated_at
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
		&user.FallbackRedirectID,
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
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, fallback_redirect_id, created_at, updated_at
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
		&user.FallbackRedirectID,
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
		SELECT id, sub, COALESCE(username, ''), email, name, picture, role, organization_id, fallback_redirect_id, created_at, updated_at
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
		&user.FallbackRedirectID,
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

// UpdateUserRoleFromOIDC persists both the raw OIDC-mapped role and the resolved
// final role for a user.  mappedRole is the group-derived value (admin | moderator | user);
// finalRole is what actually goes into the role column (admin | global_mod | org_mod | user).
func (d *DB) UpdateUserRoleFromOIDC(ctx context.Context, userID uuid.UUID, mappedRole, finalRole string) error {
	query := `UPDATE users SET role = $1, oidc_mapped_role = $2, updated_at = NOW() WHERE id = $3`
	_, err := d.Pool.Exec(ctx, query, finalRole, mappedRole, userID)
	return err
}

// PromoteOrgModerators sets role = org_mod for every user in orgID whose
// oidc_mapped_role is "moderator".  This is called once when a new organisation
// is created so that users who were already mapped to the moderator role but
// had not yet re-logged-in get the correct org_mod assignment immediately.
func (d *DB) PromoteOrgModerators(ctx context.Context, orgID uuid.UUID) error {
	query := `
		UPDATE users SET role = 'org_mod', updated_at = NOW()
		WHERE organization_id = $1 AND oidc_mapped_role = 'moderator'
	`
	_, err := d.Pool.Exec(ctx, query, orgID)
	return err
}

// UpdateUserOrganization updates a user's organization membership.
func (d *DB) UpdateUserOrganization(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID) error {
	query := `UPDATE users SET organization_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := d.Pool.Exec(ctx, query, orgID, userID)
	return err
}

// DeleteUser deletes a user by ID.
func (d *DB) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := d.Pool.Exec(ctx, query, userID)
	return err
}

// UserWithOrg represents a user with their organization details.
type UserWithOrg struct {
	models.User
	OrganizationName string
	OrganizationSlug string
}

// GetAllUsersWithOrgs retrieves all users with their organization info.
func (d *DB) GetAllUsersWithOrgs(ctx context.Context) ([]UserWithOrg, error) {
	query := `
		SELECT u.id, u.sub, COALESCE(u.username, ''), u.email, u.name, u.picture,
			   u.role, u.organization_id, u.fallback_redirect_id, u.created_at, u.updated_at,
			   COALESCE(o.name, ''), COALESCE(o.slug, '')
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		ORDER BY u.name ASC, u.email ASC
	`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserWithOrg
	for rows.Next() {
		var u UserWithOrg
		if err := rows.Scan(
			&u.ID, &u.Sub, &u.Username, &u.Email, &u.Name, &u.Picture,
			&u.Role, &u.OrganizationID, &u.FallbackRedirectID, &u.CreatedAt, &u.UpdatedAt,
			&u.OrganizationName, &u.OrganizationSlug,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// SearchUsers searches users by name or email, excluding the requesting user.
func (d *DB) SearchUsers(ctx context.Context, query string, excludeID uuid.UUID, limit int) ([]models.User, error) {
	q := `
		SELECT id, COALESCE(username, ''), email, name, sub
		FROM users
		WHERE id != $1
		  AND (
		    name ILIKE '%' || $2 || '%'
		    OR email ILIKE '%' || $2 || '%'
		    OR username ILIKE '%' || $2 || '%'
		    OR sub ILIKE '%' || $2 || '%'
		  )
		ORDER BY name ASC
		LIMIT $3
	`

	rows, err := d.Pool.Query(ctx, q, excludeID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Name, &u.Sub); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// GetUserCount returns the total number of users.
func (d *DB) GetUserCount(ctx context.Context) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// GetGlobalModeratorEmails returns email addresses for global moderators (admins and global mods).
func (d *DB) GetGlobalModeratorEmails(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT email FROM users
		WHERE email != '' AND email IS NOT NULL
		AND role IN ('admin', 'global_mod')
	`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// GetOrgModeratorEmails returns email addresses for moderators of a specific organization.
// Includes admins, global mods, and org mods for that org.
func (d *DB) GetOrgModeratorEmails(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT email FROM users
		WHERE email != '' AND email IS NOT NULL
		AND (
			role IN ('admin', 'global_mod')
			OR (role = 'org_mod' AND organization_id = $1)
		)
	`

	rows, err := d.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// GetUserCountByOrg returns user count grouped by organization.
func (d *DB) GetUserCountByOrg(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT COALESCE(o.slug, 'none'), COUNT(u.id)
		FROM users u
		LEFT JOIN organizations o ON u.organization_id = o.id
		GROUP BY o.slug
		ORDER BY COUNT(u.id) DESC
	`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			return nil, err
		}
		counts[slug] = count
	}

	return counts, rows.Err()
}
