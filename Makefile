.PHONY: run build test clean dev db-up db-down db-logs docker-up docker-down

# Application
APP_NAME := golinks
MAIN_PATH := ./cmd/server

# Database (matches docker-compose)
DATABASE_URL ?= postgres://golinks:golinks@localhost:5432/golinks?sslmode=disable

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

# Start everything (PostgreSQL + app)
docker-up:
	docker compose up -d

# Stop everything
docker-down:
	docker compose down -v

# Development: start db and run app locally
dev: db-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 2
	DATABASE_URL="$(DATABASE_URL)" go run $(MAIN_PATH)

# Connect to PostgreSQL
db-shell:
	docker compose exec postgres psql -U golinks -d golinks
