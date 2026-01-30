-- Drop the original unique constraint on keyword that was created in 002_create_links
-- This allows org keywords to shadow global keywords (the partial indexes handle scope-based uniqueness)
ALTER TABLE links DROP CONSTRAINT IF EXISTS links_keyword_key;

-- Ensure the partial unique indexes exist (these should already exist from migration 004)
CREATE UNIQUE INDEX IF NOT EXISTS idx_links_keyword_global ON links(keyword) WHERE scope = 'global';
CREATE UNIQUE INDEX IF NOT EXISTS idx_links_keyword_org ON links(keyword, organization_id) WHERE scope = 'org' AND organization_id IS NOT NULL;
