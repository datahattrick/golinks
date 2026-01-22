.PHONY: run build test clean dev db-up db-down db-logs docker-up docker-down services-up services-down

# Application
APP_NAME := golinks
MAIN_PATH := ./cmd/server

# Database (matches docker-compose)
DATABASE_URL ?= postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

# OIDC (matches docker-compose mock server)
OIDC_ISSUER ?= http://localhost:8080/golinks
OIDC_CLIENT_ID ?= golinks-app
OIDC_CLIENT_SECRET ?= secret
OIDC_REDIRECT_URL ?= http://localhost:3000/auth/callback

# Default target
all: build

# Run the application
run:
	go run $(MAIN_PATH)

# Build the binary
build:
	go build -o $(APP_NAME) $(MAIN_PATH)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(APP_NAME) coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

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
	go run $(MAIN_PATH)

# Development without OIDC
dev-no-auth: db-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 2
	DATABASE_URL="$(DATABASE_URL)" go run $(MAIN_PATH)

# Connect to PostgreSQL
db-shell:
	docker compose exec postgres psql -U golinks -d golinks

# View OIDC server logs
oidc-logs:
	docker compose logs -f oidc
