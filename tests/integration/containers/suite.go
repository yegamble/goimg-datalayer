package containers

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// IntegrationTestSuite manages all test containers for integration tests.
// It provides convenient access to database and Redis connections.
type IntegrationTestSuite struct {
	Postgres    *PostgresContainer
	Redis       *RedisContainer
	DB          *sqlx.DB
	RedisClient *redis.Client
	ctx         context.Context
	t           *testing.T
}

// NewIntegrationTestSuite creates and initializes all containers needed for integration tests.
// It automatically registers cleanup handlers to terminate containers when the test finishes.
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	t.Helper()

	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := NewPostgresContainer(ctx, t)
	require.NoError(t, err, "failed to start postgres container")

	// Start Redis container
	redisContainer, err := NewRedisContainer(ctx, t)
	require.NoError(t, err, "failed to start redis container")

	// Register cleanup to terminate containers
	t.Cleanup(func() {
		if postgresContainer != nil {
			_ = postgresContainer.Terminate(ctx)
		}
		if redisContainer != nil {
			_ = redisContainer.Terminate(ctx)
		}
	})

	return &IntegrationTestSuite{
		Postgres:    postgresContainer,
		Redis:       redisContainer,
		DB:          postgresContainer.DB,
		RedisClient: redisContainer.Client,
		ctx:         ctx,
		t:           t,
	}
}

// CleanupBetweenTests truncates all database tables and flushes Redis.
// Call this between subtests to ensure isolation.
func (s *IntegrationTestSuite) CleanupBetweenTests() {
	s.t.Helper()

	if s.Postgres != nil {
		s.Postgres.Cleanup(s.ctx, s.t)
	}
	if s.Redis != nil {
		s.Redis.Cleanup(s.ctx, s.t)
	}
}

// Context returns the test context.
func (s *IntegrationTestSuite) Context() context.Context {
	return s.ctx
}

// T returns the testing.T instance.
func (s *IntegrationTestSuite) T() *testing.T {
	return s.t
}

// RequireNoError is a helper that fails the test if err is not nil.
func (s *IntegrationTestSuite) RequireNoError(err error, msgAndArgs ...interface{}) {
	s.t.Helper()
	require.NoError(s.t, err, msgAndArgs...)
}

// RequireEqual is a helper that fails the test if expected != actual.
func (s *IntegrationTestSuite) RequireEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	s.t.Helper()
	require.Equal(s.t, expected, actual, msgAndArgs...)
}
