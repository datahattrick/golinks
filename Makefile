.PHONY: run build test clean dev db-up db-down db-logs docker-up docker-down services-up services-down \
	test-unit test-integration test-all test-cover test-verbose test-db-setup test-db-teardown \
	css css-watch

# Auto-load .env file if it exists (vars can still be overridden via command line)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Application
APP_NAME := golinks
MAIN_PATH := ./cmd/server

# Database (matches docker-compose)
DATABASE_URL ?= postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable
TEST_DATABASE_URL ?= postgres://golinks:golinks@localhost:5432/golinks_test?sslmode=disable

# Tailwind CSS (standalone CLI v3)
TAILWIND_VERSION := v3.4.17
TAILWIND_OS := $(shell uname -s | tr A-Z a-z)
TAILWIND_ARCH := $(shell uname -m | sed 's/x86_64/x64/' | sed 's/aarch64/arm64/')
TAILWIND_BIN := bin/tailwindcss

# OIDC (matches docker-compose mock server)
OIDC_ISSUER ?= http://localhost:8080/golinks
OIDC_CLIENT_ID ?= golinks-app
OIDC_CLIENT_SECRET ?= secret
OIDC_REDIRECT_URL ?= http://localhost:3000/auth/callback

# OIDC group → role mapping (for dev)
OIDC_GROUPS_CLAIM ?= groups
OIDC_ADMIN_GROUPS ?= golinks-admin
OIDC_MODERATOR_GROUPS ?= golinks-moderator

# Default target
all: build

# Run the application
run:
	go run $(MAIN_PATH)

# Build the binary (rebuilds CSS first)
build: css
	go build -o $(APP_NAME) $(MAIN_PATH)

# ============================================================================
# Tailwind CSS
# ============================================================================

# Download the Tailwind standalone CLI if not present
$(TAILWIND_BIN):
	@mkdir -p bin
	@echo "Downloading Tailwind CSS CLI $(TAILWIND_VERSION) ($(TAILWIND_OS)-$(TAILWIND_ARCH))..."
	curl -sLo $(TAILWIND_BIN) https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)
	chmod +x $(TAILWIND_BIN)

# Build minified production CSS
css: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i static/css/tailwind-input.css -o static/css/tailwind.css --minify

# Watch mode for development (rebuilds on template changes)
css-watch: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i static/css/tailwind-input.css -o static/css/tailwind.css --watch

# ============================================================================
# Testing Targets
# ============================================================================

# Run all unit tests (no database required)
test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/models/... ./internal/middleware/... ./internal/handlers/...

# Run integration tests (requires test database)
test-integration: test-db-setup
	@echo "Running integration tests..."
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v ./internal/db/...

# Run all tests (unit + integration)
test-all: test-db-setup
	@echo "Running all tests..."
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v ./...

# Run tests (alias for test-unit, no DB required)
test: test-unit

# Run tests with verbose output
test-verbose:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -count=1 ./...

# Run tests with coverage report
test-cover: test-db-setup
	@echo "Running tests with coverage..."
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with coverage and show summary
test-cover-summary: test-db-setup
	@echo "Running tests with coverage..."
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run tests for a specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./internal/db"; exit 1; fi
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v $(PKG)

# Setup test database
test-db-setup: db-up
	@echo "Setting up test database..."
	@sleep 2
	@docker compose exec -T postgres psql -U golinks -c "CREATE DATABASE golinks_test;" 2>/dev/null || true

# Teardown test database
test-db-teardown:
	@echo "Tearing down test database..."
	@docker compose exec -T postgres psql -U golinks -c "DROP DATABASE IF EXISTS golinks_test;"

# Run tests in CI mode (assumes database is already running)
test-ci:
	@echo "Running CI tests..."
	RUN_INTEGRATION_TESTS=1 TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -race -coverprofile=coverage.out ./...

# ============================================================================
# Test Feature Categories
# ============================================================================

# Test models (roles, permissions)
test-models:
	@echo "Testing models..."
	go test -v ./internal/models/...

