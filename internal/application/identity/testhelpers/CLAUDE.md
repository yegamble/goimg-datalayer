# Application Layer Testing Guide

> Comprehensive testing framework for Identity Context application layer (commands and queries).

## Overview

This directory provides a complete testing infrastructure for the application layer, including:

- **Mocks** (`mocks.go`): testify/mock implementations for all external dependencies
- **Fixtures** (`fixtures.go`): Reusable test data and value objects
- **Test Suite** (`setup.go`): Comprehensive test setup helper with common scenarios

## Testing Conventions

### 1. Table-Driven Tests

All command and query handlers should use table-driven tests for comprehensive coverage:

```go
func TestRegisterUserCommand_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        cmd     RegisterUserCommand
        setup   func(t *testing.T, suite *testhelpers.TestSuite)
        want    uuid.UUID
        wantErr error
    }{
        {
            name: "successful registration",
            cmd: RegisterUserCommand{
                Email:    "user@example.com",
                Username: "newuser",
                Password: "SecureP@ss123",
            },
            setup: func(t *testing.T, suite *testhelpers.TestSuite) {
                suite.SetupSuccessfulUserCreation()
            },
            want: testhelpers.ValidUserID,
        },
        {
            name: "duplicate email",
            cmd: RegisterUserCommand{
                Email:    "existing@example.com",
                Username: "newuser",
                Password: "SecureP@ss123",
            },
            setup: func(t *testing.T, suite *testhelpers.TestSuite) {
                existingUser := testhelpers.ValidUser()
                suite.SetupEmailAlreadyExists(existingUser)
            },
            wantErr: ErrEmailAlreadyExists,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Arrange
            suite := testhelpers.NewTestSuite(t)
            if tt.setup != nil {
                tt.setup(t, suite)
            }

            // Act
            result, err := handler.Handle(context.Background(), tt.cmd)

            // Assert
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, result)
            suite.AssertExpectations()
        })
    }
}
```

### 2. Test Structure (Arrange-Act-Assert)

Every test should follow the AAA pattern:

```go
t.Run("test case name", func(t *testing.T) {
    t.Parallel()

    // Arrange - Set up test dependencies
    suite := testhelpers.NewTestSuite(t)
    suite.UserRepo.On("FindByEmail", mock.Anything, email).Return(user, nil)

    handler := NewCommandHandler(suite.UserRepo, suite.JWTService)

    // Act - Execute the code under test
    result, err := handler.Handle(ctx, command)

    // Assert - Verify expectations
    require.NoError(t, err)
    assert.Equal(t, expectedResult, result)
    suite.AssertExpectations()
})
```

### 3. Mocking Strategy

#### Using TestSuite Helpers

For common scenarios, use the built-in setup helpers:

```go
suite := testhelpers.NewTestSuite(t)

// Successful user creation
suite.SetupSuccessfulUserCreation()

// Successful login
user := testhelpers.ValidActiveUser()
suite.SetupSuccessfulLogin(user)

// User not found
suite.SetupUserNotFound()

// Duplicate email
existingUser := testhelpers.ValidUser()
suite.SetupEmailAlreadyExists(existingUser)
```

#### Custom Mock Configuration

For specific test cases, configure mocks directly:

```go
suite := testhelpers.NewTestSuite(t)

suite.UserRepo.On("FindByEmail", mock.Anything, email).
    Return(user, nil).
    Once() // Expect exactly one call

suite.JWTService.On("GenerateAccessToken",
    mock.Anything, mock.Anything, mock.Anything, mock.Anything).
    Return("", fmt.Errorf("token generation failed"))
```

#### Verifying Mock Interactions

Always verify that all expected mock calls were made:

```go
suite.AssertExpectations() // Verifies all mocks
```

### 4. Using Fixtures

Use predefined fixtures for consistent test data:

```go
// Users
user := testhelpers.ValidUser()              // Standard test user
activeUser := testhelpers.ValidActiveUser()  // Active user
adminUser := testhelpers.ValidAdminUser()    // Admin user
suspendedUser := testhelpers.ValidSuspendedUser() // Suspended user

// Value objects
email := testhelpers.ValidEmailVO()
username := testhelpers.ValidUsernameVO()
passwordHash := testhelpers.ValidPasswordHashVO()

// JWT
claims := testhelpers.ValidJWTClaims()
expiredClaims := testhelpers.ExpiredJWTClaims()

// Sessions
pgSession := testhelpers.ValidPostgresSession()
redisSession := testhelpers.ValidRedisSession()

// Tokens
accessToken, refreshToken := testhelpers.ValidTokenPair()
metadata := testhelpers.ValidRefreshTokenMetadata()
```

