package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniredis creates a miniredis instance for testing without a real Redis server.
func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *Client) {
	t.Helper()

	mr := miniredis.RunT(t)

	cfg := Config{
		Host:     mr.Host(),
		Port:     mr.Server().Addr().Port,
		Password: "",
		DB:       0,
		PoolSize: 10,
		MinIdle:  5,
		MaxRetry: 3,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	return mr, client
}

func TestClient_Set_Success(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		key        string
		value      string
		expiration time.Duration
	}{
		{
			name:       "set without expiration",
			key:        "test:key1",
			value:      "value1",
			expiration: 0,
		},
		{
			name:       "set with expiration",
			key:        "test:key2",
			value:      "value2",
			expiration: 1 * time.Hour,
		},
		{
			name:       "set with short expiration",
			key:        "test:key3",
			value:      "value3",
			expiration: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := client.Set(ctx, tt.key, tt.value, tt.expiration)
			require.NoError(t, err)

			// Verify the value was set
			got, err := client.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestClient_Get_Success(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test:get:key"
	value := "test-value"

	// Set value directly in miniredis
	mr.Set(key, value)

	got, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, got)
}

func TestClient_Get_NotFound(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	got, err := client.Get(ctx, "nonexistent-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis get failed")
	assert.Empty(t, got)
}

func TestClient_Del_Success(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	// Set up test data
	keys := []string{"test:del:1", "test:del:2", "test:del:3"}
	for _, key := range keys {
		mr.Set(key, "value")
	}

	// Delete keys
	deleted, err := client.Del(ctx, keys...)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify keys are deleted
	for _, key := range keys {
		exists := mr.Exists(key)
		assert.False(t, exists)
	}
}

func TestClient_Del_NonexistentKeys(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	deleted, err := client.Del(ctx, "nonexistent1", "nonexistent2")
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

//nolint:tparallel // Subtests share miniredis instance and cannot run in parallel
func TestClient_Exists_Success(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		keys     []string
		setup    func()
		expected int64
	}{
		{
			name: "no keys exist",
			keys: []string{"key1", "key2"},
			setup: func() {
				// No setup
			},
			expected: 0,
		},
		{
			name: "one key exists",
			keys: []string{"key1", "key2"},
			setup: func() {
				mr.Set("key1", "value")
			},
			expected: 1,
		},
		{
			name: "all keys exist",
			keys: []string{"key1", "key2", "key3"},
			setup: func() {
				mr.Set("key1", "value1")
				mr.Set("key2", "value2")
				mr.Set("key3", "value3")
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() here because subtests share the same miniredis instance
			mr.FlushAll()
			tt.setup()

			count, err := client.Exists(ctx, tt.keys...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestClient_Expire_Success(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test:expire:key"
	mr.Set(key, "value")

	// Set expiration
	expiration := 5 * time.Minute
	err := client.Expire(ctx, key, expiration)
	require.NoError(t, err)

	// Check TTL is set
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, expiration)
}

//nolint:tparallel // Subtests share miniredis instance and cannot run in parallel
func TestClient_TTL_Success(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func(key string)
		expectedTTL time.Duration
	}{
		{
			name: "key does not exist",
			setup: func(key string) {
				// No setup
			},
			expectedTTL: -2 * time.Second,
		},
		{
			name: "key exists without expiration",
			setup: func(key string) {
				mr.Set(key, "value")
			},
			expectedTTL: -1 * time.Second,
		},
		{
			name: "key exists with expiration",
			setup: func(key string) {
				mr.Set(key, "value")
				mr.SetTTL(key, 10*time.Minute)
			},
			expectedTTL: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cannot use t.Parallel() here because subtests share the same miniredis instance
			key := "test:ttl:" + tt.name
			mr.FlushAll()
			tt.setup(key)

			ttl, err := client.TTL(ctx, key)
			require.NoError(t, err)

			if tt.expectedTTL < 0 {
				// Miniredis returns -1ns or -2ns instead of -1s or -2s
				// Just check that TTL is negative
				assert.True(t, ttl < 0, "TTL should be negative, got: %v", ttl)
			} else {
				// For positive TTLs, check within range
				assert.Greater(t, ttl, time.Duration(0))
				assert.LessOrEqual(t, ttl, tt.expectedTTL)
			}
		})
	}
}

func TestClient_Ping_Success(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	err := client.Ping(ctx)
	require.NoError(t, err)
}

func TestClient_Ping_ContextCanceled(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Ping(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis ping failed")
}

func TestClient_HealthCheck_Success(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	err := client.HealthCheck(ctx)
	require.NoError(t, err)
}

func TestClient_HealthCheck_PingFailure(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)

	// Close the client to make ping fail
	err := client.Close()
	require.NoError(t, err)

	ctx := context.Background()
	err = client.HealthCheck(ctx)
	require.Error(t, err)
}

func TestClient_UnderlyingClient_Success(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)

	underlying := client.UnderlyingClient()
	require.NotNil(t, underlying)

	// Verify it's a valid redis.Client
	ctx := context.Background()
	err := underlying.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestClient_Close_Success(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := Config{
		Host:     mr.Host(),
		Port:     mr.Server().Addr().Port,
		Password: "",
		DB:       0,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	// Close should succeed
	err = client.Close()
	require.NoError(t, err)

	// Operations after close should fail
	ctx := context.Background()
	err = client.Ping(ctx)
	require.Error(t, err)
}

func TestClient_SetGetDel_Integration(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test:integration:key"
	value := "test-value"

	// Set
	err := client.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Get
	got, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	// Exists
	exists, err := client.Exists(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	// Del
	deleted, err := client.Del(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify deleted
	exists, err = client.Exists(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestClient_SetWithExpiration_Verify(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test:expiring"
	value := "value"
	expiration := 10 * time.Minute

	err := client.Set(ctx, key, value, expiration)
	require.NoError(t, err)

	// Check TTL is set
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, expiration)
}

func TestClient_MultipleOperations(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)
	ctx := context.Background()

	// Set multiple keys
	keys := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range keys {
		err := client.Set(ctx, k, v, 0)
		require.NoError(t, err)
	}

	// Get all keys
	for k, expectedV := range keys {
		v, err := client.Get(ctx, k)
		require.NoError(t, err)
		assert.Equal(t, expectedV, v)
	}

	// Check all exist
	var keySlice []string
	for k := range keys {
		keySlice = append(keySlice, k)
	}
	count, err := client.Exists(ctx, keySlice...)
	require.NoError(t, err)
	assert.Equal(t, int64(len(keys)), count)

	// Delete all
	deleted, err := client.Del(ctx, keySlice...)
	require.NoError(t, err)
	assert.Equal(t, int64(len(keys)), deleted)

	// Verify all deleted
	count, err = client.Exists(ctx, keySlice...)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNewClient_WithMiniredis(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := Config{
		Host:     mr.Host(),
		Port:     mr.Server().Addr().Port,
		Password: "",
		DB:       0,
		PoolSize: 10,
		MinIdle:  5,
		MaxRetry: 3,
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	defer func() {
		_ = client.Close()
	}()

	// Verify client works
	ctx := context.Background()
	err = client.Ping(ctx)
	require.NoError(t, err)
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()

	_, client := setupMiniredis(t)

	tests := []struct {
		name      string
		operation func(ctx context.Context) error
	}{
		{
			name: "Set with canceled context",
			operation: func(ctx context.Context) error {
				return client.Set(ctx, "key", "value", 0)
			},
		},
		{
			name: "Get with canceled context",
			operation: func(ctx context.Context) error {
				_, err := client.Get(ctx, "key")
				return err
			},
		},
		{
			name: "Del with canceled context",
			operation: func(ctx context.Context) error {
				_, err := client.Del(ctx, "key")
				return err
			},
		},
		{
			name: "Exists with canceled context",
			operation: func(ctx context.Context) error {
				_, err := client.Exists(ctx, "key")
				return err
			},
		},
		{
			name: "Expire with canceled context",
			operation: func(ctx context.Context) error {
				return client.Expire(ctx, "key", time.Minute)
			},
		},
		{
			name: "TTL with canceled context",
			operation: func(ctx context.Context) error {
				_, err := client.TTL(ctx, "key")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := tt.operation(ctx)
			require.Error(t, err)
		})
	}
}