# Test database operations
test-db: test-db-setup
	@echo "Testing database operations..."
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v ./internal/db/...

# Test handlers (moderation, links)
test-handlers:
	@echo "Testing handlers..."
	go test -v ./internal/handlers/...

# Test middleware (auth, PKI extraction)
test-middleware:
	@echo "Testing middleware..."
	go test -v ./internal/middleware/...

# ============================================================================
# Development Targets
# ============================================================================

# Clean build artifacts
clean:
	rm -f $(APP_NAME) coverage.out coverage.html
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy

# Lint code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test-unit

# ============================================================================
# Docker/Database Targets
# ============================================================================

# Start PostgreSQL only
db-up:
	docker compose up -d postgres

# Stop PostgreSQL
db-down:
	docker compose down

# View PostgreSQL logs
db-logs:
	docker compose logs -f postgres

# Start all services (PostgreSQL + OIDC)
services-up:
	docker compose up -d postgres oidc

# Stop all services
services-down:
	docker compose down

# Start everything in containers (PostgreSQL + OIDC + app)
docker-up:
	docker compose --profile full up -d

# Stop everything and remove volumes
docker-down:
	docker compose --profile full down -v

# Development: start services and run app locally with OIDC
dev: services-up
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Starting app with OIDC enabled..."
	@echo "  OIDC Issuer: $(OIDC_ISSUER)"
	@echo "  Login at: http://localhost:3000"
	DATABASE_URL="$(DATABASE_URL)" \
	OIDC_ISSUER="$(OIDC_ISSUER)" \
	OIDC_CLIENT_ID="$(OIDC_CLIENT_ID)" \
	OIDC_CLIENT_SECRET="$(OIDC_CLIENT_SECRET)" \
	OIDC_REDIRECT_URL="$(OIDC_REDIRECT_URL)" \
	OIDC_GROUPS_CLAIM="$(OIDC_GROUPS_CLAIM)" \
	OIDC_ADMIN_GROUPS="$(OIDC_ADMIN_GROUPS)" \
	OIDC_MODERATOR_GROUPS="$(OIDC_MODERATOR_GROUPS)" \
	ENABLE_RANDOM_KEYWORDS="true" \
	go run $(MAIN_PATH)

# Development without OIDC
dev-no-auth: db-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 2
	DATABASE_URL="$(DATABASE_URL)" go run $(MAIN_PATH)

# Connect to PostgreSQL
db-shell:
	docker compose exec postgres psql -U golinks -d golinks

# Connect to test PostgreSQL
db-shell-test:
	docker compose exec postgres psql -U golinks -d golinks_test

# View OIDC server logs
oidc-logs:
	docker compose logs -f oidc

# ============================================================================
# Help
# ============================================================================

help:
	@echo "GoLinks Makefile"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          - Build CSS and application binary"
	@echo "  make run            - Run the application"
	@echo "  make dev            - Start services and run app with OIDC"
	@echo "  make css            - Build production Tailwind CSS"
	@echo "  make css-watch      - Watch and rebuild CSS on changes"
	@echo "  make clean          - Remove build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run unit tests (no DB required)"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests (requires DB)"
	@echo "  make test-all       - Run all tests"
	@echo "  make test-cover     - Run tests with coverage report"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make test-pkg PKG=./internal/db - Test specific package"
	@echo ""
	@echo "Test Categories:"
	@echo "  make test-models    - Test model logic (roles, permissions)"
	@echo "  make test-db        - Test database operations"
	@echo "  make test-handlers  - Test HTTP handlers"
	@echo "  make test-middleware - Test middleware (auth, PKI)"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run linter"
	@echo "  make check          - Run all checks"
	@echo ""
	@echo "Database:"
	@echo "  make db-up          - Start PostgreSQL"
	@echo "  make db-down        - Stop PostgreSQL"
	@echo "  make db-shell       - Connect to PostgreSQL"
	@echo "  make db-shell-test  - Connect to test database"
	@echo "  make test-db-setup  - Create test database"
