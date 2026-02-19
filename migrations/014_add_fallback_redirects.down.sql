-- Restore org-level fallback column
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS fallback_redirect_url TEXT;

-- Migrate data back: pick the first fallback per org
UPDATE organizations o SET fallback_redirect_url = fr.url
FROM (
    SELECT DISTINCT ON (organization_id) organization_id, url
    FROM fallback_redirects
    ORDER BY organization_id, created_at ASC
) fr
WHERE o.id = fr.organization_id;

-- Drop user preference column
ALTER TABLE users DROP COLUMN IF EXISTS fallback_redirect_id;

-- Drop fallback redirects table
DROP TABLE IF EXISTS fallback_redirects;
