# Development

## Project Structure

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
│   ├── handlers/            # HTTP handlers (HTMX UI)
│   │   ├── auth.go          # OIDC flow (login/callback/logout)
│   │   ├── links.go         # Link management + sparklines
│   │   ├── manage.go        # Moderator link management with org badges
│   │   ├── moderation.go    # Link approval workflow
│   │   ├── health.go        # URL health-check trigger
│   │   ├── user_links.go    # Personal link CRUD
│   │   ├── users.go         # User management (admin)
│   │   ├── profile.go       # User profile page
│   │   ├── branding.go      # Site-branding helpers
│   │   ├── handlers.go      # Shared handler utilities
│   │   ├── redirect.go      # Keyword → URL redirect
│   │   └── api/             # JSON API v1 handlers
│   │       ├── links.go     # Link CRUD (JSON)
│   │       ├── resolve.go   # Keyword resolution (JSON)
│   │       ├── users.go     # User management (JSON)
│   │       ├── moderation.go# Approve/reject (JSON)
│   │       ├── health.go    # Health check (JSON)
│   │       └── response.go  # JSON response helpers
│   ├── email/               # Email notifications
│   │   ├── email.go         # SMTP service
│   │   ├── templates.go     # Email templates
│   │   └── notifications.go # Notification handlers
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
│   ├── css/                 # Stylesheets (fonts.css, style.css)
│   ├── js/                  # Vendored JS (htmx.min.js, tailwind.min.js)
│   ├── fonts/               # Vendored WOFF2 fonts (Inter, JetBrains Mono)
│   └── favicon.svg          # Site favicon
├── chart/golinks/           # Helm chart for Kubernetes/OpenShift
├── Dockerfile               # Multi-stage Docker build
├── docker-compose.yml       # Local development stack
├── Makefile                 # Build and development commands
└── go.mod                   # Go module definition
```

## Key Technologies

| Component | Technology |
|-----------|------------|
| Backend | Go, Fiber v3, pgx v5 |
| Frontend | HTMX, Tailwind CSS (vendored) |
| Database | PostgreSQL with trigram search |
| Authentication | OpenID Connect (go-oidc v3) |
| Migrations | golang-migrate with embedded SQL |
| Dev Auth | [mock-oauth2-server](https://github.com/navikt/mock-oauth2-server) |

## Development Commands

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

# Logs
make db-logs          # PostgreSQL logs
make oidc-logs        # OIDC server logs
```

## Testing

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

## Code Quality

```bash
make fmt              # Format code with gofmt
make vet              # Run go vet
make lint             # Run golangci-lint (if installed)
make check            # Run all checks (fmt, vet, lint, test)
make tidy             # Clean up go.mod
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run checks: `make check`
5. Commit and push, then open a Pull Request

Follow standard Go conventions, use `gofmt`, and add tests for new functionality.

## Releases

Releases are created via git tags. CI/CD automatically builds and publishes container images to GitHub Container Registry and DockerHub.

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Use semantic versioning with a `v` prefix (`v<major>.<minor>.<patch>`). Published tags include exact version, minor, major, `latest`, and commit SHA.
