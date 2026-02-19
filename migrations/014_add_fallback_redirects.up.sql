CREATE TABLE IF NOT EXISTS fallback_redirects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    url             TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name)
);

-- Migrate existing org fallback URLs into the new table
INSERT INTO fallback_redirects (organization_id, name, url)
SELECT id, name || ' default', fallback_redirect_url
FROM organizations
WHERE fallback_redirect_url IS NOT NULL AND fallback_redirect_url != ''
ON CONFLICT DO NOTHING;

-- User preference (nullable = no fallback, ON DELETE SET NULL clears preference if option removed)
ALTER TABLE users ADD COLUMN IF NOT EXISTS fallback_redirect_id UUID
    REFERENCES fallback_redirects(id) ON DELETE SET NULL;

-- Drop the old org-level column
ALTER TABLE organizations DROP COLUMN IF EXISTS fallback_redirect_url;
