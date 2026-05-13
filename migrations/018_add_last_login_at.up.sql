-- Track the most recent successful OIDC sign-in for each user.
-- Nullable so existing users show "Never" on the admin page until next login.
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;
