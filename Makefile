.PHONY: help build test test-coverage test-domain test-unit test-integration test-e2e load-test load-test-quick load-test-auth load-test-browse load-test-upload load-test-social coverage-domain lint generate migrate-up migrate-down migrate-status run run-worker validate-openapi docker-up docker-down clean install-hooks pre-commit

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
	@echo "  test-e2e          - Run Newman/Postman E2E tests"
	@echo "  load-test         - Run all k6 load tests"
	@echo "  load-test-quick   - Run quick smoke load test (1 minute)"
	@echo "  load-test-auth    - Run authentication flow load test"
	@echo "  load-test-browse  - Run browsing flow load test"
	@echo "  load-test-upload  - Run upload flow load test"
	@echo "  load-test-social  - Run social interactions load test"
	@echo "  coverage-domain   - Generate HTML coverage report for domain layer"
	@echo "  lint              - Run golangci-lint"
	@echo "  generate          - Run code generation (oapi-codegen)"
	@echo "  migrate-up        - Apply pending database migrations"
	@echo "  migrate-down      - Rollback last database migration"
	@echo "  migrate-status    - Show migration status"
	@echo "  migrate-create    - Create new migration (requires NAME=migration_name)"
	@echo "  run               - Start API server locally"
	@echo "  run-worker        - Start background worker"
	@echo "  validate-openapi  - Validate OpenAPI specification"
	@echo "  docker-up         - Start Docker Compose services"
	@echo "  docker-down       - Stop Docker Compose services"
	@echo "  clean             - Remove build artifacts"
	@echo "  install-hooks     - Install git pre-commit hooks (REQUIRED for Claude agents)"
	@echo "  pre-commit        - Run pre-commit checks manually"

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
# Requires Docker to be running for testcontainers
test-integration:
	@echo "Running integration tests with testcontainers..."
	@echo "Note: Docker must be running for testcontainers"
	@go test -race -tags=integration -v -timeout=10m ./tests/integration/...

# E2E tests (Newman/Postman)
test-e2e:
	@echo "Running Newman E2E tests..."
	@if [ ! -f tests/e2e/postman/goimg-api.postman_collection.json ]; then \
		echo "Postman collection not found: tests/e2e/postman/goimg-api.postman_collection.json"; \
		exit 1; \
	fi
	@if ! command -v newman &> /dev/null; then \
		echo "Newman not installed. Install with: npm install -g newman newman-reporter-htmlextra"; \
		exit 1; \
	fi
	@newman run tests/e2e/postman/goimg-api.postman_collection.json \
		--environment tests/e2e/postman/ci.postman_environment.json \
		--reporters cli,htmlextra \
		--reporter-htmlextra-export newman-report.html
	@echo "E2E test report: newman-report.html"

# Load tests (k6)
load-test:
	@echo "Running all k6 load tests..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo "\n=== Running auth flow load test ===" && k6 run tests/load/auth-flow.js
	@echo "\n=== Running browse flow load test ===" && k6 run tests/load/browse-flow.js
	@echo "\n=== Running social flow load test ===" && k6 run tests/load/social-flow.js
	@echo "\n=== Running upload flow load test ===" && k6 run tests/load/upload-flow.js
	@echo "\n=== Running mixed traffic load test ===" && k6 run tests/load/mixed-traffic.js
	@echo "\nAll load tests completed"

load-test-quick:
	@echo "Running quick smoke load test (1 minute)..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@k6 run --duration 1m --vus 10 tests/load/browse-flow.js
	@echo "Quick load test completed"

load-test-auth:
	@echo "Running authentication flow load test..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@k6 run tests/load/auth-flow.js

load-test-browse:
	@echo "Running browsing flow load test..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@k6 run tests/load/browse-flow.js

load-test-upload:
	@echo "Running upload flow load test..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@k6 run tests/load/upload-flow.js

load-test-social:
	@echo "Running social interactions load test..."
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@k6 run tests/load/social-flow.js

# Linting
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Code generation from OpenAPI spec
generate:
	@echo "Generating server code from OpenAPI spec..."
	@mkdir -p internal/interfaces/http/generated
	@PATH="$(PATH):/root/go/bin" oapi-codegen -config api/openapi/oapi-codegen.yaml api/openapi/openapi.yaml
	@gofmt -w internal/interfaces/http/generated/
	@echo "Code generation complete: internal/interfaces/http/generated/server.gen.go"

# Database migrations with Goose
migrate-up:
	@echo "Running pending migrations..."
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-goimg} sslmode=$${DB_SSL_MODE:-disable}" goose -dir migrations up

migrate-down:
	@echo "Rolling back last migration..."
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-goimg} sslmode=$${DB_SSL_MODE:-disable}" goose -dir migrations down

migrate-status:
	@echo "Checking migration status..."
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="host=$${DB_HOST:-localhost} port=$${DB_PORT:-5432} user=$${DB_USER:-postgres} password=$${DB_PASSWORD:-postgres} dbname=$${DB_NAME:-goimg} sslmode=$${DB_SSL_MODE:-disable}" goose -dir migrations status

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	@goose -dir migrations create $(NAME) sql

# Run targets
run:
	@echo "Starting API server..."
	@go run ./cmd/api

run-worker:
	@echo "Starting background worker..."
	@go run ./cmd/worker

# OpenAPI validation
validate-openapi:
	@echo "Validating OpenAPI specification..."
	@GOTOOLCHAIN=local go run tools/validate-openapi/main.go api/openapi/openapi.yaml
	@echo "OpenAPI spec validation passed"

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

# Git hooks
install-hooks:
	@echo "Installing git hooks..."
	@./scripts/setup-hooks.sh

# Pre-commit check (run manually before pushing)
# Runs FULL lint on all files - not just changed files
pre-commit:
	@echo "Running pre-commit checks (full lint)..."
	@go fmt ./...
	@go vet ./...
	@golangci-lint run ./...
	@echo "Pre-commit checks passed!"
