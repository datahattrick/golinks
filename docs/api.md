# API Reference

## Probe Endpoints

Kubernetes liveness and readiness probes — no authentication required:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe (simple ping) |
| `GET` | `/readyz` | Readiness probe (checks database connectivity) |

Both return `200 {"status":"ok"}` on success. `/readyz` returns `503 {"status":"error","error":"database unavailable"}` if the database is unreachable.

## UI Routes (HTMX)

These routes serve the web UI. Partial responses are returned for HTMX requests (`HX-Request: true` header).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/` | Required | Home page with search and sparklines |
| `GET` | `/search` | Required | Search results page |
| `GET` | `/suggest` | Required | HTMX live suggestions |
| `GET` | `/browse` | Required | Browse all links |
| `GET` | `/new` | Required | Create link form |
| `POST` | `/links` | Required | Create link |
| `DELETE` | `/links/:id` | Required | Delete link |
| `GET` | `/my-links` | Required | Personal links list |
| `POST` | `/my-links` | Required | Create personal link |
| `DELETE` | `/my-links/:id` | Required | Delete personal link |
| `GET` | `/profile` | Required | User profile page |
| `GET` | `/moderation` | Mod+ | Moderation queue |
| `POST` | `/moderation/:id/approve` | Mod+ | Approve pending link |
| `POST` | `/moderation/:id/reject` | Mod+ | Reject pending link |
| `GET` | `/manage` | Mod+ | Link management with health and org badges |
| `GET` | `/manage/:id/edit` | Mod+ | Inline edit form |
| `PUT` | `/manage/:id` | Mod+ | Save link edits |
| `POST` | `/health/:id` | Mod+ | Trigger health check |
| `GET` | `/admin/users` | Admin | User management |
| `POST` | `/admin/users/:id/role` | Admin | Update user role |
| `POST` | `/admin/users/:id/org` | Admin | Update user org |
| `DELETE` | `/admin/users/:id` | Admin | Delete user |
| `GET` | `/random` | Required | Redirect to a random link |
| `GET` | `/go/:keyword` | See note | Redirect to URL |
| `GET` | `/:keyword` | See note | Short redirect |
| `GET` | `/auth/login` | None | Initiate OIDC login |
| `GET` | `/auth/callback` | None | OIDC callback |
| `GET` | `/auth/logout` | Required | Log out |

> In simple mode (personal and org links disabled), `/go/:keyword` does not require authentication.

## JSON API (`/api/v1`)

All endpoints use session-based authentication. Clients must authenticate via `/auth/login` first and include the session cookie on subsequent requests. All responses are JSON.

### Links

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/links` | Required | List/search links (`?q=`, `?scope=`, `?status=`) |
| `POST` | `/api/v1/links` | Required | Create a link |
| `GET` | `/api/v1/links/:id` | Required | Get a single link |
| `PUT` | `/api/v1/links/:id` | Required | Update a link |
| `DELETE` | `/api/v1/links/:id` | Required | Delete a link |
| `GET` | `/api/v1/links/check/:keyword` | Required | Check keyword availability |

### Resolve

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/resolve/:keyword` | See note | Resolve keyword to URL (no redirect) |

> In simple mode, this endpoint does not require authentication.

### Users (Admin)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/users` | Admin | List all users |
| `PUT` | `/api/v1/users/:id/role` | Admin | Update user role |
| `PUT` | `/api/v1/users/:id/org` | Admin | Update user organization |
| `DELETE` | `/api/v1/users/:id` | Admin | Delete a user |

### Moderation

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/moderation/pending` | Mod+ | List pending links |
| `POST` | `/api/v1/moderation/:id/approve` | Mod+ | Approve a pending link |
| `POST` | `/api/v1/moderation/:id/reject` | Mod+ | Reject a pending link |

### Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/health/:id` | Mod+ | Run a health check on a link |
