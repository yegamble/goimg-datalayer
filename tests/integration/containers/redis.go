package containers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer represents a running Redis testcontainer.
type RedisContainer struct {
	Container testcontainers.Container
	Client    *redis.Client
	Addr      string
}

// NewRedisContainer creates and starts a Redis 7 testcontainer.
func NewRedisContainer(ctx context.Context, t *testing.T) (*RedisContainer, error) {
	t.Helper()

	// Start Redis container
	redisC, err := rediscontainer.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	// Get connection string
	connStr, err := redisC.ConnectionString(ctx)
	if err != nil {
		_ = redisC.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis connection string: %w", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:         connStr,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		_ = redisC.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisContainer{
		Container: redisC,
		Client:    client,
		Addr:      connStr,
	}, nil
}

// Cleanup flushes all Redis data.
// This is faster than recreating the container for each test.
func (rc *RedisContainer) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()

	err := rc.Client.FlushAll(ctx).Err()
	require.NoError(t, err, "failed to flush redis")
}

// Terminate stops and removes the container.
func (rc *RedisContainer) Terminate(ctx context.Context) error {
	if rc.Client != nil {
		_ = rc.Client.Close()
	}
	if rc.Container != nil {
		if err := rc.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("terminate redis container: %w", err)
		}
	}
	return nil
}

// WaitForHealthy waits for Redis to be ready.
func (rc *RedisContainer) WaitForHealthy(ctx context.Context, t *testing.T) {
	t.Helper()

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if err := rc.Client.Ping(ctx).Err(); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.Fail(t, "redis did not become healthy within timeout")
}

// SetKey is a helper to set a key with expiration.
func (rc *RedisContainer) SetKey(ctx context.Context, key, value string, expiration time.Duration) error {
	if err := rc.Client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("redis set key %s: %w", key, err)
	}
	return nil
}

// GetKey is a helper to get a key.
func (rc *RedisContainer) GetKey(ctx context.Context, key string) (string, error) {
	val, err := rc.Client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get key %s: %w", key, err)
	}
	return val, nil
}

// KeyExists checks if a key exists.
func (rc *RedisContainer) KeyExists(ctx context.Context, key string) (bool, error) {
	result, err := rc.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis check key exists %s: %w", key, err)
	}
	return result > 0, nil
}
