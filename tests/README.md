# Testing Guide

> Practical guide for running, writing, and debugging tests in goimg-datalayer.
> For comprehensive test strategy and patterns, see [`claude/test_strategy.md`](/home/user/goimg-datalayer/claude/test_strategy.md)

## Quick Start

```bash
# Run all tests (unit + integration)
make test

# Run only unit tests (fast, no external dependencies)
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Run E2E tests (requires running server)
make test-e2e

# Generate HTML coverage report
make test-coverage

# Run tests with race detection (always use before committing)
go test -race ./...

# Run tests for specific package
go test -v ./internal/domain/identity/...

# Run specific test function
go test -v -run TestNewEmail ./internal/domain/identity/...
```

## Directory Structure

```
tests/
├── integration/              # Integration tests with testcontainers
│   ├── repository/           # Repository tests (PostgreSQL)
│   │   ├── user_repository_test.go
│   │   └── image_repository_test.go
│   ├── storage/              # Storage provider tests (S3, local)
│   │   └── s3_storage_test.go
│   ├── security/             # Security integration tests (JWT, ClamAV)
│   │   └── clamav_scanner_test.go
│   └── testhelpers/          # Test infrastructure setup
│       ├── setup.go          # Testcontainer initialization
│       └── fixtures.go       # Test data builders
├── e2e/                      # End-to-end API tests
│   ├── postman/              # Postman collections
│   │   ├── goimg-collection.json
│   │   └── environment/
│   │       ├── local.json
│   │       └── ci.json
│   └── newman/               # Newman test runner
│       └── run_tests.sh
├── contract/                 # OpenAPI contract tests
│   └── openapi_test.go
├── security/                 # Security test suite (OWASP)
│   └── owasp_test.go
├── load/                     # Load testing with k6
│   ├── image_upload.js
│   └── api_baseline.js
└── fixtures/                 # Test data files
    ├── images/               # Sample images for upload tests
    │   ├── valid_jpeg.jpg
    │   ├── valid_png.png
    │   ├── oversized.jpg
    │   └── malware_sample.bin
    └── data/                 # SQL seed data
        ├── seed_users.sql
        └── seed_images.sql
```

## Test Types

### Unit Tests

**Location**: Next to the code being tested (e.g., `internal/domain/identity/email_test.go`)

**Characteristics**:
- No external dependencies (no DB, no HTTP, no filesystem)
- Use mocks for dependencies
- Fast execution (< 1 second total)
- Run with `-short` flag

**Example**:
```go
func TestNewEmail(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        input   string
        wantErr error
    }{
        {"valid", "user@example.com", nil},
        {"empty", "", ErrEmailEmpty},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            _, err := NewEmail(tt.input)
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

**Location**: `tests/integration/`

**Characteristics**:
- Use real external dependencies via testcontainers
- Test interactions between components
- Slower execution (seconds per test)
- Skip with `-short` flag

**Setup Required**:
```bash
# Docker must be running
docker ps

# Tests will automatically start containers
go test ./tests/integration/...
```

**Example**:
```go
func TestUserRepository_Save(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    suite := testhelpers.SetupTestSuite(t)
    repo := postgres.NewUserRepository(suite.DB)

    user := newTestUser(t)
    err := repo.Save(context.Background(), user)

    require.NoError(t, err)
}
```

### E2E Tests

**Location**: `tests/e2e/postman/`

**Characteristics**:
- Test complete user workflows
- Validate API contracts
- Run against running server

**Running E2E Tests**:
```bash
# 1. Start dependencies
docker-compose -f docker/docker-compose.yml up -d

# 2. Run migrations
make migrate-up

# 3. Start API server
make run &

# 4. Wait for server to be ready
curl -f http://localhost:8080/health || sleep 5

# 5. Run Newman tests
newman run tests/e2e/postman/goimg-collection.json \
  -e tests/e2e/postman/environment/local.json \
  --reporters cli,json \
  --reporter-json-export newman-results.json
```

### Contract Tests

**Location**: `tests/contract/`

**Characteristics**:
- Validate implementation matches OpenAPI spec
- Ensure API contract compliance

**Running Contract Tests**:
```bash
go test -v ./tests/contract/...

# Also validate spec itself
make validate-openapi
```

## Testcontainers Setup

### Prerequisites

- Docker installed and running
- Sufficient resources (at least 4GB RAM for Docker)

### Configuration

Testcontainers automatically:
1. Pulls required Docker images (postgres:16-alpine, redis:7-alpine)
2. Starts containers before tests
3. Runs database migrations
4. Cleans up containers after tests

### Debugging Testcontainers

```bash
# Enable testcontainers debug logs
export TESTCONTAINERS_RYUK_DISABLED=true
export TESTCONTAINERS_DEBUG=true

