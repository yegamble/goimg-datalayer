# Tests Guide

> Unit, integration, E2E, and contract tests for goimg-datalayer.

## Critical Rules

1. **Test pyramid** - 60-70% unit, 20-25% integration, 10-15% E2E
2. **Coverage targets** - 80% overall, 90% domain, 85% application, 70% infrastructure, 75% HTTP
3. **Parallel tests** - Use `t.Parallel()` for all unit tests
4. **Table-driven** - Prefer table-driven tests for clarity and maintainability
5. **Test helpers** - Use `t.Helper()` for setup functions
6. **No test interdependence** - Tests must be independent and idempotent
7. **Testcontainers for integration** - Use Docker containers for real dependencies

## Structure

```
tests/
├── integration/          # Integration tests (testcontainers)
│   ├── setup_test.go     # Container setup helpers
│   ├── user_test.go      # User repository integration tests
│   ├── image_test.go     # Image repository integration tests
│   └── cache_test.go     # Redis cache integration tests
├── e2e/                  # End-to-end API tests
│   ├── postman/
│   │   └── goimg-collection.json
│   ├── newman/
│   │   └── run_tests.sh
│   └── api_test.go       # Go-based E2E tests
├── contract/             # OpenAPI compliance tests
│   └── openapi_test.go
├── fixtures/             # Test data
│   ├── images/
│   │   ├── test.jpg
│   │   ├── test.png
│   │   └── invalid.txt
│   └── data/
│       ├── users.json
│       └── images.json
└── mocks/                # Generated mocks (testify/mockery)
    ├── user_repository_mock.go
    └── event_publisher_mock.go
```

## Commands

```bash
# Full test suite
make test

# Unit tests only (fast)
make test-unit

# Integration tests (requires Docker)
make test-integration

# E2E tests
make test-e2e

# Coverage report (HTML)
make test-coverage

# Race detector
go test -race ./...

# Specific package
go test -v ./internal/domain/identity

# Single test
go test -v -run TestNewEmail ./internal/domain/identity
```

## Unit Tests (Domain & Application)

### Table-Driven Test Pattern

```go
package identity_test

import (
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/domain/identity"
)

func TestNewEmail(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        input     string
        wantValue string
        wantErr   error
    }{
        {
            name:      "valid email",
            input:     "user@example.com",
            wantValue: "user@example.com",
            wantErr:   nil,
        },
        {
            name:    "empty email",
            input:   "",
            wantErr: identity.ErrEmailEmpty,
        },
        {
            name:    "invalid format - no @",
            input:   "notanemail",
            wantErr: identity.ErrEmailInvalid,
        },
        {
            name:    "invalid format - no domain",
            input:   "user@",
            wantErr: identity.ErrEmailInvalid,
        },
        {
            name:      "whitespace trimmed",
            input:     "  user@example.com  ",
            wantValue: "user@example.com",
            wantErr:   nil,
        },
        {
            name:      "uppercase normalized to lowercase",
            input:     "User@Example.COM",
            wantValue: "user@example.com",
            wantErr:   nil,
        },
        {
            name:    "exceeds max length",
            input:   strings.Repeat("a", 250) + "@test.com",
            wantErr: identity.ErrEmailTooLong,
        },
        {
            name:      "unicode characters",
            input:     "user+tag@example.com",
            wantValue: "user+tag@example.com",
            wantErr:   nil,
        },
    }

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Act
            email, err := identity.NewEmail(tt.input)

            // Assert
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                assert.True(t, email.IsEmpty())
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.wantValue, email.String())
                assert.False(t, email.IsEmpty())
            }
        })
    }
}
```

### Testing Domain Entities

