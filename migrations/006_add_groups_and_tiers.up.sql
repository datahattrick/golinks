-- Create groups table for tier-based hierarchy
-- Tier 0 = global, 1-99 = custom groups (orgs, teams), 100 = personal
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    tier INTEGER NOT NULL DEFAULT 50,
    parent_id UUID REFERENCES groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_tier CHECK (tier >= 1 AND tier <= 99)
);

CREATE INDEX IF NOT EXISTS idx_groups_slug ON groups(slug);
CREATE INDEX IF NOT EXISTS idx_groups_tier ON groups(tier);
CREATE INDEX IF NOT EXISTS idx_groups_parent_id ON groups(parent_id);

-- Create user-group membership table
CREATE TABLE IF NOT EXISTS user_group_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_user_group_memberships_user_id ON user_group_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_user_group_memberships_group_id ON user_group_memberships(group_id);
CREATE INDEX IF NOT EXISTS idx_user_group_memberships_is_primary ON user_group_memberships(user_id, is_primary) WHERE is_primary = true;

-- Create group_links table for group-scoped links
CREATE TABLE IF NOT EXISTS group_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    keyword TEXT NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'approved',
    click_count INTEGER NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    health_status TEXT NOT NULL DEFAULT 'unknown',
    health_checked_at TIMESTAMPTZ,
    health_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, keyword)
);

CREATE INDEX IF NOT EXISTS idx_group_links_group_id ON group_links(group_id);
CREATE INDEX IF NOT EXISTS idx_group_links_keyword ON group_links(keyword);
CREATE INDEX IF NOT EXISTS idx_group_links_status ON group_links(status);
CREATE INDEX IF NOT EXISTS idx_group_links_group_keyword_status ON group_links(group_id, keyword, status);

-- Migrate existing organizations to groups at tier 50
INSERT INTO groups (id, name, slug, tier, created_at, updated_at)
SELECT id, name, slug, 50, created_at, updated_at
FROM organizations
ON CONFLICT (slug) DO NOTHING;

-- Migrate existing user-organization memberships to user_group_memberships
INSERT INTO user_group_memberships (user_id, group_id, is_primary, role, created_at, updated_at)
SELECT
    u.id as user_id,
    u.organization_id as group_id,
    true as is_primary,
    CASE
        WHEN u.role = 'org_mod' THEN 'moderator'
        ELSE 'member'
    END as role,
    u.created_at,
    u.updated_at
FROM users u
WHERE u.organization_id IS NOT NULL
ON CONFLICT (user_id, group_id) DO NOTHING;

-- Migrate org-scoped links to group_links
INSERT INTO group_links (
    id, group_id, keyword, url, description, status, click_count,
    created_by, submitted_by, reviewed_by, reviewed_at,
    health_status, health_checked_at, health_error,
    created_at, updated_at
)
SELECT
    id, organization_id, keyword, url, description, status, click_count,
    created_by, submitted_by, reviewed_by, reviewed_at,
    health_status, health_checked_at, health_error,
    created_at, updated_at
FROM links
WHERE scope = 'org' AND organization_id IS NOT NULL
ON CONFLICT (group_id, keyword) DO NOTHING;
