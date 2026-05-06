package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const clickDedupTTL = time.Hour

// ShouldRecordClick returns true if this is the first click from actor on
// linkID within the dedup window (1 hour). Uses Redis SET NX so only one
// click per actor per link per hour reaches the write buffer.
//
// actor is a stable per-request identifier built from (in priority order):
// the user's OIDC sub, their session ID, or their IP address.
//
// Returns true unconditionally when Redis is not configured or on error, so
// clicks are never silently dropped due to infrastructure issues.
func (d *DB) ShouldRecordClick(ctx context.Context, actor string, linkID uuid.UUID) bool {
	if d.redis == nil {
		return true
	}
	key := fmt.Sprintf("click:%s:%s", actor, linkID)
	ok, err := d.redis.SetNX(ctx, key, 1, clickDedupTTL).Result()
	if err != nil {
		return true
	}
	return ok
}
