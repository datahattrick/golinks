-- Add health check fields to links table
ALTER TABLE links ADD COLUMN health_status TEXT DEFAULT 'unknown';
ALTER TABLE links ADD COLUMN health_checked_at TIMESTAMPTZ;
ALTER TABLE links ADD COLUMN health_error TEXT;
CREATE INDEX idx_links_health_status ON links(health_status);

-- Add health check fields to user_links table
ALTER TABLE user_links ADD COLUMN health_status TEXT DEFAULT 'unknown';
ALTER TABLE user_links ADD COLUMN health_checked_at TIMESTAMPTZ;
ALTER TABLE user_links ADD COLUMN health_error TEXT;
