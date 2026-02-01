# GoLinks

A self-hosted URL shortener for teams. Create memorable short links like `go/docs` that redirect to long URLs.

---

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Docker Deployment](#docker-deployment)
  - [Manual Installation](#manual-installation)
  - [Kubernetes Deployment](#kubernetes-deployment)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [OIDC Authentication](#oidc-authentication)
  - [Organization Configuration](#organization-configuration)
  - [Site Branding](#site-branding)
  - [TLS/mTLS Configuration](#tlsmtls-configuration)
- [Usage Guide](#usage-guide)
  - [Creating Links](#creating-links)
  - [Link Scopes](#link-scopes)
  - [Link Resolution Priority](#link-resolution-priority)
  - [Searching Links](#searching-links)
  - [Redirects](#redirects)
- [Administration](#administration)
  - [User Roles](#user-roles)
  - [Moderation](#moderation)
  - [User Management](#user-management)
  - [Organization Management](#organization-management)
- [API Reference](#api-reference)
- [Database](#database)
  - [Schema Overview](#schema-overview)
  - [Migrations](#migrations)
- [Development](#development)
  - [Project Structure](#project-structure)
  - [Development Commands](#development-commands)
  - [Testing](#testing)
  - [Code Quality](#code-quality)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Simple Interface** - Clean, centered search with instant results via HTMX
- **Dark Mode** - Automatic detection with manual toggle
- **OIDC Authentication** - SSO integration with any OpenID Connect provider
- **Click Tracking** - Track how often each link is used
- **Fast Search** - Trigram-based fuzzy search across keywords, URLs, and descriptions
- **Multi-tenant** - Organizations with separate link namespaces
- **Link Scopes** - Global, Organization, and Personal links with priority resolution
- **Moderation** - Approval workflow for new links with role-based permissions
- **Health Checking** - Automatic URL health monitoring
- **Fallback Redirects** - Per-organization fallback URLs when links aren't found
- **Groups & Tiers** - Hierarchical group structure with tier-based link resolution

---

## Quick Start

The fastest way to get started is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/datahattrick/golinks.git
cd golinks

# Start PostgreSQL + mock OIDC server, then run the app
make dev
```

Visit http://localhost:3000 and click "Login" to authenticate with the mock OIDC server.

**Test credentials**: The mock server has interactive login - enter any username/password.

---

## Installation

### Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.22+ | For building from source |
| PostgreSQL | 14+ | Database backend |
| Docker | 20.10+ | For containerized deployment |
| Docker Compose | 2.0+ | For local development |

### Docker Deployment

#### Using Pre-built Images

Container images are automatically built and published to GitHub Container Registry:

```bash
# Pull latest image
docker pull ghcr.io/datahattrick/golinks:latest

# Pull specific version
docker pull ghcr.io/datahattrick/golinks:v1.0.0

# Run with environment variables
docker run -d \
  -p 3000:3000 \
  -e DATABASE_URL="postgres://user:pass@host:5432/golinks" \
  -e OIDC_ISSUER="https://accounts.google.com" \
  -e OIDC_CLIENT_ID="your-client-id" \
  -e OIDC_CLIENT_SECRET="your-client-secret" \
  -e SESSION_SECRET="your-32-char-secret-here" \
  ghcr.io/datahattrick/golinks:main
```

#### Using Docker Compose

```bash
# Start full stack (PostgreSQL + OIDC + App)
make docker-up

# Or with docker-compose directly
docker compose --profile full up -d

# Stop and clean up
make docker-down
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/datahattrick/golinks.git
cd golinks

# Install dependencies
go mod download

# Build the binary
go build -o golinks ./cmd/server

# Configure environment (see Configuration section)
export DATABASE_URL="postgres://golinks:golinks@localhost:5432/golinks"
export SESSION_SECRET="your-32-character-secret-here"

# Run the application
./golinks
```

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: golinks-secrets
type: Opaque
stringData:
  database-url: "postgres://user:password@postgres:5432/golinks?sslmode=require"
  session-secret: "your-32-character-secret-here"
  oidc-client-secret: "your-oidc-client-secret"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: golinks
  labels:
    app: golinks
spec:
  replicas: 2
  selector:
    matchLabels:
      app: golinks
  template:
    metadata:
      labels:
        app: golinks
    spec:
      containers:
        - name: golinks
          image: ghcr.io/datahattrick/golinks:main
          ports:
            - containerPort: 3000
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: database-url
            - name: SESSION_SECRET
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: session-secret
            - name: OIDC_ISSUER
              value: "https://accounts.google.com"
            - name: OIDC_CLIENT_ID
              value: "your-client-id"
            - name: OIDC_CLIENT_SECRET
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: oidc-client-secret
            - name: OIDC_REDIRECT_URL
              value: "https://go.example.com/auth/callback"
            - name: OIDC_ORG_CLAIM
              value: "hd"  # Google uses 'hd' for hosted domain
          resources:
            limits:
              memory: "256Mi"
              cpu: "500m"
            requests:
              memory: "128Mi"
              cpu: "100m"
          livenessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: golinks
spec:
  selector:
    app: golinks
  ports:
    - port: 80
      targetPort: 3000
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: golinks
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - go.example.com
      secretName: golinks-tls
  rules:
    - host: go.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: golinks
                port:
                  number: 80
```

---

## Configuration

All settings are configured via environment variables. Copy `.env.example` to `.env` for local development.

### Environment Variables

#### Core Settings

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ENV` | Environment mode (`development`, `production`) | `development` | No |
| `SERVER_ADDR` | Server listen address | `:3000` | No |
| `BASE_URL` | Public base URL for redirects | `http://localhost:3000` | No |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost:5432/golinks` | Yes |
| `SESSION_SECRET` | Cookie signing secret (min 32 chars) | - | Yes (production) |

#### OIDC Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OIDC_ISSUER` | OIDC provider URL | (disabled if empty) | No |
| `OIDC_CLIENT_ID` | OIDC client ID | - | If OIDC enabled |
| `OIDC_CLIENT_SECRET` | OIDC client secret | - | If OIDC enabled |
| `OIDC_REDIRECT_URL` | OAuth callback URL | `http://localhost:3000/auth/callback` | If OIDC enabled |
| `OIDC_ORG_CLAIM` | Claim name for organization extraction | `organisation` | No |

#### Site Branding

| Variable | Description | Default |
|----------|-------------|---------|
| `SITE_TITLE` | Site title displayed in header | `GoLinks` |
| `SITE_TAGLINE` | Tagline displayed on home page | `Fast URL shortcuts for your team` |
| `SITE_FOOTER` | Footer text | `GoLinks - Fast URL shortcuts for your team` |
| `SITE_LOGO_URL` | URL to logo image | (text only if empty) |

#### Features

| Variable | Description | Default |
|----------|-------------|---------|
| `ENABLE_RANDOM_KEYWORDS` | Enable "I'm Feeling Lucky" feature | `false` |
| `ENABLE_PERSONAL_LINKS` | Enable personal link scopes | `true` |
| `ENABLE_ORG_LINKS` | Enable organization link scopes | `true` |
| `CORS_ORIGINS` | Comma-separated allowed CORS origins | (none) |

**Simple Mode**: When both `ENABLE_PERSONAL_LINKS` and `ENABLE_ORG_LINKS` are set to `false`, the application runs in "simple mode":
- Only global links are available
- The redirect API (`/go/:keyword`) does not require authentication
- Frontend still requires OIDC authentication

#### Organization Fallbacks

| Variable | Description | Default |
|----------|-------------|---------|
| `ORG_FALLBACKS` | Per-org fallback redirect URLs | (none) |

Format: `org_slug=fallback_url,org2_slug=fallback_url2`

Example:
```bash
ORG_FALLBACKS=acme=https://other-golinks.acme.com/go/,corp=https://corp-links.example.com/go/
```

When a keyword is not found, users in the specified org are redirected to the fallback URL with the keyword appended.

#### TLS/mTLS

| Variable | Description | Default |
|----------|-------------|---------|
| `TLS_ENABLED` | Enable TLS | (disabled if empty) |
| `TLS_CERT_FILE` | Path to TLS certificate | - |
| `TLS_KEY_FILE` | Path to TLS private key | - |
| `TLS_CA_FILE` | CA file for client cert verification (mTLS) | - |
| `CLIENT_CERT_HEADER` | Header containing client cert CN (for ingress-terminated TLS) | - |

### OIDC Authentication

GoLinks supports any OpenID Connect compliant identity provider:

#### Google Workspace

```bash
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id.apps.googleusercontent.com
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=hd  # Google uses 'hd' for hosted domain
```

#### Microsoft Entra ID (Azure AD)

```bash
OIDC_ISSUER=https://login.microsoftonline.com/YOUR_TENANT_ID/v2.0
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=tid  # Tenant ID
```

#### Okta

```bash
OIDC_ISSUER=https://your-domain.okta.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=organization  # Or your custom claim
```

#### Keycloak

```bash
OIDC_ISSUER=https://keycloak.example.com/realms/your-realm
OIDC_CLIENT_ID=golinks
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback
OIDC_ORG_CLAIM=organisation
```

#### Development (Mock Server)

For local development, use the included mock OIDC server:

```bash
make dev  # Starts mock server on port 8080
```

The mock server provides interactive login - enter any username/password.

### Organization Configuration

Organizations allow teams to have their own link namespaces. They can be:

1. **Auto-created from OIDC claims** - Set `OIDC_ORG_CLAIM` to extract organization from tokens
2. **Configured with fallback URLs** - Set `ORG_FALLBACKS` for cross-instance redirects
3. **Managed via admin UI** - Admins can create/edit organizations

### Site Branding

Customize the look and feel:

```bash
SITE_TITLE=MyCompany Links
SITE_TAGLINE=Internal URL shortcuts for the team
SITE_LOGO_URL=https://example.com/logo.png
SITE_FOOTER=Powered by GoLinks
```

### TLS/mTLS Configuration

For production deployments with TLS:

```bash
TLS_ENABLED=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

For mutual TLS (client certificate authentication):

```bash
TLS_CA_FILE=/path/to/ca.pem
```

For ingress-terminated TLS with client cert forwarding:

```bash
CLIENT_CERT_HEADER=X-Client-CN
```

---

## Usage Guide

### Creating Links

1. Log in via the Login button
2. Use the form at the top of the page
3. Enter:
   - **Keyword**: The short name (e.g., `docs`, `wiki`, `hr`)
   - **URL**: The full destination URL
   - **Description**: (Optional) Helpful context about the link
   - **Scope**: Global, Organization, or Personal

### Link Scopes

| Scope | Visibility | Use Case |
|-------|------------|----------|
| **Global** | All users | Company-wide resources |
| **Organization** | Organization members | Team-specific links |
| **Personal** | Only you | Personal shortcuts |

### Link Resolution Priority

When a keyword is requested, GoLinks resolves it in this order:

1. **Personal links** (highest priority) - User's own shortcuts
2. **Organization links** - Shared within the user's organization
3. **Global links** (lowest priority) - Available to everyone

This means organization links can shadow global links with the same keyword, and personal links can shadow both.

### Searching Links

Type in the search box to filter links. Results update instantly via HTMX.

Search matches against:
- Keyword
- URL
- Description

The search uses PostgreSQL trigram matching for fuzzy results.

### Redirects

Access links via:

| URL Pattern | Example |
|-------------|---------|
| `/go/:keyword` | `http://go.example.com/go/docs` |
| `/:keyword` | `http://go.example.com/docs` |

Both patterns redirect to the same destination URL.

---

## Administration

### User Roles

| Role | Permissions |
|------|-------------|
| `user` | Create personal links, view approved links |
| `org_mod` | Moderate links within their organization |
| `global_mod` | Moderate all links (global + all organizations) |
| `admin` | Full access including user/org management |

### Moderation

Links may require approval before becoming active:

1. User submits a new link
2. Link enters `pending` status
3. Moderator reviews and approves/rejects
4. Approved links become active

Moderators can access the moderation queue via the navigation menu.

**Global moderators and admins** can moderate all pending links, including organization-specific ones.

### User Management

Administrators can manage users at `/users`:

- View all users
- Assign organizations
- Change user roles
- Delete users

### Organization Management

Organizations are automatically created when:
- Users authenticate with a new organization claim
- Fallback URLs are configured via `ORG_FALLBACKS`

Organizations can have:
- **Fallback redirect URL**: When a keyword isn't found, redirect to another GoLinks instance

---

## API Reference

### Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/` | Optional | Home page with search |
| `GET` | `/search` | Optional | HTMX search results |
| `GET` | `/new` | Required | Create link form |
| `POST` | `/links` | Required | Create link (HTMX) |
| `DELETE` | `/links/:id` | Required | Delete link (HTMX) |
| `GET` | `/profile` | Required | User profile page |
| `GET` | `/moderation` | Required (Mod+) | Moderation queue |
| `POST` | `/moderation/:id/approve` | Required (Mod+) | Approve pending link |
| `POST` | `/moderation/:id/reject` | Required (Mod+) | Reject pending link |
| `GET` | `/users` | Required (Admin) | User management |
| `PATCH` | `/users/:id` | Required (Admin) | Update user |
| `DELETE` | `/users/:id` | Required (Admin) | Delete user |
| `GET` | `/go/:keyword` | None | Redirect to URL |
| `GET` | `/:keyword` | None | Short redirect |
| `GET` | `/auth/login` | None | Initiate OIDC login |
| `GET` | `/auth/callback` | None | OIDC callback |
| `POST` | `/auth/logout` | Required | Log out |

### Response Formats

Most endpoints return HTMX partials for dynamic updates. The redirect endpoints return HTTP 302 redirects.

---

## Database

### Schema Overview

GoLinks uses PostgreSQL with the following main tables:

#### `users`
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `sub` | TEXT | OIDC subject identifier |
| `username` | TEXT | Extracted from PKI CN |
| `email` | TEXT | User email |
| `name` | TEXT | Display name |
| `picture` | TEXT | Profile picture URL |
| `role` | TEXT | User role |
| `organization_id` | UUID | FK to organizations |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

#### `organizations`
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | TEXT | Organization name |
| `slug` | TEXT | URL-friendly identifier |
| `fallback_redirect_url` | TEXT | Fallback URL for missing keywords |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

#### `links`
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `keyword` | TEXT | Short keyword |
| `url` | TEXT | Destination URL |
| `description` | TEXT | Link description |
| `scope` | TEXT | `global` or `org` |
| `organization_id` | UUID | FK for org-scoped links |
| `status` | TEXT | `pending`, `approved`, `rejected` |
| `click_count` | BIGINT | Number of clicks |
| `created_by` | UUID | Original creator |
| `submitted_by` | UUID | Submitter for approval |
| `reviewed_by` | UUID | Moderator |
| `reviewed_at` | TIMESTAMPTZ | Review timestamp |
| `health_status` | TEXT | `unknown`, `healthy`, `unhealthy` |
| `health_checked_at` | TIMESTAMPTZ | Last health check |
| `health_error` | TEXT | Health check error message |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

#### `groups` (Tier-based Hierarchy)
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | TEXT | Group name |
| `slug` | TEXT | URL-friendly identifier |
| `tier` | INTEGER | Resolution priority (1-99) |
| `parent_id` | UUID | Parent group FK |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

#### `user_group_memberships`
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | FK to users |
| `group_id` | UUID | FK to groups |
| `is_primary` | BOOLEAN | Primary group flag |
| `role` | TEXT | Role within group |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update |

### Migrations

Migrations are embedded in the binary and run automatically on startup using `golang-migrate`.

Current migrations:
1. `001_create_users` - Initial users table
2. `002_create_links` - Initial links table with trigram search
3. `003_add_username_and_user_links` - Username field and personal links
4. `004_add_roles_and_orgs` - Role-based access and organizations
5. `005_add_health_check_fields` - URL health monitoring
6. `006_add_groups_and_tiers` - Hierarchical group structure
7. `007_add_org_fallback_url` - Per-org fallback redirects
8. `008_fix_keyword_unique_constraint` - Allow org keywords to shadow global

---

## Development

### Project Structure

```
golinks/
├── cmd/server/
│   └── main.go              # Entry point, initializes DB and server
├── internal/
│   ├── config/              # Environment variable loading
│   │   └── config.go        # Configuration struct and loader
│   ├── db/                  # Database layer (pgx v5 pool)
│   │   ├── db.go            # Connection pool + migration runner
│   │   ├── links.go         # Link CRUD operations
│   │   ├── users.go         # User CRUD operations
│   │   ├── organizations.go # Organization operations
│   │   └── groups.go        # Group/tier operations
│   ├── handlers/            # HTTP handlers
│   │   ├── auth.go          # OIDC flow (login/callback/logout)
│   │   ├── links.go         # Link management + HTMX partials
│   │   ├── moderation.go    # Link approval workflow
│   │   ├── users.go         # User management (admin)
│   │   ├── profile.go       # User profile page
│   │   └── redirect.go      # Keyword → URL redirect
│   ├── middleware/
│   │   └── auth.go          # Session-based auth middleware
│   ├── models/              # Data structures
│   │   ├── user.go          # User model with role helpers
│   │   ├── link.go          # Link model with status helpers
│   │   ├── organization.go  # Organization model
│   │   └── group.go         # Group model for tiers
│   └── server/              # Server and API configuration
│       ├── server.go        # Fiber app setup, middleware, TLS config
│       └── routes.go        # Route registration
├── migrations/              # Embedded SQL migrations (golang-migrate)
├── views/                   # Go HTML templates
│   ├── layouts/
│   │   └── main.html        # Base layout with HTMX script
│   └── partials/            # HTMX partial templates
├── static/
│   ├── css/                 # Stylesheets
│   └── favicon.svg          # Site favicon
├── Dockerfile               # Multi-stage Docker build
├── docker-compose.yml       # Local development stack
├── Makefile                 # Build and development commands
└── go.mod                   # Go module definition
```

### Development Commands

```bash
# Start development environment
make dev              # Start services + run app with OIDC
make dev-no-auth      # Start db only + run app without OIDC

# Build
make build            # Build binary
make clean            # Remove build artifacts

# Database
make db-up            # Start PostgreSQL
make db-down          # Stop PostgreSQL
make db-shell         # Connect to PostgreSQL

# Docker
make services-up      # PostgreSQL + OIDC only
make docker-up        # Full stack in containers
make docker-down      # Stop and clean up

# View logs
make db-logs          # PostgreSQL logs
make oidc-logs        # OIDC server logs
```

### Testing

```bash
# Unit tests (no database required)
make test             # Alias for test-unit
make test-unit        # Run unit tests only

# Integration tests (requires database)
make test-integration # Run integration tests
make test-db          # Test database operations only

# All tests
make test-all         # Unit + integration tests
make test-ci          # CI mode with race detection

# Coverage
make test-cover       # Generate HTML coverage report
make test-cover-summary # Show coverage summary

# Specific packages
make test-pkg PKG=./internal/db
make test-models      # Test model logic
make test-handlers    # Test HTTP handlers
make test-middleware  # Test middleware
```

### Code Quality

```bash
make fmt              # Format code with gofmt
make vet              # Run go vet
make lint             # Run golangci-lint (if installed)
make check            # Run all checks (fmt, vet, lint, test)
make tidy             # Clean up go.mod
```

---

## Troubleshooting

### Common Issues

#### Database Connection Failed

```
Error: failed to connect to database
```

**Solution**: Ensure PostgreSQL is running and `DATABASE_URL` is correct:

```bash
make db-up
export DATABASE_URL="postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable"
```

#### OIDC Discovery Failed

```
Error: failed to get provider: Get ".../.well-known/openid-configuration": dial tcp: lookup ... no such host
```

**Solution**: Verify `OIDC_ISSUER` is correct and accessible. For local development:

```bash
make services-up  # Starts mock OIDC server
```

#### Session Secret Too Short

```
Warning: SESSION_SECRET is less than 32 characters
```

**Solution**: Set a secure session secret:

```bash
export SESSION_SECRET=$(openssl rand -base64 32)
```

#### Port Already in Use

```
Error: listen tcp :3000: bind: address already in use
```

**Solution**: Change the port or stop the conflicting process:

```bash
export SERVER_ADDR=:3001
# or
lsof -i :3000 | grep LISTEN | awk '{print $2}' | xargs kill
```

#### Links Not Resolving

- Check the link status is `approved`
- Verify the scope matches user's organization
- Check link resolution priority (personal > org > global)
- For org links, ensure user is in the correct organization

#### OIDC Claims Not Populating

- Enable development mode for debug logging: `ENV=development`
- Verify the OIDC provider returns the expected claims
- Check `OIDC_ORG_CLAIM` matches the actual claim name
- GoLinks fetches both ID token and userinfo endpoint claims

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `make check`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feature/my-feature`
7. Open a Pull Request

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new functionality
- Update documentation as needed

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go, Fiber v3, pgx v5 |
| Frontend | HTMX, Tailwind CSS |
| Database | PostgreSQL with trigram search |
| Authentication | OpenID Connect (go-oidc v3) |
| Migrations | golang-migrate with embedded SQL |
| Dev Auth | [mock-oauth2-server](https://github.com/navikt/mock-oauth2-server) |

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/datahattrick/golinks/issues)
- **Discussions**: [GitHub Discussions](https://github.com/datahattrick/golinks/discussions)