```go
package identity_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/domain/identity"
)

func TestUser_ChangeEmail(t *testing.T) {
    t.Parallel()

    // Arrange
    oldEmail, _ := identity.NewEmail("old@example.com")
    newEmail, _ := identity.NewEmail("new@example.com")
    username, _ := identity.NewUsername("testuser")
    password, _ := identity.HashPassword("password123")

    user, err := identity.NewUser(oldEmail, username, password)
    require.NoError(t, err)

    // Clear creation event for clean assertion
    user.ClearEvents()

    // Act
    err = user.ChangeEmail(newEmail)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, newEmail, user.Email())
    assert.Len(t, user.Events(), 1)

    event := user.Events()[0]
    assert.Equal(t, "identity.user.email_changed", event.EventType())

    emailChangedEvent, ok := event.(identity.UserEmailChangedEvent)
    require.True(t, ok)
    assert.Equal(t, newEmail, emailChangedEvent.NewEmail)
}

func TestUser_ChangeEmail_NoOp(t *testing.T) {
    t.Parallel()

    // Arrange
    email, _ := identity.NewEmail("test@example.com")
    username, _ := identity.NewUsername("testuser")
    password, _ := identity.HashPassword("password123")

    user, _ := identity.NewUser(email, username, password)
    user.ClearEvents()

    // Act - change to same email
    err := user.ChangeEmail(email)

    // Assert - should be no-op
    require.NoError(t, err)
    assert.Len(t, user.Events(), 0)
}
```

### Testing Application Handlers with Mocks

```go
package commands_test

import (
    "context"
    "testing"

    "github.com/rs/zerolog"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/application/commands"
    "goimg-datalayer/internal/domain/identity"
)

// Mock repository (using testify/mock)
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
    args := m.Called(ctx, username)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

// Mock event publisher
type MockEventPublisher struct {
    mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
    args := m.Called(ctx, event)
    return args.Error(0)
}

func TestCreateUserHandler_Handle_Success(t *testing.T) {
    t.Parallel()

    // Arrange
    mockRepo := new(MockUserRepository)
    mockPublisher := new(MockEventPublisher)
    logger := zerolog.Nop()

    handler := commands.NewCreateUserHandler(mockRepo, mockPublisher, &logger)

    cmd := commands.CreateUserCommand{
        Email:    "test@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    // Set up mock expectations
    email, _ := identity.NewEmail(cmd.Email)
    username, _ := identity.NewUsername(cmd.Username)

    mockRepo.On("FindByEmail", mock.Anything, email).
        Return(nil, identity.ErrUserNotFound)
    mockRepo.On("FindByUsername", mock.Anything, username).
        Return(nil, identity.ErrUserNotFound)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).
        Return(nil)
    mockPublisher.On("Publish", mock.Anything, mock.Anything).
        Return(nil)

    // Act
    user, err := handler.Handle(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, email, user.Email())
    assert.Equal(t, username, user.Username())

    mockRepo.AssertExpectations(t)
    mockPublisher.AssertExpectations(t)
}

func TestCreateUserHandler_Handle_DuplicateEmail(t *testing.T) {
    t.Parallel()

    // Arrange
    mockRepo := new(MockUserRepository)
    mockPublisher := new(MockEventPublisher)
    logger := zerolog.Nop()

    handler := commands.NewCreateUserHandler(mockRepo, mockPublisher, &logger)

    cmd := commands.CreateUserCommand{
        Email:    "existing@example.com",
        Username: "testuser",
        Password: "Password123!",
    }

    email, _ := identity.NewEmail(cmd.Email)
    existingUser, _ := identity.NewUser(
        email,
        identity.MustUsername("existing"),
        identity.MustHashPassword("pass"),
    )

    mockRepo.On("FindByEmail", mock.Anything, email).
        Return(existingUser, nil)

    // Act
    user, err := handler.Handle(context.Background(), cmd)

    // Assert
    assert.Nil(t, user)
    require.ErrorIs(t, err, identity.ErrUserAlreadyExists)

    mockRepo.AssertExpectations(t)
    mockPublisher.AssertNotCalled(t, "Publish")
}
```

## Integration Tests with testcontainers

### Setup Test Suite

