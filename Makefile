.PHONY: help check-go-version build test test-coverage test-domain test-unit test-integration test-e2e load-test load-test-quick load-test-auth load-test-browse load-test-upload load-test-social load-test-groups test-load-sprint10-login test-load-sprint10-hibp test-load-sprint10-failopen test-load-sprint10-all coverage-domain fmt lint generate migrate-up migrate-down migrate-status run run-worker validate-openapi docker-up docker-down clean install-hooks pre-commit

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
	@echo "  load-test                    - Run all k6 load tests"
	@echo "  load-test-quick              - Run quick smoke load test (1 minute)"
	@echo "  load-test-auth               - Run authentication flow load test"
	@echo "  load-test-browse             - Run browsing flow load test"
	@echo "  load-test-upload             - Run upload flow load test"
	@echo "  load-test-social             - Run social interactions load test"
	@echo "  load-test-groups             - Run Sprint 20 groups load test (< 200ms p95)"
	@echo "  test-load-sprint10-login     - Run Sprint 10 login timing load test (S10-PERF-001)"
	@echo "  test-load-sprint10-hibp      - Run Sprint 10 HIBP registration load test"
	@echo "  test-load-sprint10-failopen  - Run Sprint 10 HIBP fail-open load test"
	@echo "  test-load-sprint10-all       - Run all Sprint 10 load tests"
	@echo "  coverage-domain              - Generate HTML coverage report for domain layer"
	@echo "  fmt               - Format Go source code"
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

# Go version check - enforces minimum Go 1.25
GO_VERSION_MIN := 1.25
GO_VERSION_CURRENT := $(shell go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
GO_VERSION_OK := $(shell printf '%s\n%s' "$(GO_VERSION_MIN)" "$(GO_VERSION_CURRENT)" | sort -V | head -n1)

check-go-version:
	@if [ "$(GO_VERSION_OK)" != "$(GO_VERSION_MIN)" ]; then \
		echo "ERROR: Go $(GO_VERSION_MIN)+ required, but found Go $(GO_VERSION_CURRENT)"; \
		echo "Download the latest Go from: https://go.dev/dl/"; \
		exit 1; \
	fi
	@echo "Go version check passed: $(GO_VERSION_CURRENT) >= $(GO_VERSION_MIN)"

# Build targets
build: check-go-version
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

load-test-groups:
	@echo "=========================================="
	@echo "Sprint 20 Load Test: Groups/Communities"
	@echo "Performance Requirement: List groups < 200ms at p95"
	@echo "=========================================="
	@if ! command -v k6 &> /dev/null; then \
		echo "k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo "Starting groups flow load test..."
	@k6 run tests/load/groups-flow.js
	@echo ""
	@echo "=========================================="
	@echo "Sprint 20 Load Test Complete"
	@echo "Check output above for p95 threshold results"
	@echo "Target: list_groups p95 < 200ms"
	@echo "=========================================="

# Sprint 10 load tests (Security features)
test-load-sprint10-login:
	@echo "=========================================="
	@echo "Sprint 10 Load Test: Login Timing"
	@echo "Security Gate: S10-PERF-001"
	@echo "Requirement: p95 login latency < 500ms"
	@echo "=========================================="
	@if ! command -v k6 &> /dev/null; then \
		echo "ERROR: k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo "Starting test... (duration: ~13 minutes)"
	@k6 run tests/load/sprint10-login-timing.js
	@echo ""
	@echo "✓ Login timing load test complete"
	@echo "Review output above for Security Gate S10-PERF-001 status"

test-load-sprint10-hibp:
	@echo "=========================================="
	@echo "Sprint 10 Load Test: HIBP Registration"
	@echo "Security Gate: S10-HIBP-003"
	@echo "Requirement: Compromised password rejection"
	@echo "=========================================="
	@if ! command -v k6 &> /dev/null; then \
		echo "ERROR: k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo "Starting test... (duration: ~9 minutes)"
	@k6 run tests/load/sprint10-hibp-registration.js
	@echo ""
	@echo "✓ HIBP registration load test complete"
	@echo "Review output above for rejection/acceptance rates"

test-load-sprint10-failopen:
	@echo "=========================================="
	@echo "Sprint 10 Load Test: HIBP Fail-Open"
	@echo "Security Gate: S10-HIBP-003"
	@echo "Requirement: Fail-open behavior"
	@echo "=========================================="
	@if ! command -v k6 &> /dev/null; then \
		echo "ERROR: k6 not installed. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo ""
	@echo "IMPORTANT: This test requires HIBP to be disabled"
	@echo "Set environment variable: export HIBP_ENABLED=false"
	@echo "Or block network access to api.pwnedpasswords.com"
	@echo ""
	@read -p "Press Enter to continue (Ctrl+C to cancel)..."
	@echo ""
	@echo "Starting test... (duration: ~5 minutes)"
	@k6 run tests/load/sprint10-hibp-failopen.js
	@echo ""
	@echo "✓ HIBP fail-open load test complete"
	@echo "Verify server logs show HIBP warnings (not errors)"

test-load-sprint10-all:
	@echo "=========================================="
	@echo "Sprint 10 Load Tests - Full Suite"
	@echo "Security Gates: S10-PERF-001, S10-HIBP-003"
	@echo "=========================================="
	@echo ""
	@echo "Running all Sprint 10 load tests..."
	@echo "Total duration: ~30 minutes"
	@echo ""
	@$(MAKE) test-load-sprint10-login
	@echo ""
	@$(MAKE) test-load-sprint10-hibp
	@echo ""
	@echo "Skipping fail-open test (requires manual HIBP disable)"
	@echo "Run separately with: make test-load-sprint10-failopen"
	@echo ""
	@echo "=========================================="
	@echo "✓ All Sprint 10 load tests complete"
	@echo "=========================================="
	@echo ""
	@echo "Security Gate Status:"
	@echo "  S10-PERF-001: Check login timing results above"
	@echo "  S10-HIBP-003: Check HIBP rejection rates above"
	@echo ""
	@echo "See docs/testing/SPRINT10_LOAD_TESTING_GUIDE.md for details"

# Formatting
fmt:
	@echo "Formatting Go source code..."
	@go fmt ./...
	@echo "Formatting complete"

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
# Detect docker compose v2 (docker compose) vs v1 (docker-compose)
DOCKER_COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)

docker-up:
	@echo "Starting Docker Compose services using $(DOCKER_COMPOSE)..."
	@$(DOCKER_COMPOSE) -f docker/docker-compose.yml up -d

docker-down:
	@echo "Stopping Docker Compose services..."
	@$(DOCKER_COMPOSE) -f docker/docker-compose.yml down

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
pre-commit: check-go-version
	@echo "Running pre-commit checks (full lint)..."
	@go fmt ./...
	@go vet ./...
	@golangci-lint run ./...
	@echo "Pre-commit checks passed!"
