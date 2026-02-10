# Testing Strategy & CI/CD - Quick Reference

> Quick reference for testing in goimg-datalayer. For comprehensive strategy, see `test_strategy.md`.

## Documentation Hierarchy

📚 **Comprehensive Strategy**: [`test_strategy.md`](/home/user/goimg-datalayer/claude/test_strategy.md)
- Complete test patterns for all layers
- Security test requirements (OWASP)
- Sprint-by-sprint test execution matrix
- Agent collaboration model

📖 **Practical Guide**: [`tests/README.md`](/home/user/goimg-datalayer/tests/README.md)
- Day-to-day test commands
- Debugging tips
- Testcontainers setup
- Best practices

📋 **This Guide**: Quick reference for common patterns

---

## Test Pyramid

```
                    ┌───────────────┐
                    │     E2E       │  10-15%
                    │  Newman/API   │  Full system
                    ├───────────────┤
                    │  Integration  │  20-25%
                    │ Testcontainers│  Repository/DB
                    ├───────────────┤
                    │               │
                    │     Unit      │  60-70%
                    │   Domain +    │  Pure logic
                    │   Handlers    │
                    └───────────────┘
```

## Coverage Requirements

| Layer | Minimum | Rationale |
|-------|---------|-----------|
| **Overall** | 80% | Project baseline (CI enforced) |
| **Domain** | 90% | Core business logic |
| **Application** | 85% | Use case coverage |
| **Infrastructure** | 70% | External integrations |
| **Handlers** | 75% | HTTP layer |

## Test Commands

```bash
# Basic Test Commands
make test              # Full suite with race detection
make test-unit         # Unit tests only (-short flag)
make test-integration  # Integration tests (requires DB)
make test-coverage     # Generate HTML coverage report

# E2E Test Commands (290 Newman tests)
make setup-e2e         # Install Newman and dependencies
make test-e2e          # Run all E2E tests (290 tests)
make test-e2e-folder   # Run specific folder (e.g., FOLDER=01-auth)
make test-e2e-dry      # Dry run - show tests without executing
make test-e2e-report   # Generate HTML report in tests/e2e/reports/

# CI/CD Commands
make ci-local          # Run full CI pipeline locally
make test-all          # Run all test types (unit + integration + e2e)
```

**Note**: E2E tests now support both `docker-compose` (standalone) and `docker compose` (plugin) commands automatically.

## Unit Test Patterns

### Value Object Tests

```go
func TestNewEmail(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        input     string
        wantErr   error
        wantValue string
    }{
        {
            name:      "valid email",
            input:     "user@example.com",
            wantValue: "user@example.com",
        },
        {
            name:      "normalizes uppercase",
            input:     "User@Example.COM",
            wantValue: "user@example.com",
        },
        {
            name:    "empty email",
            input:   "",
            wantErr: identity.ErrEmailEmpty,
        },
        {
            name:    "invalid format",
            input:   "not-an-email",
            wantErr: identity.ErrEmailInvalid,
        },
    }

    for _, tt := range tests {
        tt := tt // capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            email, err := identity.NewEmail(tt.input)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.wantValue, email.String())
        })
    }
}
```

### Aggregate Tests

```go
func TestImage_AddVariant(t *testing.T) {
    t.Parallel()

    t.Run("adds variant successfully", func(t *testing.T) {
        image := gallery.NewTestImage(t) // Test helper
        variant := gallery.NewTestVariant(t, gallery.SizeSmall)

        err := image.AddVariant(variant)

        require.NoError(t, err)
        assert.Len(t, image.Variants(), 1)
        assert.Len(t, image.Events(), 1) // Domain event emitted
    })

    t.Run("rejects duplicate size", func(t *testing.T) {
        image := gallery.NewTestImage(t)
        variant := gallery.NewTestVariant(t, gallery.SizeSmall)
        _ = image.AddVariant(variant)

        err := image.AddVariant(variant)

        require.ErrorIs(t, err, gallery.ErrVariantExists)
    })
}
```

## Integration Tests

### Testcontainers Setup

```go
// tests/integration/setup_test.go
type TestSuite struct {
    PostgresContainer testcontainers.Container
    RedisContainer    testcontainers.Container
    PostgresDSN       string
    RedisAddr         string
}

func SetupTestSuite(t *testing.T) *TestSuite {
    t.Helper()
    ctx := context.Background()

    // PostgreSQL
    pgContainer, _ := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("goimg_test"),
    )

    // Redis
    redisContainer, _ := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )

    t.Cleanup(func() {
        _ = pgContainer.Terminate(ctx)
        _ = redisContainer.Terminate(ctx)
    })

    return &TestSuite{...}
}
```

### Repository Tests

```go
func TestUserRepository_FindByEmail(t *testing.T) {
    suite := SetupTestSuite(t)
    repo := postgres.NewUserRepository(suite.PostgresDSN)

    t.Run("finds existing user", func(t *testing.T) {
        // Arrange
        email, _ := identity.NewEmail("test@example.com")
        user := identity.NewTestUser(t, email)
        _ = repo.Save(context.Background(), user)

        // Act
        found, err := repo.FindByEmail(context.Background(), email)

        // Assert
        require.NoError(t, err)
        assert.True(t, user.Email().Equals(found.Email()))
    })

    t.Run("returns not found for missing user", func(t *testing.T) {
        email, _ := identity.NewEmail("missing@example.com")

        _, err := repo.FindByEmail(context.Background(), email)

        require.ErrorIs(t, err, identity.ErrUserNotFound)
    })
}
```

## E2E Tests (Newman)

**Current Status**: ✅ 100% API Coverage - 290 tests across 20 categories

Location: `tests/e2e/postman/`

