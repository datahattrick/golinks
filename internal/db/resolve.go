package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"golinks/internal/models"
)

// ResolveKeywordForUser resolves a keyword using the scope hierarchy:
// personal (user_links) > org (links scope=org) > global (links scope=global).
// Returns the first matching link, or ErrLinkNotFound if none exists.
func (d *DB) ResolveKeywordForUser(ctx context.Context, userID *uuid.UUID, orgID *uuid.UUID, keyword string) (*models.ResolvedLink, error) {
	resolved := &models.ResolvedLink{}

	if userID == nil {
		// Unauthenticated: global links only
		err := d.Pool.QueryRow(ctx, `
			SELECT id, url, 'global'::text
			FROM links
			WHERE keyword = $1 AND scope = 'global' AND status = 'approved'
			LIMIT 1
		`, keyword).Scan(&resolved.ID, &resolved.URL, &resolved.Source)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrLinkNotFound
			}
			return nil, fmt.Errorf("failed to resolve keyword: %w", err)
		}
		return resolved, nil
	}

	if orgID != nil {
		// Authenticated with org: personal > org > global
		err := d.Pool.QueryRow(ctx, `
			SELECT id, url, source FROM (
				SELECT id, url, 'personal'::text AS source, 1 AS priority
				FROM user_links
				WHERE user_id = $1 AND keyword = $3
				UNION ALL
				SELECT id, url, 'org'::text AS source, 2 AS priority
				FROM links
				WHERE keyword = $3 AND scope = 'org' AND organization_id = $2 AND status = 'approved'
				UNION ALL
				SELECT id, url, 'global'::text AS source, 3 AS priority
				FROM links
				WHERE keyword = $3 AND scope = 'global' AND status = 'approved'
			) combined
			ORDER BY priority ASC
			LIMIT 1
		`, userID, orgID, keyword).Scan(&resolved.ID, &resolved.URL, &resolved.Source)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrLinkNotFound
			}
			return nil, fmt.Errorf("failed to resolve keyword: %w", err)
		}
		return resolved, nil
	}

	// Authenticated without org: personal > global
	err := d.Pool.QueryRow(ctx, `
		SELECT id, url, source FROM (
			SELECT id, url, 'personal'::text AS source, 1 AS priority
			FROM user_links
			WHERE user_id = $1 AND keyword = $2
			UNION ALL
			SELECT id, url, 'global'::text AS source, 2 AS priority
			FROM links
			WHERE keyword = $2 AND scope = 'global' AND status = 'approved'
		) combined
		ORDER BY priority ASC
		LIMIT 1
	`, userID, keyword).Scan(&resolved.ID, &resolved.URL, &resolved.Source)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("failed to resolve keyword: %w", err)
	}
	return resolved, nil
}

// IncrementResolvedLinkClickCount records a click for a resolved link. Writes
// are buffered in memory and flushed in batches to reduce WAL write frequency.
func (d *DB) IncrementResolvedLinkClickCount(_ context.Context, resolved *models.ResolvedLink, userID *uuid.UUID) error {
	switch resolved.Source {
	case "personal":
		if userID != nil {
			d.buf.recordUserLinkClick(resolved.ID)
		}
	case "org", "global":
		d.buf.recordLinkClick(resolved.ID)
	}
	return nil
}