### 5. Testing Error Cases

Test all error paths explicitly:

```go
tests := []struct {
    name    string
    setup   func(suite *testhelpers.TestSuite)
    wantErr error
}{
    {
        name: "user not found",
        setup: func(suite *testhelpers.TestSuite) {
            suite.SetupUserNotFound()
        },
        wantErr: identity.ErrUserNotFound,
    },
    {
        name: "database connection error",
        setup: func(suite *testhelpers.TestSuite) {
            suite.UserRepo.On("Save", mock.Anything, mock.Anything).
                Return(fmt.Errorf("connection refused"))
        },
        wantErr: ErrDatabaseError,
    },
}
```

### 6. Testing Business Logic

Focus tests on business rules, not infrastructure:

```go
t.Run("cannot register with disposable email", func(t *testing.T) {
    t.Parallel()

    suite := testhelpers.NewTestSuite(t)
    cmd := RegisterUserCommand{
        Email:    "user@mailinator.com", // Disposable
        Username: "testuser",
        Password: "SecureP@ss123",
    }

    _, err := handler.Handle(ctx, cmd)

    require.ErrorIs(t, err, identity.ErrEmailDisposable)
    // No database calls should have been made
    suite.UserRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
})
```

## Coverage Requirements

### Target Coverage: 85% minimum for application layer

| Component | Target | Priority |
|-----------|--------|----------|
| Command handlers | 90% | Critical - contain business logic |
| Query handlers | 80% | High - data retrieval |
| DTOs and mappers | 70% | Medium - mostly validation |
| Application services | 85% | High - orchestrate domain |

### Critical Paths Requiring 100% Coverage

1. **Authentication flows**: Login, token refresh, logout
2. **Authorization checks**: Role-based access control
3. **User registration**: Email uniqueness, password validation
4. **Session management**: Creation, revocation, expiry
5. **Error handling**: All error paths must be tested

## Running Tests

### Run All Application Layer Tests

```bash
go test -race ./internal/application/... -cover
```

### Run Specific Test

```bash
go test -race -run TestRegisterUserCommand ./internal/application/identity/commands
```

### Generate Coverage Report