```bash
# Run all E2E tests
make test-e2e

# Run specific test folder
make test-e2e-folder FOLDER=01-auth

# Dry run (show tests without executing)
make test-e2e-dry

# Generate HTML report
make test-e2e-report
```

### Test Categories (20 Total)

| Category | Tests | Coverage |
|----------|-------|----------|
| Authentication | 15 | Register, Login, Refresh, Logout |
| Two-Factor Auth | 13 | Setup, Verify, Disable, Backup Codes |
| OAuth | 9 | Google/GitHub flows, Link/Unlink |
| Images | 25 | Upload, CRUD, Search, Variants |
| Albums | 18 | CRUD, Nested, Image management |
| Tags | 12 | Popular, Trending, Search |
| Moderation | 17 | Reports, Bans, Reviews |
| NSFW Detection | 8 | Scan, List flagged, History |
| Groups | 27 | CRUD, Membership, Roles |
| Group Albums | 7 | CRUD, Image management |
| Invitations | 6 | Invite, Accept, Decline |
| Follows | 18 | Follow/Unfollow, List |
| Activity Feeds | 4 | User/Group timelines |
| Notifications | 6 | List, Mark read, Count |
| IPFS | 8 | Pin, Unpin, Status |
| Featured Picks | 6 | Feature/Unfeature, List |
| oEmbed | 8 | Rich previews |
| Guest Uploads | 5 | Session, Claim |
| User Profile | 12 | Update, Privacy, Settings |
| Comments & Likes | 20 | Add, Remove, Moderation |

**Total**: 290 tests ensuring all endpoints have E2E coverage

### Collection Structure

```
tests/e2e/postman/
├── goimg-api.postman_collection.json    # Main collection (290 tests)
├── ci.postman_environment.json          # CI environment variables
└── reports/                             # HTML reports (generated)
```

## Contract Tests

Validate implementation matches OpenAPI spec:

```go
// tests/contract/openapi_test.go
func TestAPIMatchesOpenAPISpec(t *testing.T) {
    loader := openapi3.NewLoader()
    doc, _ := loader.LoadFromFile("../../api/openapi/openapi.yaml")
    router, _ := gorillamux.NewRouter(doc)

    // Test each endpoint against spec
    testCases := []struct{
        method, path string
        expectedStatus int
    }{
        {"GET", "/api/v1/health", 200},
        {"POST", "/api/v1/auth/login", 200},
        // ...
    }

    for _, tc := range testCases {
        // Validate request/response against OpenAPI schema
    }
}
```

## CI Pipeline (GitHub Actions)

### Jobs

| Job | Purpose | Tests | Triggers |
|-----|---------|-------|----------|
| `lint` | golangci-lint | N/A | Push, PR |
| `test-unit` | Unit tests + coverage | ~500 | Push, PR |
| `test-integration` | Integration tests | ~150 | Push, PR |
| `test-e2e` | Newman API tests | **290** | Push, PR |
| `contract-validation` | OpenAPI compliance | N/A | Push, PR |
| `security` | gosec, trivy | N/A | Push, PR, Weekly |

### Required Checks

All must pass before merge:
- Lint
- Unit tests
- Integration tests
- Contract validation
- Coverage threshold (80%)

## Observability

### Structured Logging

```go
logger := log.With().
    Str("handler", "image.upload").
    Str("request_id", middleware.GetRequestID(ctx)).
    Logger()

logger.Info().
    Str("content_type", contentType).
    Int64("size", size).
    Msg("processing upload")
```

### Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `goimg_http_requests_total` | Counter | method, path, status |
| `goimg_http_request_duration_seconds` | Histogram | method, path |
| `goimg_image_uploads_total` | Counter | status, format |
| `goimg_image_processing_duration_seconds` | Histogram | operation |

### Health Checks

```
GET /health        # Liveness - is service running?
GET /health/ready  # Readiness - can accept traffic?
```

## Test Fixtures

Location: `tests/fixtures/`

```
tests/fixtures/
├── images/
│   ├── valid_jpeg.jpg
│   ├── valid_png.png
│   └── malware_sample.bin
└── data/
    └── seed.sql
```

## Debugging Tips

1. **Flaky tests**: Check for parallel test interference, use `t.Parallel()` correctly
2. **Integration failures**: Ensure containers are healthy before tests run
3. **Coverage drops**: Run `go tool cover -html=coverage.out` to find gaps
4. **Contract mismatches**: Regenerate with `make generate` and check diff

---

## Agent Workflow

| When to Use | Which Agent | What to Load |
|-------------|-------------|--------------|
| Designing test suite | `backend-test-architect` | `test_strategy.md` |
| Writing unit/integration tests | `backend-test-architect` | `test_strategy.md` + this guide |
| Writing E2E tests | `test-strategist` | `test_strategy.md` (E2E section) |
| Debugging test failures | Any agent | `tests/README.md` (Troubleshooting) |
| Setting up testcontainers | `backend-test-architect` | `tests/README.md` (Testcontainers) |
| Reviewing coverage | `senior-go-architect` | This guide (Coverage Requirements) |

---

## Related Documentation

- **Comprehensive Test Strategy**: [`claude/test_strategy.md`](/home/user/goimg-datalayer/claude/test_strategy.md)
- **Practical Testing Guide**: [`tests/README.md`](/home/user/goimg-datalayer/tests/README.md)
- **Sprint Plan**: [`claude/sprint_plan.md`](/home/user/goimg-datalayer/claude/sprint_plan.md) - Test deliverables per sprint
- **Agent Checklist**: [`claude/agent_checklist.md`](/home/user/goimg-datalayer/claude/agent_checklist.md) - Pre-commit test verification
