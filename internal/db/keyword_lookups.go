package db

import (
	"context"

	"golinks/internal/models"
)

// IncrementKeywordLookup records a keyword lookup by outcome. The write is
// buffered in memory and flushed to the database in batches.
func (d *DB) IncrementKeywordLookup(_ context.Context, keyword, outcome string) error {
	d.buf.recordKeywordLookup(keyword, outcome)
	return nil
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
