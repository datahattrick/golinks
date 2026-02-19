-- Add reason column to links (for initial submission reason)
ALTER TABLE links ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

-- Create link_edit_requests table for "stay live" edit flow
CREATE TABLE IF NOT EXISTS link_edit_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Only one pending edit request per link per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_link_edit_requests_pending
    ON link_edit_requests (link_id, user_id) WHERE status = 'pending';
