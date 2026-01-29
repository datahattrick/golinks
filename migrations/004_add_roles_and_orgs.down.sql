-- Remove indexes
DROP INDEX IF EXISTS idx_links_keyword_org;
DROP INDEX IF EXISTS idx_links_keyword_global;
DROP INDEX IF EXISTS idx_links_scope_status;
DROP INDEX IF EXISTS idx_links_status;
DROP INDEX IF EXISTS idx_links_organization_id;
DROP INDEX IF EXISTS idx_links_scope;

-- Remove columns from links
ALTER TABLE links DROP COLUMN IF EXISTS reviewed_at;
ALTER TABLE links DROP COLUMN IF EXISTS reviewed_by;
ALTER TABLE links DROP COLUMN IF EXISTS submitted_by;
ALTER TABLE links DROP COLUMN IF EXISTS status;
ALTER TABLE links DROP COLUMN IF EXISTS organization_id;
ALTER TABLE links DROP COLUMN IF EXISTS scope;

-- Recreate original keyword index
CREATE UNIQUE INDEX IF NOT EXISTS idx_links_keyword ON links(keyword);

-- Remove columns from users
DROP INDEX IF EXISTS idx_users_organization_id;
DROP INDEX IF EXISTS idx_users_role;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- Drop organizations table
DROP INDEX IF EXISTS idx_organizations_slug;
DROP TABLE IF EXISTS organizations;
