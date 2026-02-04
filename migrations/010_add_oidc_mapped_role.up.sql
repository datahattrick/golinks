-- Store the raw OIDC-group-derived role (admin | moderator | user) so that
-- when a new organisation is created we can promote existing moderator-mapped
-- users in that org to org_mod without waiting for them to re-login.
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_mapped_role TEXT;

CREATE INDEX IF NOT EXISTS idx_users_oidc_mapped_role ON users(oidc_mapped_role);
