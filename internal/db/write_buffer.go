package db

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// historyKey identifies a (link, hour) bucket in click_history.
type historyKey struct {
	linkID     uuid.UUID
	hourBucket time.Time
}

// kwLookupKey identifies a (keyword, outcome) pair in keyword_lookups.
type kwLookupKey struct {
	keyword string
	outcome string
}

// writeBuffer accumulates hot counter increments in memory and flushes them
// to the database in batches, dramatically reducing WAL write amplification.
type writeBuffer struct {
	mu             sync.Mutex
	linkClicks     map[uuid.UUID]int64   // delta for links.click_count
	userLinkClicks map[uuid.UUID]int64   // delta for user_links.click_count (by ID)
	historyClicks  map[historyKey]int64  // delta for click_history.click_count
	kwLookups      map[kwLookupKey]int64 // delta for keyword_lookups.count
}

func newWriteBuffer() *writeBuffer {
	return &writeBuffer{
		linkClicks:     make(map[uuid.UUID]int64),
		userLinkClicks: make(map[uuid.UUID]int64),
		historyClicks:  make(map[historyKey]int64),
		kwLookups:      make(map[kwLookupKey]int64),
	}
}

func (b *writeBuffer) recordLinkClick(id uuid.UUID) {
	hour := time.Now().UTC().Truncate(time.Hour)
	b.mu.Lock()
	b.linkClicks[id]++
	b.historyClicks[historyKey{id, hour}]++
	b.mu.Unlock()
}

func (b *writeBuffer) recordUserLinkClick(id uuid.UUID) {
	b.mu.Lock()
	b.userLinkClicks[id]++
	b.mu.Unlock()
}

func (b *writeBuffer) recordKeywordLookup(keyword, outcome string) {
	b.mu.Lock()
	b.kwLookups[kwLookupKey{keyword, outcome}]++
	b.mu.Unlock()
}

// swap atomically drains all pending writes and returns them for flushing.
// The buffer is reset to empty maps immediately, so new writes during flush
// are accumulated for the next cycle.
func (b *writeBuffer) swap() (
	links map[uuid.UUID]int64,
	userLinks map[uuid.UUID]int64,
	history map[historyKey]int64,
	kw map[kwLookupKey]int64,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	links, b.linkClicks = b.linkClicks, make(map[uuid.UUID]int64)
	userLinks, b.userLinkClicks = b.userLinkClicks, make(map[uuid.UUID]int64)
	history, b.historyClicks = b.historyClicks, make(map[historyKey]int64)
	kw, b.kwLookups = b.kwLookups, make(map[kwLookupKey]int64)
	return
}

func (d *DB) flush(ctx context.Context) {
	links, userLinks, history, kw := d.buf.swap()

	total := len(links) + len(userLinks) + len(history) + len(kw)
	if total == 0 {
		return
	}

	batch := &pgx.Batch{}

	for id, delta := range links {
		batch.Queue(
			`UPDATE links SET click_count = click_count + $2 WHERE id = $1`,
			id, delta,
		)
	}
	for id, delta := range userLinks {
		batch.Queue(
			`UPDATE user_links SET click_count = click_count + $2 WHERE id = $1`,
			id, delta,
		)
	}
	for k, delta := range history {
		batch.Queue(`
			INSERT INTO click_history (link_id, hour_bucket, click_count)
			VALUES ($1, $2, $3)
			ON CONFLICT (link_id, hour_bucket) DO UPDATE
			  SET click_count = click_history.click_count + EXCLUDED.click_count`,
			k.linkID, k.hourBucket, delta,
		)
	}
	for k, delta := range kw {
		batch.Queue(`
			INSERT INTO keyword_lookups (keyword, outcome, count, last_seen_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (keyword, outcome) DO UPDATE
			  SET count        = keyword_lookups.count + EXCLUDED.count,
			      last_seen_at = NOW()`,
			k.keyword, k.outcome, delta,
		)
	}

	results := d.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			slog.Warn("write buffer flush error", "error", err)
		}
	}

	slog.Debug("write buffer flushed",
		"links", len(links),
		"user_links", len(userLinks),
		"history", len(history),
		"keyword_lookups", len(kw),
	)
}

// StartWriteBuffer starts a background goroutine that flushes buffered writes
// to the database at the given interval. Call FlushWriteBuffer before closing.
func (d *DB) StartWriteBuffer(ctx context.Context, interval time.Duration) {
	slog.Info("write buffer started", "flush_interval", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.flush(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// FlushWriteBuffer performs a final synchronous flush of all buffered writes.
// Must be called before closing the database connection on shutdown.
func (d *DB) FlushWriteBuffer(ctx context.Context) {
	slog.Info("flushing write buffer before shutdown")
	d.flush(ctx)
}
