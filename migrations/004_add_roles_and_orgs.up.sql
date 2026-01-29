-- Create organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);

-- Add role and organization to users
-- Roles: 'user', 'org_mod', 'global_mod', 'admin'
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);

-- Update links table for scope and approval
-- Scope: 'global', 'org'
-- Status: 'pending', 'approved', 'rejected'
ALTER TABLE links ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'global';
ALTER TABLE links ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);
ALTER TABLE links ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'approved';
ALTER TABLE links ADD COLUMN IF NOT EXISTS submitted_by UUID REFERENCES users(id);
ALTER TABLE links ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id);
ALTER TABLE links ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_links_scope ON links(scope);
CREATE INDEX IF NOT EXISTS idx_links_organization_id ON links(organization_id);
CREATE INDEX IF NOT EXISTS idx_links_status ON links(status);
CREATE INDEX IF NOT EXISTS idx_links_scope_status ON links(scope, status);

-- Add unique constraint for keyword per scope/org combination
-- Global keywords must be unique globally
-- Org keywords must be unique within the org
DROP INDEX IF EXISTS idx_links_keyword;
CREATE UNIQUE INDEX idx_links_keyword_global ON links(keyword) WHERE scope = 'global';
CREATE UNIQUE INDEX idx_links_keyword_org ON links(keyword, organization_id) WHERE scope = 'org' AND organization_id IS NOT NULL;
