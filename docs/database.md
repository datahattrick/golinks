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
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### `organizations`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | TEXT | Organization name |
| `slug` | TEXT | URL-friendly identifier |
| `fallback_redirect_url` | TEXT | Fallback URL for missing keywords |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

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

### `groups`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | TEXT | Group name |
| `slug` | TEXT | URL-friendly identifier |
| `tier` | INTEGER | Resolution priority (1–99) |
| `parent_id` | UUID | Parent group FK |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### `user_group_memberships`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | FK → users |
| `group_id` | UUID | FK → groups |
| `is_primary` | BOOLEAN | Primary group flag |
| `role` | TEXT | Role within group |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

## Migrations

| # | Name | Description |
|---|------|-------------|
| 001 | `create_users` | Initial users table |
| 002 | `create_links` | Links table with trigram search |
| 003 | `add_username_and_user_links` | Username field and personal links |
| 004 | `add_roles_and_orgs` | Role-based access and organizations |
| 005 | `add_health_check_fields` | URL health monitoring columns |
| 006 | `add_groups_and_tiers` | Hierarchical group structure |
| 007 | `add_org_fallback_url` | Per-org fallback redirects |
| 008 | `fix_keyword_unique_constraint` | Allow org keywords to shadow global |
| 009 | `add_click_history` | Hourly click buckets for sparklines |
| 011 | `add_shared_links` | Personal link sharing between users |
