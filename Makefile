.PHONY: help build test test-coverage test-domain test-unit test-integration coverage-domain lint generate migrate-up migrate-down migrate-status run run-worker validate-openapi docker-up docker-down clean

# Default target
help:
	@echo "goimg-datalayer - Image Gallery Backend"
	@echo ""
	@echo "Available targets:"
	@echo "  build             - Compile API and worker binaries"
	@echo "  test              - Run all tests with race detector"
	@echo "  test-coverage     - Generate HTML coverage report (all layers)"
	@echo "  test-domain       - Run domain layer tests with 90% threshold"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  coverage-domain   - Generate HTML coverage report for domain layer"
	@echo "  lint              - Run golangci-lint"
	@echo "  generate          - Run code generation (oapi-codegen)"
	@echo "  migrate-up        - Apply database migrations"
	@echo "  migrate-down      - Rollback database migrations"
	@echo "  migrate-status    - Show migration status"
	@echo "  run               - Start API server locally"
	@echo "  run-worker        - Start background worker"
	@echo "  validate-openapi  - Validate OpenAPI specification"
	@echo "  docker-up         - Start Docker Compose services"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  clean             - Remove build artifacts"

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
	@echo "Running all tests with race detector..."
	@go test -race -v ./...

test-coverage:
	@echo "Generating coverage report for all layers..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | grep total

# Domain layer tests (90% coverage threshold)
test-domain:
	@echo "Running domain layer tests..."
	@if go list ./internal/domain/... 2>/dev/null | grep -q .; then \
		go test -race -coverprofile=domain-coverage.out -covermode=atomic ./internal/domain/...; \
		COVERAGE=$$(go tool cover -func=domain-coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
		echo "Domain layer coverage: $${COVERAGE}%"; \
		if [ -n "$$COVERAGE" ] && [ $$(echo "$$COVERAGE < 90" | bc -l) -eq 1 ]; then \
			echo "ERROR: Domain coverage $${COVERAGE}% is below 90% threshold"; \
			exit 1; \
		fi; \
		echo "SUCCESS: Domain coverage meets 90% threshold"; \
	else \
		echo "No domain packages found yet (expected during Sprint 1 Week 3-4)"; \
	fi

coverage-domain:
	@echo "Generating domain layer coverage report..."
	@if go list ./internal/domain/... 2>/dev/null | grep -q .; then \
		go test -race -coverprofile=domain-coverage.out -covermode=atomic ./internal/domain/...; \
		go tool cover -html=domain-coverage.out -o domain-coverage.html; \
		echo "Domain coverage report: domain-coverage.html"; \
		go tool cover -func=domain-coverage.out | grep total; \
	else \
		echo "No domain packages found yet (expected during Sprint 1 Week 3-4)"; \
	fi

# Unit tests (fast tests without external dependencies)
test-unit:
	@echo "Running unit tests..."
	@go test -race -short -v ./...

# Integration tests (tests with database, Redis, etc.)
test-integration:
	@echo "Running integration tests..."
	@go test -race -tags=integration -v ./...

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
	@rm -rf bin/ coverage.out coverage.html domain-coverage.out domain-coverage.html
	@echo "Clean complete"
