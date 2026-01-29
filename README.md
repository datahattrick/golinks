# GoLinks

A self-hosted URL shortener for teams. Create memorable short links like `go/docs` that redirect to long URLs.

## Features

- **Simple Interface** - Clean, centered search with instant results via HTMX
- **Dark Mode** - Automatic detection with manual toggle
- **OIDC Authentication** - SSO integration with mock server for development
- **Click Tracking** - Track how often each link is used
- **Fast Search** - Trigram-based fuzzy search across keywords, URLs, and descriptions

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

### Basic Deployment

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
          volumeMounts:
            - name: config
              mountPath: /app/config.yaml
              subPath: config.yaml
      volumes:
        - name: config
          configMap:
            name: golinks-config
```

### Configuration via ConfigMap

For complex organization and group hierarchies, use a YAML configuration file:

```bash
# Create ConfigMap from config file
kubectl create configmap golinks-config --from-file=config.yaml
```

See `config.yaml.example` for the full configuration schema.

## Advanced Configuration

### Environment Variables

Simple settings are configured via environment variables. Copy `.env.example` to `.env`:

```bash
# Database
DATABASE_URL=postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

# OIDC Authentication
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://go.example.com/auth/callback

# Site Branding
SITE_TITLE=MyCompany Links
SITE_TAGLINE=Internal URL shortcuts
SITE_LOGO_URL=https://example.com/logo.png
```

### YAML Configuration

Complex hierarchical configuration (organizations, groups, tiers) is defined in a YAML file.
Set `CONFIG_FILE` env var or mount to `/app/config.yaml`:

```yaml
# Organizations
organizations:
  - slug: acme
    name: Acme Corporation
    domains: [acme.com]

# Groups with tier-based priority (1-99)
groups:
  - slug: acme-all
    name: All Employees
    tier: 25
    organization: acme

  - slug: acme-engineering
    name: Engineering
    tier: 50
    organization: acme
    parent: acme-all

# Auto-assign users to groups based on OIDC claims
auto_assignment:
  claim: groups
  mappings:
    "engineering": [acme-engineering]
```

Link resolution priority (highest to lowest):
1. **Personal links** (tier 100) - User's own shortcuts
2. **Team groups** (tier 75-99) - Team-specific links
3. **Department groups** (tier 50-74) - Department links
4. **Organization groups** (tier 25-49) - Company-wide links
5. **Global links** (tier 0) - Available to everyone

## Tech Stack

- **Backend**: Go, Fiber v3, pgx v5
- **Frontend**: HTMX, Tailwind CSS
- **Database**: PostgreSQL with trigram search
- **Auth**: OpenID Connect (go-oidc)
- **Dev Auth**: [mock-oauth2-server](https://github.com/navikt/mock-oauth2-server)

## License

MIT
