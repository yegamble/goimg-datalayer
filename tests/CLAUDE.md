# Tests Guide

> Unit, integration, E2E, and contract tests.

## Key Rules

1. **Test pyramid** - 60-70% unit, 20-25% integration, 10-15% E2E
2. **Coverage targets** - 80% overall, 90% domain
3. **Parallel tests** - Use `t.Parallel()` where possible
4. **Table-driven** - Prefer table-driven tests for clarity
5. **Test helpers** - Use `t.Helper()` for setup functions

## Structure

```
tests/
├── integration/          # Testcontainers-based tests
│   ├── setup_test.go     # Container setup
│   ├── user_test.go
│   └── image_test.go
├── e2e/                  # End-to-end API tests
│   ├── postman/
│   │   └── goimg-collection.json
│   └── newman/
│       └── run_tests.sh
├── contract/             # OpenAPI compliance tests
│   └── openapi_test.go
└── fixtures/             # Test data
    ├── images/
    └── data/
```

## Commands

```bash
make test              # Full suite
make test-unit         # Unit tests only
make test-integration  # Integration (needs DB)
make test-e2e          # Newman/Postman
make test-coverage     # HTML coverage report
```

## Unit Test Pattern

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
        {"invalid", "not-email", ErrEmailInvalid},
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

## Integration Test Setup

```go
func SetupTestSuite(t *testing.T) *TestSuite {
    t.Helper()
    // Start Postgres and Redis containers
    // Run migrations
    // Return connection strings
    t.Cleanup(func() { /* terminate containers */ })
}
```

## See Also

- Full testing guide: `claude/testing_ci.md`
- CI pipeline: `.github/workflows/ci.yml`