```go
package integration_test

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/jmoiron/sqlx"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"

    "goimg-datalayer/internal/infrastructure/persistence/postgres"
)

type TestSuite struct {
    DB             *sqlx.DB
    RedisClient    *redis.Client
    PostgresCleanup func()
    RedisCleanup    func()
}

func SetupTestSuite(t *testing.T) *TestSuite {
    t.Helper()

    ctx := context.Background()

    // Start Postgres container
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        postgres.WithInitScripts("../internal/infrastructure/persistence/postgres/migrations"),
    )
    require.NoError(t, err)

    connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := sqlx.Connect("postgres", connStr)
    require.NoError(t, err)

    // Start Redis container
    redisContainer, err := rediscontainer.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)

    redisAddr, err := redisContainer.ConnectionString(ctx)
    require.NoError(t, err)

    redisClient := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })

    err = redisClient.Ping(ctx).Err()
    require.NoError(t, err)

    suite := &TestSuite{
        DB:          db,
        RedisClient: redisClient,
        PostgresCleanup: func() {
            _ = db.Close()
            _ = postgresContainer.Terminate(ctx)
        },
        RedisCleanup: func() {
            _ = redisClient.Close()
            _ = redisContainer.Terminate(ctx)
        },
    }

    t.Cleanup(func() {
        suite.PostgresCleanup()
        suite.RedisCleanup()
    })

    return suite
}

// CleanDatabase removes all data from tables (keeps schema)
func (s *TestSuite) CleanDatabase(t *testing.T) {
    t.Helper()

    tables := []string{"images", "users", "sessions"}
    for _, table := range tables {
        _, err := s.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
        require.NoError(t, err)
    }
}
```

### Integration Test Example

```go
package integration_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/domain/identity"
    "goimg-datalayer/internal/infrastructure/persistence/postgres"
)

func TestPostgresUserRepository_SaveAndFindByID(t *testing.T) {
    // Setup
    suite := SetupTestSuite(t)
    suite.CleanDatabase(t)

    repo := postgres.NewPostgresUserRepository(suite.DB)

    // Arrange
    email, _ := identity.NewEmail("test@example.com")
    username, _ := identity.NewUsername("testuser")
    password, _ := identity.HashPassword("password123")

    user, err := identity.NewUser(email, username, password)
    require.NoError(t, err)

    // Act - Save
    err = repo.Save(context.Background(), user)
    require.NoError(t, err)

    // Act - Find
    found, err := repo.FindByID(context.Background(), user.ID())

    // Assert
    require.NoError(t, err)
    assert.Equal(t, user.ID(), found.ID())
    assert.Equal(t, user.Email(), found.Email())
    assert.Equal(t, user.Username(), found.Username())
}

func TestPostgresUserRepository_FindByEmail_NotFound(t *testing.T) {
    suite := SetupTestSuite(t)
    suite.CleanDatabase(t)

    repo := postgres.NewPostgresUserRepository(suite.DB)

    email, _ := identity.NewEmail("nonexistent@example.com")

    user, err := repo.FindByEmail(context.Background(), email)

    assert.Nil(t, user)
    require.ErrorIs(t, err, identity.ErrUserNotFound)
}
```

## HTTP Handler Tests

### Testing Handlers with httptest

```go
package handlers_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/interfaces/http/handlers"
    "goimg-datalayer/internal/interfaces/http/dto/requests"
    "goimg-datalayer/internal/interfaces/http/dto/responses"
)

func TestUserHandler_GetByID_Success(t *testing.T) {
    // Arrange
    mockGetUser := new(MockGetUserHandler)
    logger := zerolog.Nop()

    handler := handlers.NewUserHandler(nil, mockGetUser, nil, nil, &logger)

    userID := "123e4567-e89b-12d3-a456-426614174000"
    expectedUser := createMockUser()

    mockGetUser.On("Handle", mock.Anything, mock.MatchedBy(func(q queries.GetUserQuery) bool {
        return q.UserID == userID
    })).Return(expectedUser, nil)

    // Create request with chi context
    req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("userID", userID)
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

    rec := httptest.NewRecorder()

    // Act
    handler.GetByID(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)

    var resp responses.UserResponse
    err := json.NewDecoder(rec.Body).Decode(&resp)
    require.NoError(t, err)

    assert.Equal(t, expectedUser.ID().String(), resp.ID)
    assert.Equal(t, expectedUser.Email().String(), resp.Email)

    mockGetUser.AssertExpectations(t)
}
```

### Testing Middleware

