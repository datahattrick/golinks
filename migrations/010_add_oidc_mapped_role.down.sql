DROP INDEX IF EXISTS idx_users_oidc_mapped_role;

ALTER TABLE users DROP COLUMN IF EXISTS oidc_mapped_role;
