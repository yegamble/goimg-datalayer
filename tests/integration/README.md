# Integration Tests

This directory contains integration tests for the goimg-datalayer project using testcontainers for PostgreSQL and Redis.

## Overview

Integration tests verify that components work together correctly with real dependencies (database, cache, etc.). We use [testcontainers-go](https://golang.testcontainers.org/) to spin up PostgreSQL and Redis containers automatically during test execution.

## Prerequisites

- **Docker**: Must be running on your machine
- **Go 1.24+**: For running tests
- **5GB+ free disk space**: For Docker images

## Running Integration Tests

```bash
# Run all integration tests
make test-integration

# Run specific integration test
go test -race -tags=integration -v ./tests/integration -run TestUserRepository_Create

# Run with short mode (skips integration tests)
go test -short ./...
```

## Test Structure

```
tests/integration/
├── containers/                    # Testcontainer setup
│   ├── postgres.go               # PostgreSQL container with migrations
│   ├── redis.go                  # Redis container
│   └── suite.go                  # Integration test suite base
├── fixtures/                     # Test data and factories
│   ├── users.go                  # User fixtures
│   ├── sessions.go               # Session fixtures
│   └── jwt.go                    # JWT test keys
├── user_repository_test.go       # User repository integration tests
├── session_repository_test.go    # Session repository integration tests
├── session_store_test.go         # Redis session store tests
├── token_blacklist_test.go       # Redis token blacklist tests
└── README.md                     # This file
```

## Container Lifecycle

### Automatic Management

The `IntegrationTestSuite` automatically manages container lifecycle:

1. **Startup**: Containers start when `NewIntegrationTestSuite()` is called
2. **Migrations**: PostgreSQL migrations run automatically on startup
3. **Cleanup**: Containers terminate automatically when test completes (`t.Cleanup()`)

### Manual Container Control

```go
// Start containers manually
suite := containers.NewIntegrationTestSuite(t)

// Clean database between subtests
suite.CleanupBetweenTests()

// Access containers directly
suite.DB          // *sqlx.DB
suite.RedisClient // *redis.Client
suite.Postgres    // *PostgresContainer
suite.Redis       // *RedisContainer
```

## Writing Integration Tests

### Basic Pattern

```go
// +build integration

package integration_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/yegamble/goimg-datalayer/tests/integration/containers"
    "github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

func TestMyFeature_Integration(t *testing.T) {
    // Setup test suite with containers
    suite := containers.NewIntegrationTestSuite(t)
    ctx := context.Background()

    // Arrange - use fixtures for test data
    userFixture := fixtures.ValidUser(t)
    user := userFixture.ToEntity(t)

    // Act - test your code
    // repo := postgres.NewUserRepository(suite.DB)
    // err := repo.Save(ctx, user)

    // Assert
    // require.NoError(t, err)

    t.Skip("Skipping until repository implementation is available")
}
```

### Table-Driven Integration Tests

```go
func TestRepository_MultipleScenarios(t *testing.T) {
    suite := containers.NewIntegrationTestSuite(t)
    ctx := context.Background()

    tests := []struct {
        name    string
        fixture *fixtures.UserFixture
        wantErr error
    }{
        {"valid user", fixtures.ValidUser(t), nil},
        {"suspended user", fixtures.SuspendedUser(t), nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Clean database between subtests
            suite.CleanupBetweenTests()

            // Test logic here
        })
    }
}
```

### Using Fixtures

```go
// Simple fixture
user := fixtures.ValidUser(t)

// Customized fixture
admin := fixtures.ValidUser(t).WithRole(identity.RoleAdmin).WithEmail("admin@example.com")

// Multiple unique users
user1 := fixtures.UniqueUser(t, "user1")
user2 := fixtures.UniqueUser(t, "user2")

// Convert to domain entity
entity := userFixture.ToEntity(t)
```

## Test Containers

### PostgreSQL Container

- **Image**: `postgres:16-alpine`
- **Features**:
  - Automatic migration execution via Goose
  - Health check waiting
  - Cleanup/truncate between tests
- **Connection**: Available via `suite.DB`

### Redis Container

- **Image**: `redis:7-alpine`
- **Features**:
  - Automatic health check waiting
  - FlushAll cleanup between tests
- **Connection**: Available via `suite.RedisClient`

## Coverage Requirements

| Component | Minimum Coverage | Focus |
|-----------|-----------------|-------|
| Repository implementations | 80% | CRUD operations, error handling, transactions |
| Redis stores | 70% | Cache operations, TTL, expiration |
| Integration points | 75% | Component interactions, data flow |

## Troubleshooting

### Docker Connection Issues

```bash
# Check if Docker is running
docker info

# Check Docker socket permissions
ls -la /var/run/docker.sock

# On Linux, add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```

### Tests Hang or Timeout

- Increase timeout: `go test -timeout=15m ...`
- Check Docker resource limits (CPU, memory)
- Verify network connectivity to Docker daemon

### Container Cleanup Issues

```bash
# List all containers
docker ps -a

# Remove testcontainers manually
docker rm -f $(docker ps -a -q --filter "label=org.testcontainers")

# Clean up volumes
docker volume prune
```

### Migration Errors

```bash
# Verify migrations are valid
cd migrations
goose postgres "connection_string" validate

# Check migration files
ls -la migrations/
```

## Best Practices

1. **Use `t.Parallel()` cautiously**: Integration tests with shared resources may not be safe to parallelize
2. **Clean between tests**: Always call `suite.CleanupBetweenTests()` between subtests
3. **Use fixtures**: Don't hardcode test data; use fixtures for consistency
4. **Test real scenarios**: Integration tests should test real-world use cases, not just happy paths
5. **Keep tests fast**: Aim for < 60 seconds total runtime
6. **Verify cleanup**: Ensure tests don't leak resources (containers, connections)

## Integration with CI/CD

Integration tests run automatically in GitHub Actions:

```yaml
- name: Run integration tests
  run: make test-integration
  env:
    DOCKER_HOST: unix:///var/run/docker.sock
```

## See Also

- [Testcontainers Go Documentation](https://golang.testcontainers.org/)
- [Project Test Strategy](../CLAUDE.md)
- [Test Fixtures Guide](./fixtures/README.md)
- [Domain Tests](../../internal/domain/CLAUDE.md)
