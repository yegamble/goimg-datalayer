package redis

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// setupRedisContainer starts a Redis container and returns its configuration.
// It automatically cleans up the container when the test ends.
func setupRedisContainer(t *testing.T) Config {
	t.Helper()

	ctx := context.Background()

	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err, "failed to start redis container")

	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	})

	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err, "failed to get redis connection string")

	// Parse connection string (redis://host:port)
	u, err := url.Parse(connStr)
	require.NoError(t, err, "failed to parse redis connection string")

	host := u.Hostname()
	portStr := u.Port()
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "failed to parse redis port")

	return Config{
		Host:     host,
		Port:     port,
		PoolSize: 5,
		MinIdle:  2,
		MaxRetry: 2,
		Timeout:  5 * time.Second,
	}
}
