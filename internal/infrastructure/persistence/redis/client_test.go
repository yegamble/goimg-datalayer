package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6379, cfg.Port)
	assert.Empty(t, cfg.Password)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Equal(t, 5, cfg.MinIdle)
	assert.Equal(t, 3, cfg.MaxRetry)
	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestNewClient_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       Config
		wantError string
	}{
		{
			name: "empty host",
			cfg: Config{
				Host: "",
				Port: 6379,
			},
			wantError: "redis host cannot be empty",
		},
		{
			name: "invalid port - zero",
			cfg: Config{
				Host: "localhost",
				Port: 0,
			},
			wantError: "invalid redis port: 0",
		},
		{
			name: "invalid port - negative",
			cfg: Config{
				Host: "localhost",
				Port: -1,
			},
			wantError: "invalid redis port: -1",
		},
		{
			name: "invalid port - too large",
			cfg: Config{
				Host: "localhost",
				Port: 65536,
			},
			wantError: "invalid redis port: 65536",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(tt.cfg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			assert.Nil(t, client)
		})
	}
}

func TestNewClient_ConnectionFailure(t *testing.T) {
	t.Parallel()

	// Use an invalid host that won't resolve
	cfg := Config{
		Host:    "invalid-redis-host-that-does-not-exist",
		Port:    6379,
		Timeout: 1 * time.Second,
	}

	client, err := NewClient(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to redis")
	assert.Nil(t, client)
}

// Integration tests - require Redis to be running
// Skip if Redis is not available

func getTestClient(t *testing.T) *Client {
	t.Helper()

	cfg := Config{
		Host:     "localhost",
		Port:     6379,
		DB:       15, // Use a different DB for tests
		PoolSize: 5,
		MinIdle:  2,
		MaxRetry: 2,
		Timeout:  2 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	return client
}

func TestClient_Ping(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	err := client.Ping(ctx)

	require.NoError(t, err)
}

func TestClient_HealthCheck(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	err := client.HealthCheck(ctx)

	require.NoError(t, err)
}

func TestClient_SetAndGet(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	key := "test:key:1"
	value := "test-value"

	// Clean up after test
	defer func() {
		_, _ = client.Del(ctx, key) // Cleanup best effort
	}()

	// Set a value
	err := client.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Get the value
	got, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, got)
}

func TestClient_SetWithExpiration(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	key := "test:key:expiring"
	value := "test-value"
	expiration := 2 * time.Second

	// Clean up after test
	defer func() {
		_, _ = client.Del(ctx, key) // Cleanup best effort
	}()

	// Set a value with expiration
	err := client.Set(ctx, key, value, expiration)
	require.NoError(t, err)

	// Verify it exists
	got, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	// Check TTL
	ttl, err := client.TTL(ctx, key)
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, expiration)

	// Wait for expiration
	time.Sleep(expiration + 100*time.Millisecond)

	// Key should be expired
	_, err = client.Get(ctx, key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis: nil")
}

func TestClient_Del(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	key1 := "test:key:del:1"
	key2 := "test:key:del:2"

	// Set values
	err := client.Set(ctx, key1, "value1", 0)
	require.NoError(t, err)
	err = client.Set(ctx, key2, "value2", 0)
	require.NoError(t, err)

	// Delete keys
	deleted, err := client.Del(ctx, key1, key2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify keys are deleted
	exists, err := client.Exists(ctx, key1, key2)
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestClient_Exists(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	key := "test:key:exists"

	// Clean up after test
	defer func() {
		_, _ = client.Del(ctx, key) // Cleanup best effort
	}()

	// Key should not exist initially
	exists, err := client.Exists(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)

	// Set a value
	err = client.Set(ctx, key, "value", 0)
	require.NoError(t, err)

	// Key should exist now
	exists, err = client.Exists(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestClient_Expire(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()
	key := "test:key:expire"

	// Clean up after test
	defer func() {
		_, _ = client.Del(ctx, key) // Cleanup best effort
	}()

	// Set a value without expiration
	err := client.Set(ctx, key, "value", 0)
	require.NoError(t, err)

	// Check TTL (should be -1 for no expiration)
	ttl, err := client.TTL(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)

	// Set expiration
	expiration := 2 * time.Second
	err = client.Expire(ctx, key, expiration)
	require.NoError(t, err)

	// Check TTL again
	ttl, err = client.TTL(ctx, key)
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, expiration)
}

func TestClient_TTL(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func(key string)
		expectedTTL time.Duration
	}{
		{
			name: "key does not exist",
			setup: func(key string) {
				// Do nothing, key doesn't exist
			},
			expectedTTL: -2 * time.Second,
		},
		{
			name: "key exists without expiration",
			setup: func(key string) {
				_ = client.Set(ctx, key, "value", 0) // Setup best effort
			},
			expectedTTL: -1 * time.Second,
		},
		{
			name: "key exists with expiration",
			setup: func(key string) {
				_ = client.Set(ctx, key, "value", 5*time.Second) // Setup best effort
			},
			expectedTTL: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			key := "test:key:ttl:" + tt.name

			// Clean up after test
			defer func() {
				_, _ = client.Del(ctx, key) // Cleanup best effort
			}()

			// Setup
			tt.setup(key)

			// Get TTL
			ttl, err := client.TTL(ctx, key)
			require.NoError(t, err)

			if tt.expectedTTL < 0 {
				// go-redis may surface TTL sentinels as either seconds or raw duration units.
				// Accept both forms for "missing key" (-2) and "no expiration" (-1).
				if tt.expectedTTL == -2*time.Second {
					assert.Contains(t, []time.Duration{-2 * time.Second, -2 * time.Nanosecond}, ttl)
				} else if tt.expectedTTL == -1*time.Second {
					assert.Contains(t, []time.Duration{-1 * time.Second, -1 * time.Nanosecond}, ttl)
				} else {
					assert.Equal(t, tt.expectedTTL, ttl)
				}
			} else {
				// For positive TTLs, check it's within a reasonable range
				assert.Greater(t, ttl, time.Duration(0))
				assert.LessOrEqual(t, ttl, tt.expectedTTL)
			}
		})
	}
}

func TestClient_UnderlyingClient(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		_ = client.Close() // Cleanup best effort
	}()

	underlying := client.UnderlyingClient()
	assert.NotNil(t, underlying)

	// Verify we can use it directly
	ctx := context.Background()
	err := underlying.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestClient_Close(t *testing.T) {
	client := getTestClient(t)

	err := client.Close()
	require.NoError(t, err)

	// After closing, operations should fail
	ctx := context.Background()
	err = client.Ping(ctx)
	require.Error(t, err)
}

func TestConfigFromEnv(t *testing.T) {
	// Not parallel: mutates process-wide environment. Save and clear all
	// REDIS_* vars this test manipulates so it neither reads leaked shell
	// state nor leaks into other tests.
	keys := []string{"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_USE_TLS"}
	saved := make(map[string]string, len(keys))
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if v := saved[k]; v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})

	t.Run("unset falls back to defaults", func(t *testing.T) {
		for _, k := range keys {
			_ = os.Unsetenv(k)
		}

		cfg := ConfigFromEnv()

		// Identical to DefaultConfig() when nothing is set (localhost:6379).
		assert.Equal(t, DefaultConfig(), cfg)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, defaultRedisPort, cfg.Port)
		assert.Empty(t, cfg.Password)
		assert.False(t, cfg.UseTLS)
	})

	t.Run("host and port are honored", func(t *testing.T) {
		t.Setenv("REDIS_HOST", "redis")
		t.Setenv("REDIS_PORT", "6380")

		cfg := ConfigFromEnv()

		assert.Equal(t, "redis", cfg.Host)
		assert.Equal(t, 6380, cfg.Port)
	})

	t.Run("password and TLS are honored", func(t *testing.T) {
		t.Setenv("REDIS_PASSWORD", "s3cret")
		t.Setenv("REDIS_USE_TLS", "true")

		cfg := ConfigFromEnv()

		assert.Equal(t, "s3cret", cfg.Password)
		assert.True(t, cfg.UseTLS)
	})

	t.Run("invalid port falls back to the default", func(t *testing.T) {
		t.Setenv("REDIS_PORT", "not-a-number")

		cfg := ConfigFromEnv()

		assert.Equal(t, defaultRedisPort, cfg.Port)
	})
}