```bash
go test -race ./internal/application/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Check Coverage Threshold

```bash
# Ensure minimum 85% coverage
go test ./internal/application/... -cover | grep -E 'coverage: [0-9]+' | awk '{if ($2 < 85.0) exit 1}'
```

## Common Testing Patterns

### Pattern 1: Testing Command Validation

```go
func TestCommand_Validate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        cmd     MyCommand
        wantErr error
    }{
        {
            name: "valid command",
            cmd:  MyCommand{Field: "valid"},
        },
        {
            name:    "empty field",
            cmd:     MyCommand{Field: ""},
            wantErr: ErrFieldRequired,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            err := tt.cmd.Validate()
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Pattern 2: Testing Query Filters

```go
func TestListUsersQuery_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name   string
        filter UserFilter
        setup  func(suite *testhelpers.TestSuite)
        want   int // Expected user count
    }{
        {
            name:   "filter by role",
            filter: UserFilter{Role: identity.RoleAdmin},
            setup: func(suite *testhelpers.TestSuite) {
                adminUsers := []*identity.User{testhelpers.ValidAdminUser()}
                suite.UserRepo.On("List", mock.Anything, mock.Anything, mock.Anything).
                    Return(adminUsers, 1, nil)
            },
            want: 1,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            suite := testhelpers.NewTestSuite(t)
            tt.setup(suite)

            query := ListUsersQuery{Filter: tt.filter}
            result, err := handler.Handle(context.Background(), query)

            require.NoError(t, err)
            assert.Len(t, result.Users, tt.want)
            suite.AssertExpectations()
        })
    }
}
```

### Pattern 3: Testing Event Emission

```go
func TestCommand_EmitsDomainEvents(t *testing.T) {
    t.Parallel()

    suite := testhelpers.NewTestSuite(t)
    suite.SetupSuccessfulUserCreation()

    // Capture saved user to check events
    var capturedUser *identity.User
    suite.UserRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
        capturedUser = u
        return true
    })).Return(nil)

    cmd := RegisterUserCommand{...}
    _, err := handler.Handle(context.Background(), cmd)

    require.NoError(t, err)
    require.NotNil(t, capturedUser)
    events := capturedUser.Events()
    require.Len(t, events, 1)
    assert.Equal(t, "identity.user.created", events[0].EventType())
}
```

### Pattern 4: Testing Concurrent Operations

```go
func TestConcurrentUserRegistration(t *testing.T) {
    t.Parallel()

    suite := testhelpers.NewTestSuite(t)

    // First registration succeeds
    suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
        Return(nil, identity.ErrUserNotFound).Once()
    suite.UserRepo.On("Save", mock.Anything, mock.Anything).
        Return(nil).Once()

    // Second registration detects duplicate
    suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
        Return(testhelpers.ValidUser(), nil).Once()

    cmd := RegisterUserCommand{...}

    // Execute concurrently
    var wg sync.WaitGroup
    results := make(chan error, 2)

    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := handler.Handle(context.Background(), cmd)
            results <- err
        }()
    }

    wg.Wait()
    close(results)

    // One should succeed, one should fail with duplicate error
    var successCount, errorCount int
    for err := range results {
        if err == nil {
            successCount++
        } else {
            errorCount++
        }
    }

    assert.Equal(t, 1, successCount)
    assert.Equal(t, 1, errorCount)
}
```

## Anti-Patterns to Avoid

### 1. Testing Mocks Instead of Behavior

**Bad:**
```go
suite.UserRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
handler.Handle(ctx, cmd)
suite.UserRepo.AssertCalled(t, "Save", mock.Anything, mock.Anything)
```

**Good:**
```go
suite.SetupSuccessfulUserCreation()
userID, err := handler.Handle(ctx, cmd)
require.NoError(t, err)
assert.False(t, userID.IsZero())
```

### 2. Not Using Parallel Tests

**Bad:**
```go
func TestMyCommand(t *testing.T) {
    // No t.Parallel()
    ...
}
```

**Good:**
```go
func TestMyCommand(t *testing.T) {
    t.Parallel() // Enables concurrent test execution
    ...
}
```

### 3. Hardcoding Test Values

**Bad:**
```go
email, _ := identity.NewEmail("test@example.com")
username, _ := identity.NewUsername("testuser")
```

**Good:**
```go
email := testhelpers.ValidEmailVO()
username := testhelpers.ValidUsernameVO()
```

### 4. Not Cleaning Up Events

**Bad:**
```go
user := testhelpers.ValidUser()
// User has UserCreated event from factory
```

**Good:**
```go
user := testhelpers.ValidUser()
user.ClearEvents() // Start with clean slate
```

### 5. Swallowing Errors in Setup

**Bad:**
```go
user, _ := identity.NewUser(email, username, hash) // Ignores error
```

**Good:**
```go
user, err := identity.NewUser(email, username, hash)
require.NoError(t, err) // Fail fast if setup is invalid
```

## Debugging Tests

### Enable Verbose Output

```bash
go test -v ./internal/application/identity/commands
```

### Run with Race Detector

```bash
go test -race ./internal/application/...
```

### Debug Specific Test

```bash
go test -run TestRegisterUserCommand/successful_registration -v
```

### Print Mock Call History

```go
suite.UserRepo.AssertExpectations(t)
// If assertion fails, mock will print all actual calls
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Run Application Layer Tests
  run: |
    go test -race -coverprofile=coverage.out ./internal/application/...

- name: Check Coverage Threshold
  run: |
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 85" | bc -l) )); then
      echo "Coverage $coverage% is below 85% threshold"
      exit 1
    fi
```

## Best Practices Summary

1. **Always use `t.Parallel()`** for concurrent test execution
2. **Use TestSuite helpers** for common scenarios
3. **Test business logic**, not infrastructure
4. **Verify all mock expectations** with `suite.AssertExpectations()`
5. **Use fixtures** for consistent test data
6. **Test error paths** explicitly
7. **Follow AAA pattern**: Arrange, Act, Assert
8. **Clear domain events** when using fixtures
9. **Use table-driven tests** for multiple scenarios
10. **Target 85% coverage** minimum

## References

- Domain layer guide: `/home/user/goimg-datalayer/internal/domain/CLAUDE.md`
- Test strategy: `/home/user/goimg-datalayer/claude/test_strategy.md`
- testify/mock docs: https://pkg.go.dev/github.com/stretchr/testify/mock