# Run integration tests with verbose output
go test -v ./tests/integration/...

# View running containers
docker ps
```

### Performance Optimization

Testcontainers can be slow on first run. Speed improvements:

```go
// Use container reuse (experimental)
func SetupTestSuite(t *testing.T) *TestSuite {
    req := testcontainers.ContainerRequest{
        Image: "postgres:16-alpine",
        Reuse: true,  // Reuse container across test runs
    }
    // ...
}
```

## Writing Tests

### Table-Driven Test Pattern

```go
func TestFunction(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr error
    }{
        {
            name:  "valid input",
            input: validInput,
            want:  expectedOutput,
        },
        {
            name:    "invalid input",
            input:   invalidInput,
            wantErr: ErrInvalidInput,
        },
    }

    for _, tt := range tests {
        tt := tt  // Capture range variable for parallel tests
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            got, err := Function(tt.input)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Using Test Helpers

```go
// Mark function as test helper
func newTestUser(t *testing.T) *identity.User {
    t.Helper()  // Stack traces will skip this function

    email, _ := identity.NewEmail("test@example.com")
    user, err := identity.NewUser(email, "testuser", "SecurePass123!")
    require.NoError(t, err)

    return user
}

// Use builder pattern for complex objects
func TestWithUserBuilder(t *testing.T) {
    user := testhelpers.NewUserBuilder().
        WithEmail("admin@example.com").
        WithRole(identity.RoleAdmin).
        Build(t)

    assert.Equal(t, identity.RoleAdmin, user.Role())
}
```

### Mocking Dependencies

```go
// Use testify/mock for interface mocking
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

// In test:
func TestRegisterUser(t *testing.T) {
    mockRepo := new(MockUserRepository)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).Return(nil)

    handler := commands.NewRegisterUserHandler(mockRepo)
    err := handler.Handle(context.Background(), cmd)

    require.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### HTTP Handler Testing

```go
func TestAuthHandler_Register(t *testing.T) {
    mockService := new(MockAuthService)
    handler := handlers.NewAuthHandler(mockService)

    requestBody := `{"email":"test@example.com","password":"SecurePass123!"}`
    req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(requestBody))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.Register(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)

    var response map[string]interface{}
    json.NewDecoder(rec.Body).Decode(&response)
    assert.Contains(t, response, "id")
}
```

## Coverage Reports

### Generating Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage Thresholds

| Layer | Minimum Coverage |
|-------|------------------|
| Domain | 90% |
| Application | 85% |
| Infrastructure | 70% |
| Handlers | 75% |
| **Overall** | **80%** |

### CI Enforcement

Coverage is automatically checked in CI:

```yaml
# .github/workflows/test.yml
- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 80.0" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80% threshold"
      exit 1
    fi
```

## Debugging Tests

### Debugging Flaky Tests

```bash
# Run test multiple times to reproduce flakiness
go test -count=100 -run TestFlakyTest ./...

# Run with race detector
go test -race -run TestConcurrentAccess ./...

# Increase verbosity
go test -v -run TestDebugMe ./...
```

### Common Issues

**Issue: Integration tests fail with "connection refused"**
```bash
# Check Docker is running
docker ps

# Check containers are healthy
docker-compose -f docker/docker-compose.yml ps

# View container logs
docker logs goimg-postgres
docker logs goimg-redis
```

**Issue: Tests timeout**
```bash
# Increase timeout for slow tests
go test -timeout 10m ./...

# Check for deadlocks with race detector
go test -race -timeout 30s ./...
```

**Issue: Testcontainers cleanup fails**
```bash
# Manually clean up containers
docker ps -a | grep testcontainers | awk '{print $1}' | xargs docker rm -f

