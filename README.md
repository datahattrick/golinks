# GoLinks

A self-hosted URL shortener for teams. Create memorable short links like `go/docs` that redirect to long URLs.

## Features

- **Simple Interface** - Clean, centered search with instant results via HTMX
- **Dark Mode** - Automatic detection with manual toggle
- **OIDC Authentication** - SSO integration with mock server for development
- **Click Tracking** - Track how often each link is used
- **Fast Search** - Trigram-based fuzzy search across keywords, URLs, and descriptions
- **Multi-tenant** - Organizations with separate link namespaces
- **Link Scopes** - Global, Organization, and Personal links

## Quick Start

```bash
# Start PostgreSQL + mock OIDC server, then run the app
make dev
```

Visit http://localhost:3000 and click "Login" to authenticate with the mock OIDC server.

**Test credentials**: The mock server has interactive login - enter any username/password.

## Requirements

- Go 1.22+
- PostgreSQL 14+
- Docker (for PostgreSQL and mock OIDC server)

## Development

```bash
make dev          # Start services + run app with OIDC
make dev-no-auth  # Start db only + run app without OIDC
make build        # Build binary
make test         # Run tests
make db-shell     # Connect to PostgreSQL
make oidc-logs    # View OIDC server logs
```

## Usage

### Redirects

- `http://localhost:3000/go/docs` → Redirects to the URL for "docs"
- `http://localhost:3000/docs` → Same redirect (shorter form)

### Link Scopes

Links are resolved in priority order:
1. **Personal** - User's own shortcuts (highest priority)
2. **Organization** - Shared within the user's organization
3. **Global** - Available to everyone (lowest priority)

### Creating Links

1. Log in via the Login button
2. Use the form at the top of the page
3. Enter a keyword, URL, and optional description

### Searching

Type in the search box to filter links. Results update instantly via HTMX.

## Docker

```bash
make services-up    # PostgreSQL + OIDC only
make docker-up      # Full stack in containers
make docker-down    # Stop and clean up
```

### Container Images

Container images are automatically built and published to GitHub Container Registry on push to `main` and tags.

```bash
# Pull latest image
docker pull ghcr.io/OWNER/golinks:main

# Pull specific version
docker pull ghcr.io/OWNER/golinks:v1.0.0
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: golinks
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
          image: ghcr.io/OWNER/golinks:main
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
            - name: OIDC_ORG_CLAIM
              value: "hd"  # Google uses 'hd' for hosted domain
```

## Configuration

All settings are configured via environment variables. Copy `.env.example` to `.env`:

```bash
# Database
DATABASE_URL=postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

# OIDC Authentication
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback

# Organization claim - extracts org from OIDC token
# Examples: "hd" (Google), "org", "organization", "tenant"
OIDC_ORG_CLAIM=hd

# Site Branding
SITE_TITLE=MyCompany Links
SITE_TAGLINE=Internal URL shortcuts
SITE_LOGO_URL=https://example.com/logo.png

# Features
ENABLE_RANDOM_KEYWORDS=true
```

### Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost:5432/golinks` |
| `SERVER_ADDR` | Listen address | `:3000` |
| `OIDC_ISSUER` | OIDC provider URL | (disabled if empty) |
| `OIDC_CLIENT_ID` | OIDC client ID | |
| `OIDC_CLIENT_SECRET` | OIDC client secret | |
| `OIDC_REDIRECT_URL` | OAuth callback URL | `http://localhost:3000/auth/callback` |
| `OIDC_ORG_CLAIM` | Claim name for organization | (disabled if empty) |
| `SESSION_SECRET` | Cookie signing secret (min 32 chars) | |
| `SITE_TITLE` | Site title | `GoLinks` |
| `SITE_TAGLINE` | Site tagline | `Fast URL shortcuts for your team` |
| `SITE_LOGO_URL` | Logo image URL | (text only if empty) |
| `ENABLE_RANDOM_KEYWORDS` | Enable "I'm Feeling Lucky" | `false` |

## Tech Stack

- **Backend**: Go, Fiber v3, pgx v5
- **Frontend**: HTMX, Tailwind CSS
- **Database**: PostgreSQL with trigram search
- **Auth**: OpenID Connect (go-oidc)
- **Dev Auth**: [mock-oauth2-server](https://github.com/navikt/mock-oauth2-server)

## License

MIT
