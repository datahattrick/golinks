-- Composite index for GetNotificationsForUser: covers both the WHERE user_id filter
-- and the ORDER BY created_at DESC sort in a single index scan.
CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications(user_id, created_at DESC);

-- Index for CountPendingRequestsByUser and edit request status queries.
CREATE INDEX IF NOT EXISTS idx_link_edit_requests_user_status
    ON link_edit_requests(user_id, status);

-- Aggressive autovacuum on high-churn tables.
-- click_history and keyword_lookups receive batched upserts on every flush cycle.
-- Lower the scale factors so autovacuum triggers sooner, keeping dead tuple bloat
-- and WAL amplification from table bloat under control.
ALTER TABLE click_history SET (
    autovacuum_vacuum_scale_factor  = 0.01,
    autovacuum_analyze_scale_factor = 0.005,
    autovacuum_vacuum_cost_delay    = 2
);
ALTER TABLE keyword_lookups SET (
    autovacuum_vacuum_scale_factor  = 0.01,
    autovacuum_analyze_scale_factor = 0.005,
    autovacuum_vacuum_cost_delay    = 2
);
