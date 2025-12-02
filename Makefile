.PHONY: help build test test-coverage lint generate migrate-up migrate-down migrate-status run run-worker validate-openapi docker-up docker-down clean

# Default target
help:
	@echo "goimg-datalayer - Image Gallery Backend"
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Compile API and worker binaries"
	@echo "  test            - Run all tests with race detector"
	@echo "  test-coverage   - Generate HTML coverage report"
	@echo "  lint            - Run golangci-lint"
	@echo "  generate        - Run code generation (oapi-codegen)"
	@echo "  migrate-up      - Apply database migrations"
	@echo "  migrate-down    - Rollback database migrations"
	@echo "  migrate-status  - Show migration status"
	@echo "  run             - Start API server locally"
	@echo "  run-worker      - Start background worker"
	@echo "  validate-openapi - Validate OpenAPI specification"
	@echo "  docker-up       - Start Docker Compose services"
	@echo "  docker-down     - Stop Docker Compose services"
	@echo "  clean           - Remove build artifacts"

# Build targets
build:
	@echo "Building binaries..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api
	@go build -o bin/worker ./cmd/worker
	@go build -o bin/migrate ./cmd/migrate
	@echo "Build complete: bin/api, bin/worker, bin/migrate"

# Test targets
test:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

test-coverage:
	@echo "Generating coverage report..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Linting
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Code generation (placeholder for Sprint 2)
generate:
	@echo "Code generation not yet configured (Sprint 2: OpenAPI)"

# Database migrations (placeholder - will need Goose in Sprint 3)
migrate-up:
	@echo "Migration up not yet configured (Sprint 3)"

migrate-down:
	@echo "Migration down not yet configured (Sprint 3)"

migrate-status:
	@echo "Migration status not yet configured (Sprint 3)"

# Run targets
run:
	@echo "Starting API server..."
	@go run ./cmd/api

run-worker:
	@echo "Starting background worker..."
	@go run ./cmd/worker

# OpenAPI validation (placeholder for Sprint 2)
validate-openapi:
	@echo "OpenAPI validation not yet configured (Sprint 2)"

# Docker Compose
docker-up:
	@echo "Starting Docker Compose services..."
	@docker-compose -f docker/docker-compose.yml up -d

docker-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose -f docker/docker-compose.yml down

# Clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ coverage.out coverage.html
	@echo "Clean complete"
