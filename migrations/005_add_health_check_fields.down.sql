-- Remove health check fields from links table
DROP INDEX IF EXISTS idx_links_health_status;
ALTER TABLE links DROP COLUMN IF EXISTS health_error;
ALTER TABLE links DROP COLUMN IF EXISTS health_checked_at;
ALTER TABLE links DROP COLUMN IF EXISTS health_status;

-- Remove health check fields from user_links table
ALTER TABLE user_links DROP COLUMN IF EXISTS health_error;
ALTER TABLE user_links DROP COLUMN IF EXISTS health_checked_at;
ALTER TABLE user_links DROP COLUMN IF EXISTS health_status;