```go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"

    "goimg-datalayer/internal/interfaces/http/middleware"
)

func TestSecurityHeaders(t *testing.T) {
    // Arrange
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    wrapped := middleware.SecurityHeaders(handler)

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()

    // Act
    wrapped.ServeHTTP(rec, req)

    // Assert
    assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
    assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
    assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
    assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=31536000")
    assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
}
```

## Test Fixtures

### Loading Test Data

```go
package fixtures

import (
    "encoding/json"
    "os"
    "testing"

    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/domain/identity"
)

type UserFixture struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    Password string `json:"password"`
    Role     string `json:"role"`
}

func LoadUserFixtures(t *testing.T) []UserFixture {
    t.Helper()

    data, err := os.ReadFile("testdata/users.json")
    require.NoError(t, err)

    var fixtures []UserFixture
    err = json.Unmarshal(data, &fixtures)
    require.NoError(t, err)

    return fixtures
}

func CreateTestUser(t *testing.T, email, username, password string) *identity.User {
    t.Helper()

    emailVO, err := identity.NewEmail(email)
    require.NoError(t, err)

    usernameVO, err := identity.NewUsername(username)
    require.NoError(t, err)

    passwordHash, err := identity.HashPassword(password)
    require.NoError(t, err)

    user, err := identity.NewUser(emailVO, usernameVO, passwordHash)
    require.NoError(t, err)

    return user
}
```

## Coverage Requirements by Layer

| Layer              | Target | Focus Areas |
|--------------------|--------|-------------|
| Domain             | 90%    | Entities, value objects, business methods |
| Application        | 85%    | Command/query handlers, error handling |
| Infrastructure     | 70%    | Repository implementations, mapping logic |
| HTTP               | 75%    | Handlers, error mapping, middleware |
| **Overall Project**| **80%**| All production code |

### Checking Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# Check coverage by package
go test -cover ./internal/domain/...
go test -cover ./internal/application/...
go test -cover ./internal/infrastructure/...
go test -cover ./internal/interfaces/http/...

# Fail if below threshold
go test -coverprofile=coverage.out ./... && \
  go tool cover -func=coverage.out | grep total | awk '{print $3}' | \
  awk -F% '{if ($1 < 80) exit 1}'
```

## Common Test Helpers

### Assert Helpers

```go
package testutil

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "goimg-datalayer/internal/domain/identity"
)

func AssertUserEquals(t *testing.T, expected, actual *identity.User) {
    t.Helper()

    assert.Equal(t, expected.ID(), actual.ID())
    assert.Equal(t, expected.Email(), actual.Email())
    assert.Equal(t, expected.Username(), actual.Username())
    assert.Equal(t, expected.Role(), actual.Role())
    assert.Equal(t, expected.Status(), actual.Status())
}

func AssertErrorContains(t *testing.T, err error, substring string) {
    t.Helper()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), substring)
}
```

## Best Practices

1. **Test naming**: `Test<Function>_<Scenario>_<ExpectedResult>`
2. **Arrange-Act-Assert**: Clear test structure
3. **Single assertion focus**: Each test should verify one behavior
4. **Clean up resources**: Use `t.Cleanup()` for teardown
5. **Avoid sleeps**: Use synchronization primitives or polling
6. **Test error paths**: Don't just test happy paths
7. **Use constants**: Define test constants for magic values

## Anti-Patterns to Avoid

1. **Fragile tests**: Tests that break on unrelated changes
2. **Test interdependence**: Tests that depend on execution order
3. **Excessive mocking**: Mocking too many layers
4. **Testing implementation**: Test behavior, not implementation details
5. **Ignoring test failures**: All tests must pass before merging
6. **Missing cleanup**: Leaking resources between tests

## Agent Responsibilities

- **test-strategist**: Reviews overall test strategy, coverage targets
- **senior-go-architect**: Reviews integration test setup, testcontainers usage
- **backend-developer**: Writes unit and integration tests
- **All agents**: Must run tests before committing

## See Also

- Testing & CI guide: `claude/testing_ci.md`
- Domain testing: `internal/domain/CLAUDE.md`
- Application testing: `internal/application/CLAUDE.md`
- Infrastructure testing: `internal/infrastructure/CLAUDE.md`
- HTTP testing: `internal/interfaces/http/CLAUDE.md`
