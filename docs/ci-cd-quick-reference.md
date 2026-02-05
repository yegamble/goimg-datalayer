# CI/CD Quick Reference Card
**Last Updated:** 2026-02-05

---

## Pre-Push Checklist (MANDATORY)

```bash
# 1. Run pre-commit checks (REQUIRED for Claude agents)
make pre-commit

# 2. Run tests locally
make test

# 3. Validate OpenAPI (if API changes)
make validate-openapi

# 4. Check test coverage
make test-coverage
```

**Never push without running `make pre-commit`!**

---

## GitHub Actions Workflows

### Main CI Pipeline (`.github/workflows/ci.yml`)

| Job | Timeout | Purpose | Local Command |
|-----|---------|---------|---------------|
| **lint** | 10m | Code quality checks | `make pre-commit` |
| **test-unit** | 15m | Unit tests + race detection | `make test-unit` |
| **domain-tests** | 10m | Domain layer (90% threshold) | `make test-domain` |
| **test-integration** | 20m | Integration tests (DB, Redis) | `make test-integration` |
| **openapi-validation** | 10m | Validate API spec | `make validate-openapi` |
| **build** | 15m | Build binaries (Ubuntu, macOS) | `make build` |
| **docker-build** | 20m | Build Docker images | N/A |
| **coverage** | 10m | Merge coverage (80% threshold) | `make test-coverage` |
| **e2e-tests** | 15m | Newman/Postman E2E tests | `make test-e2e` |
| **ci-success** | 1m | Final status gate | N/A |

### Security Pipeline (`.github/workflows/security.yml`)

| Job | Purpose |
|-----|---------|
| **dependency-audit** | Check for vulnerable dependencies |
| **sbom** | Generate Software Bill of Materials |
| **gosec** | Go security scanner |
| **trivy** | Container vulnerability scan |
| **govulncheck** | Go vulnerability database check |
| **secret-scan** | Detect committed secrets |

---

## Coverage Thresholds

| Layer | Threshold | Enforcement |
|-------|-----------|-------------|
| **Domain** | 90% | Always (blocks CI) |
| **Overall** | 80% | main/develop only |
| **Feature Branches** | 80% | Warning only |

---

## act Tool Commands

### Prerequisites
```bash
# Install act
brew install act

# Start Docker
open -a Docker

# Verify Docker is running
docker ps
```

### Basic Commands
```bash
# List all jobs
act -l --container-architecture linux/amd64

# Test lint job
act -j lint --container-architecture linux/amd64 push

# Test unit tests
act -j test-unit --container-architecture linux/amd64 push

# Dry run (see what would execute)
act -j lint --container-architecture linux/amd64 --dryrun
```

### Limitations
- ❌ Service containers (PostgreSQL, Redis) not fully supported
- ❌ GitHub-specific features (step summaries, PR comments)
- ❌ Artifact upload/download between jobs
- ✅ Basic workflow execution and testing

**Recommendation:** Use `make` targets for comprehensive testing.

---

## Service Container Requirements

### Jobs That Need PostgreSQL + Redis
- `test-integration`
- `e2e-tests`

### Local Setup
```bash
# Start services
docker-compose -f docker/docker-compose.yml up -d

# Run migrations
make migrate-up

# Verify services
docker ps

# Stop services
docker-compose -f docker/docker-compose.yml down
```

---

## Common Issues

### Issue: Pre-commit checks fail
```bash
# Run individual checks to identify issue
go fmt ./...
go vet ./...
golangci-lint run

# Fix issues, then retry
make pre-commit
```

### Issue: Tests fail locally but pass in CI
```bash
# Ensure Go version matches CI
go version  # Should be 1.25.x

# Check for uncommitted go.mod changes
go mod tidy
git diff go.mod go.sum

# Verify services are running
docker ps
```

### Issue: Coverage below threshold
```bash
# Generate coverage report
make test-coverage
open coverage.html

# Focus on domain layer (90% required)
make coverage-domain
open domain-coverage.html
```

### Issue: act fails to run
```bash
# Verify Docker is running
docker ps

# Update act to latest
brew upgrade act

# Use correct architecture flag
act -j lint --container-architecture linux/amd64 push
```

---

## Workflow Triggers

### ci.yml
- **Push:** main, develop, claude/*
- **Pull Request:** All branches

### security.yml
- **Push:** main, develop
- **Pull Request:** All branches
- **Schedule:** Daily at midnight
- **Manual:** workflow_dispatch

### auto-merge.yml
- **Pull Request:** Only when PR is opened

---

## Job Dependencies (ci.yml)

```
Stage 0 (Parallel):
├─ lint
├─ test-unit
├─ domain-tests
├─ test-integration
└─ openapi-validation

Stage 1 (After Stage 0):
├─ build (needs: lint, test-unit, test-integration, domain-tests)
├─ docker-build (needs: lint, test-unit)
└─ coverage (needs: test-unit, test-integration, domain-tests)

Stage 2 (After Stage 1):
└─ e2e-tests (needs: build)

Stage 3 (After Stage 2):
└─ ci-success (needs: all jobs)
```

---

## Composite Actions

### `.github/actions/setup-go-env`
**Purpose:** Setup Go + install libvips

**Usage:**
```yaml
- uses: ./.github/actions/setup-go-env
  with:
    go-version: '1.25.5'
    install-libvips: 'true'
```

### `.github/actions/setup-database`
**Purpose:** Wait for PostgreSQL + run migrations

**Usage:**
```yaml
- uses: ./.github/actions/setup-database
  with:
    database-url: ${{ env.DATABASE_URL }}
    skip-migrations: 'false'
```

---

## Debugging Failed CI

### Step 1: Check Logs
1. Click on failed job in GitHub Actions
2. Expand failing step
3. Look for error messages

### Step 2: Reproduce Locally
```bash
# Run equivalent local command
make pre-commit  # For lint job
make test-unit   # For test-unit job
make test-integration  # For test-integration job
```

### Step 3: Common Fixes
```bash
# Format code
go fmt ./...

# Fix linting issues
golangci-lint run --fix

# Update dependencies
go mod tidy
go mod download
go mod verify

# Regenerate code
make generate

# Re-run migrations
make migrate-down
make migrate-up
```

---

## Environment Variables (CI)

### Workflow-Level
- `GO_VERSION: "1.25.5"`
- `GOLANGCI_LINT_VERSION: "v2.6.2"`

### Job-Level (Integration/E2E)
- `DATABASE_URL: postgresql://goimg_test:test_password@localhost:5432/goimg_test?sslmode=disable`
- `REDIS_URL: redis://localhost:6379/0`

---

## Validation Script

**Location:** `/Users/yosefgamble/github/goimg-datalayer/scripts/validate-ci-local.sh`

**Usage:**
```bash
# Run comprehensive validation
./scripts/validate-ci-local.sh

# This script will:
# 1. Check prerequisites (act, Docker, Go)
# 2. Validate workflow files
# 3. Check composite actions
# 4. Run pre-commit checks
# 5. Test critical jobs with act (if Docker running)
```

---

## Resources

- **Full Report:** `/Users/yosefgamble/github/goimg-datalayer/docs/ci-cd-validation-report.md`
- **act Documentation:** https://github.com/nektos/act
- **GitHub Actions Docs:** https://docs.github.com/en/actions
- **golangci-lint Config:** `/Users/yosefgamble/github/goimg-datalayer/.golangci.yml`

---

**Quick Start:**
```bash
make pre-commit && make test && git push
```
