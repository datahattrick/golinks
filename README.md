# GoLinks

A self-hosted URL shortener for teams. Create memorable short links like `docs` that redirect to long URLs.

## Features

- OIDC authentication (Google, Entra, Okta, Keycloak, or a local mock)
- OIDC group-to-role mapping (auto-assign admin/moderator roles from IdP groups)
- Multi-tenant with organizations, scoped links, and role-based moderation
- JSON API at `/api/v1` alongside the HTMX UI
- Trigram-based fuzzy search
- URL health monitoring with email alerts
- Click tracking with 24-hour sparkline graphs
- Personal link sharing with accept/decline workflow and anti-spam limits
- Per-org fallback redirects and tier-based group resolution
- Configurable site banner with custom text and colors
- Structured logging with configurable log levels
- PostgreSQL-backed session store for multi-pod deployments
- Helm chart with OpenShift support
- Dark mode

## Quick Start

```bash
git clone https://github.com/datahattrick/golinks.git
cd golinks
make dev
```

Visit [http://localhost:3000](http://localhost:3000) and click **Login**. The mock OIDC server accepts any username/password.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go, Fiber v3, pgx v5 |
| Frontend | HTMX, Tailwind CSS v3 (build-time CLI) |
| Database | PostgreSQL with trigram search |
| Auth | OpenID Connect (go-oidc v3) |
| Migrations | golang-migrate (embedded SQL) |

## Documentation

| Topic | File |
|-------|------|
| Configuration & env vars | [docs/configuration.md](docs/configuration.md) |
| Deployment (Docker, Helm, K8s) | [docs/deployment.md](docs/deployment.md) |
| API reference (UI + JSON) | [docs/api.md](docs/api.md) |
| Database schema & migrations | [docs/database.md](docs/database.md) |
| Development & testing | [docs/development.md](docs/development.md) |
| Administration & roles | [docs/administration.md](docs/administration.md) |
| Usage guide | [docs/usage.md](docs/usage.md) |
| Troubleshooting | [docs/troubleshooting.md](docs/troubleshooting.md) |

## License

MIT — see [LICENSE](LICENSE) for details.
