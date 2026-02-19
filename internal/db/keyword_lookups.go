package db

import (
	"context"

	"golinks/internal/models"
)

// IncrementKeywordLookup upserts a keyword lookup count by outcome.
func (d *DB) IncrementKeywordLookup(ctx context.Context, keyword, outcome string) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO keyword_lookups (keyword, outcome, count, last_seen_at)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (keyword, outcome) DO UPDATE
		SET count = keyword_lookups.count + 1, last_seen_at = NOW()
	`, keyword, outcome)
	return err
}

// GetAllKeywordLookups returns all keyword lookup rows for metrics export.
func (d *DB) GetAllKeywordLookups(ctx context.Context) ([]models.KeywordLookup, error) {
	rows, err := d.Pool.Query(ctx, `SELECT keyword, outcome, count, last_seen_at FROM keyword_lookups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lookups []models.KeywordLookup
	for rows.Next() {
		var l models.KeywordLookup
		if err := rows.Scan(&l.Keyword, &l.Outcome, &l.Count, &l.LastSeenAt); err != nil {
			return nil, err
		}
		lookups = append(lookups, l)
	}
	return lookups, rows.Err()
}