# Clean up dangling volumes
docker volume prune -f
```

**Issue: Coverage lower than expected**
```bash
# Find uncovered code
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check specific package
go test -cover ./internal/domain/identity
```

## Best Practices

### DO

- ✅ Use `t.Parallel()` for independent tests
- ✅ Use `t.Helper()` for test utility functions
- ✅ Use table-driven tests for multiple scenarios
- ✅ Use `require` for assertions that should stop test execution
- ✅ Use `assert` for assertions that should continue
- ✅ Clean up resources with `t.Cleanup()`
- ✅ Skip integration tests with `-short` flag
- ✅ Use meaningful test names describing behavior
- ✅ Test both success and error paths
- ✅ Test edge cases and boundary conditions

### DON'T

- ❌ Don't use `t.Parallel()` when tests share state
- ❌ Don't use `time.Sleep()` for synchronization (use channels/contexts)
- ❌ Don't test implementation details (test behavior)
- ❌ Don't mock what you don't own (use real dependencies or adapters)
- ❌ Don't write tests that pass on invalid code (mutation test mindset)
- ❌ Don't skip cleanup (use `t.Cleanup()` or `defer`)
- ❌ Don't hardcode ports or paths (use dynamic allocation)
- ❌ Don't test private functions directly (test through public API)
- ❌ Don't ignore errors in test code
- ❌ Don't commit commented-out tests

## Test Data Management

### Fixtures

```go
// Load test image from fixtures
func loadTestImage(t *testing.T, filename string) []byte {
    t.Helper()

    path := filepath.Join("tests", "fixtures", "images", filename)
    data, err := os.ReadFile(path)
    require.NoError(t, err, "failed to load fixture: %s", filename)

    return data
}

// Usage:
imageData := loadTestImage(t, "valid_jpeg.jpg")
```

### Test Database Seeding

```sql
-- tests/fixtures/data/seed_users.sql
INSERT INTO users (id, email, username, password_hash, role, status, created_at, updated_at)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@example.com', 'admin', '$argon2id$v=19$...', 'admin', 'active', NOW(), NOW()),
    ('00000000-0000-0000-0000-000000000002', 'user@example.com', 'testuser', '$argon2id$v=19$...', 'user', 'active', NOW(), NOW());
```

```go
// Load seed data in test
func seedDatabase(t *testing.T, db *sqlx.DB) {
    t.Helper()

    seed, err := os.ReadFile("tests/fixtures/data/seed_users.sql")
    require.NoError(t, err)

    _, err = db.Exec(string(seed))
    require.NoError(t, err)
}
```

## Performance Testing

### Benchmarking

```go
func BenchmarkPasswordHashing(b *testing.B) {
    password := "SecurePassword123!"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        HashPassword(password)
    }
}

// Run benchmarks
go test -bench=. -benchmem ./...
```

### Load Testing with k6

```bash
# Install k6
brew install k6  # macOS
# or
curl -L https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz | tar xvz

# Run load test
k6 run tests/load/image_upload.js

# Run with custom VUs and duration
k6 run --vus 50 --duration 5m tests/load/api_baseline.js
```

## Continuous Integration

### Running Tests in CI

Tests run automatically on:
- Every push to `main` or `develop`
- Every pull request
- Scheduled daily security scans

### Required Checks

All PRs must pass:
- ✅ Linting (`golangci-lint`)
- ✅ Unit tests
- ✅ Integration tests
- ✅ E2E tests
- ✅ Contract validation
- ✅ Coverage threshold (80%)
- ✅ Security scan (`gosec`)

### Viewing Test Results

```bash
# CI test results are available in:
# - GitHub Actions tab (https://github.com/your-org/goimg-datalayer/actions)
# - Coverage reports (Codecov)
# - Security scan results (GitHub Security tab)
```

## Troubleshooting

### Test Hangs Indefinitely

```bash
# Add timeout to identify hanging test
go test -timeout 30s ./...

# Check for goroutine leaks
go test -v -run TestSuspect 2>&1 | grep "goroutine"
```

### Out of Memory

```bash
# Increase Docker memory limit
# Docker Desktop -> Settings -> Resources -> Memory (4GB minimum)

# Run tests sequentially instead of parallel
go test -p 1 ./...
```

### Port Already in Use

```bash
# Testcontainers uses random ports by default
# If using fixed ports, check what's using them:
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis

# Kill process using port
kill -9 $(lsof -t -i:5432)
```

## Resources

- [Test Strategy Documentation](/home/user/goimg-datalayer/claude/test_strategy.md) - Comprehensive testing strategy
- [Testing & CI Guide](/home/user/goimg-datalayer/claude/testing_ci.md) - CI/CD integration details
- [testify Documentation](https://github.com/stretchr/testify) - Assertion library
- [testcontainers-go Documentation](https://golang.testcontainers.org/) - Container testing
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test) - Official Go docs

## Getting Help

- Check existing tests for patterns
- Review test strategy document for comprehensive examples
- Ask in team chat for test-specific questions
- Tag `@backend-test-architect` for test design reviews
