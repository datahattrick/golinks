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

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Database (matches docker-compose defaults)
DATABASE_URL=postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

# OIDC - Local mock server (default for make dev)
OIDC_ISSUER=http://localhost:8080/golinks
OIDC_CLIENT_ID=golinks-app
OIDC_CLIENT_SECRET=secret
OIDC_REDIRECT_URL=http://localhost:3000/auth/callback

# OIDC - Production (e.g., Google)
# OIDC_ISSUER=https://accounts.google.com
# OIDC_CLIENT_ID=your-client-id
# OIDC_CLIENT_SECRET=your-client-secret
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

## Tech Stack

- **Backend**: Go, Fiber v3, pgx v5
- **Frontend**: HTMX, Tailwind CSS
- **Database**: PostgreSQL with trigram search
- **Auth**: OpenID Connect (go-oidc)
- **Dev Auth**: [mock-oauth2-server](https://github.com/navikt/mock-oauth2-server)

## License

MIT
