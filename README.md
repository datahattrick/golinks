# GoLinks

A self-hosted URL shortener for teams. Create memorable short links like `go/docs` that redirect to long URLs.

## Features

- **Simple Interface** - Clean, centered search with instant results via HTMX
- **Dark Mode** - Automatic detection with manual toggle
- **OIDC Authentication** - Optional SSO integration (Google, Okta, etc.)
- **Click Tracking** - Track how often each link is used
- **Fast Search** - Trigram-based fuzzy search across keywords, URLs, and descriptions

## Quick Start

```bash
# Start PostgreSQL
make db-up

# Run the application
make dev
```

Visit http://localhost:3000

## Requirements

- Go 1.22+
- PostgreSQL 14+
- Docker (optional, for running PostgreSQL)

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Server
SERVER_ADDR=:3000

# Database (matches docker-compose defaults)
DATABASE_URL=postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

# OIDC (optional - leave OIDC_ISSUER empty to disable auth)
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=http://localhost:3000/auth/callback
```

## Usage

### Redirects

- `http://localhost:3000/go/docs` → Redirects to the URL for "docs"
- `http://localhost:3000/docs` → Same redirect (shorter form)

### Creating Links

1. Log in via SSO (if configured)
2. Use the form at the top of the page
3. Enter a keyword, URL, and optional description

### Searching

Type in the search box to filter links by keyword, URL, or description. Results update instantly.

## Development

```bash
make dev          # Start db + run app
make build        # Build binary
make test         # Run tests
make db-shell     # Connect to PostgreSQL
```

## Docker

```bash
make db-up                            # PostgreSQL only
docker compose --profile full up -d   # Full stack
make docker-down                      # Stop and clean up
```

## Tech Stack

- **Backend**: Go, Fiber v3, pgx v5
- **Frontend**: HTMX, Tailwind CSS
- **Database**: PostgreSQL with trigram search
- **Auth**: OpenID Connect (go-oidc)

## License

MIT
