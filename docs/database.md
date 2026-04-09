# Database

GoLinks uses PostgreSQL with the `pg_trgm` extension for fuzzy search. Migrations are embedded in the binary and run automatically on startup via `golang-migrate`.

## Schema

### `users`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `sub` | TEXT | OIDC subject identifier |
| `username` | TEXT | Extracted from PKI CN |
| `email` | TEXT | User email |
| `name` | TEXT | Display name |
| `picture` | TEXT | Profile picture URL |
| `role` | TEXT | User role (`user`, `org_mod`, `global_mod`, `admin`) |
| `organization_id` | UUID | FK → organizations |
| `fallback_redirect_id` | UUID | FK → fallback_redirects (user's chosen fallback, nullable) |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### `organizations`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | TEXT | Organization name |
| `slug` | TEXT | URL-friendly identifier |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### `fallback_redirects`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `organization_id` | UUID | FK to organizations |
| `name` | TEXT | Display name for this option |
| `url` | TEXT | URL prefix (keyword is appended) |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

Unique constraint on `(organization_id, name)`. Users reference a fallback via `users.fallback_redirect_id` (nullable, ON DELETE SET NULL).

### `links`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `keyword` | TEXT | Short keyword |
| `url` | TEXT | Destination URL |
| `description` | TEXT | Link description |
| `scope` | TEXT | `global` or `org` |
| `organization_id` | UUID | FK for org-scoped links |
| `status` | TEXT | `pending`, `approved`, `rejected` |
| `click_count` | BIGINT | Total click count |
| `created_by` | UUID | Original creator |
| `submitted_by` | UUID | Submitter for approval |
| `reviewed_by` | UUID | Reviewing moderator |
| `reviewed_at` | TIMESTAMPTZ | Review timestamp |
| `health_status` | TEXT | `unknown`, `healthy`, `unhealthy` |
| `health_checked_at` | TIMESTAMPTZ | Last health check |
| `health_error` | TEXT | Health check error message |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### `click_history`

| Column | Type | Description |
|--------|------|-------------|
| `link_id` | UUID | FK → links (CASCADE on delete) |
| `hour_bucket` | TIMESTAMPTZ | Hour the clicks occurred in |
| `click_count` | INTEGER | Clicks in that hour |

Unique constraint on `(link_id, hour_bucket)`. Powers the 24-hour sparkline graphs on the home page.

### `shared_links`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `sender_id` | UUID | FK → users (CASCADE) |
| `recipient_id` | UUID | FK → users (CASCADE) |
| `keyword` | VARCHAR(255) | Link keyword being shared |
| `url` | TEXT | Destination URL |
| `description` | TEXT | Optional description |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

Constraints: `no_self_share` CHECK prevents sender = recipient. `unique_pending_share` UNIQUE on (sender_id, recipient_id, keyword) prevents duplicate offers. Indexes on `sender_id` and `recipient_id`.

## Migrations

| # | Name | Description |
|---|------|-------------|
| 001 | `create_users` | Initial users table |
| 002 | `create_links` | Links table with trigram search |
| 003 | `add_username_and_user_links` | Username field and personal links |
| 004 | `add_roles_and_orgs` | Role-based access and organizations |
| 005 | `add_health_check_fields` | URL health monitoring columns |
| 006 | `add_groups_and_tiers` | Hierarchical group structure (later removed) |
| 007 | `add_org_fallback_url` | Per-org fallback redirects |
| 008 | `fix_keyword_unique_constraint` | Allow org keywords to shadow global |
| 009 | `add_click_history` | Hourly click buckets for sparklines |
| 010 | `add_oidc_mapped_role` | Intermediate OIDC group role column |
| 011 | `add_shared_links` | Personal link sharing between users |
| 012 | `add_authorship_and_requests` | Submission reason + link edit requests |
| 013 | `add_keyword_lookups` | Keyword resolution outcome tracking |
| 014 | `add_fallback_redirects` | Named per-org fallback redirect options |
| 015 | `remove_groups_and_tiers` | Drop unused group/tier tables |
| 016 | `add_notifications` | In-app notification bell |
| 017 | `indexes_and_autovacuum` | Composite indexes + aggressive autovacuum on hot tables |

## Write Buffer

Click counts (`links.click_count`, `user_links.click_count`, `click_history`) and keyword lookup counts (`keyword_lookups`) are **not** written to the database on every request. Instead, increments are accumulated in an in-memory buffer and flushed to PostgreSQL in a single batch every 5 seconds.

This eliminates per-request WAL writes for counters that don't need real-time accuracy. On graceful shutdown, the buffer is flushed before the database connection is closed so no counts are lost.

The flush interval is fixed at 5 seconds. Click counts may lag by up to that amount, which is imperceptible in practice.
