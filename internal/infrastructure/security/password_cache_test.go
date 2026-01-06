package security

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestInMemoryPasswordCache tests the in-memory cache implementation (for testing purposes).
func TestInMemoryPasswordCache(t *testing.T) {
	t.Parallel()

	t.Run("get returns false when cache is empty", func(t *testing.T) {
		t.Parallel()

		cache := NewInMemoryPasswordCache()

		pwned, found := cache.Get(context.Background(), "ABCDE", "12345")

		assert.False(t, found)
		assert.False(t, pwned)
	})

	t.Run("set and get stores and retrieves values", func(t *testing.T) {
		t.Parallel()

		cache := NewInMemoryPasswordCache()

		// Set a pwned password
		err := cache.Set(context.Background(), "ABCDE", "12345", true, 0)
		require.NoError(t, err)

		// Retrieve it
		pwned, found := cache.Get(context.Background(), "ABCDE", "12345")
		assert.True(t, found)
		assert.True(t, pwned)
	})

	t.Run("set and get with not pwned password", func(t *testing.T) {
		t.Parallel()

		cache := NewInMemoryPasswordCache()

		// Set a clean password
		err := cache.Set(context.Background(), "FGHIJ", "67890", false, 0)
		require.NoError(t, err)

		// Retrieve it
		pwned, found := cache.Get(context.Background(), "FGHIJ", "67890")
		assert.True(t, found)
		assert.False(t, pwned)
	})

	t.Run("delete removes entry", func(t *testing.T) {
		t.Parallel()

		cache := NewInMemoryPasswordCache()

		// Set and verify
		err := cache.Set(context.Background(), "KLMNO", "11111", true, 0)
		require.NoError(t, err)

		pwned, found := cache.Get(context.Background(), "KLMNO", "11111")
		assert.True(t, found)
		assert.True(t, pwned)

		// Delete
		err = cache.Delete(context.Background(), "KLMNO", "11111")
		require.NoError(t, err)

		// Verify deleted
		pwned, found = cache.Get(context.Background(), "KLMNO", "11111")
		assert.False(t, found)
		assert.False(t, pwned)
	})

	t.Run("different prefix-suffix combinations are distinct", func(t *testing.T) {
		t.Parallel()

		cache := NewInMemoryPasswordCache()

		// Set multiple entries
		err := cache.Set(context.Background(), "AAA", "111", true, 0)
		require.NoError(t, err)

		err = cache.Set(context.Background(), "AAA", "222", false, 0)
		require.NoError(t, err)

		err = cache.Set(context.Background(), "BBB", "111", false, 0)
		require.NoError(t, err)

		// Verify each is distinct
		pwned1, found1 := cache.Get(context.Background(), "AAA", "111")
		assert.True(t, found1)
		assert.True(t, pwned1)

		pwned2, found2 := cache.Get(context.Background(), "AAA", "222")
		assert.True(t, found2)
		assert.False(t, pwned2)

		pwned3, found3 := cache.Get(context.Background(), "BBB", "111")
		assert.True(t, found3)
		assert.False(t, pwned3)
	})
}

// TestRedisPasswordCache tests the Redis-backed cache implementation.
// Uses testcontainers to spin up a real Redis instance for integration testing.
func TestRedisPasswordCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	ctx := context.Background()

	// Start Redis container
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = redisContainer.Terminate(ctx)
	})

	// Get Redis connection details
	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	// Verify connection
	err = redisClient.Ping(ctx).Err()
	require.NoError(t, err)

	t.Run("get returns false when key does not exist", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		pwned, found := cache.Get(ctx, "XXXXX", "99999")

		assert.False(t, found)
		assert.False(t, pwned)
	})

	t.Run("set and get stores and retrieves pwned password", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		prefix := "ABCDE"
		suffix := "12345"

		// Set pwned password with 1 hour TTL
		err := cache.Set(ctx, prefix, suffix, true, time.Hour)
		require.NoError(t, err)

		// Retrieve it
		pwned, found := cache.Get(ctx, prefix, suffix)
		assert.True(t, found)
		assert.True(t, pwned)

		// Verify Redis key format
		key := "test:hibp:ABCDE:12345"
		val, err := redisClient.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "1", val)
	})

	t.Run("set and get stores and retrieves clean password", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		prefix := "FGHIJ"
		suffix := "67890"

		// Set clean password with 24 hour TTL
		err := cache.Set(ctx, prefix, suffix, false, 24*time.Hour)
		require.NoError(t, err)

		// Retrieve it
		pwned, found := cache.Get(ctx, prefix, suffix)
		assert.True(t, found)
		assert.False(t, pwned)

		// Verify Redis value
		key := "test:hibp:FGHIJ:67890"
		val, err := redisClient.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "0", val)
	})

	t.Run("set with TTL=0 stores indefinitely", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		prefix := "KLMNO"
		suffix := "11111"

		// Set with no expiration (TTL=0)
		err := cache.Set(ctx, prefix, suffix, true, 0)
		require.NoError(t, err)

		// Verify no TTL is set
		key := "test:hibp:KLMNO:11111"
		ttl := redisClient.TTL(ctx, key).Val()
		assert.Equal(t, time.Duration(-1), ttl) // -1 means no expiration
	})

	t.Run("TTL causes entry to expire", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		prefix := "PQRST"
		suffix := "22222"

		// Set with very short TTL
		err := cache.Set(ctx, prefix, suffix, false, 100*time.Millisecond)
		require.NoError(t, err)

		// Verify it exists
		pwned, found := cache.Get(ctx, prefix, suffix)
		assert.True(t, found)
		assert.False(t, pwned)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Verify it's gone
		pwned, found = cache.Get(ctx, prefix, suffix)
		assert.False(t, found)
	})

	t.Run("delete removes entry", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		prefix := "UVWXY"
		suffix := "33333"

		// Set entry
		err := cache.Set(ctx, prefix, suffix, true, time.Hour)
		require.NoError(t, err)

		// Verify exists
		pwned, found := cache.Get(ctx, prefix, suffix)
		assert.True(t, found)
		assert.True(t, pwned)

		// Delete
		err = cache.Delete(ctx, prefix, suffix)
		require.NoError(t, err)

		// Verify deleted
		pwned, found = cache.Get(ctx, prefix, suffix)
		assert.False(t, found)
	})

	t.Run("different prefixes are isolated", func(t *testing.T) {
		cache1 := NewRedisPasswordCache(redisClient, "app1:hibp")
		cache2 := NewRedisPasswordCache(redisClient, "app2:hibp")

		prefix := "ZZZZZ"
		suffix := "44444"

		// Set in cache1
		err := cache1.Set(ctx, prefix, suffix, true, time.Hour)
		require.NoError(t, err)

		// Verify exists in cache1
		pwned, found := cache1.Get(ctx, prefix, suffix)
		assert.True(t, found)
		assert.True(t, pwned)

		// Verify NOT in cache2 (different prefix)
		pwned, found = cache2.Get(ctx, prefix, suffix)
		assert.False(t, found)
	})

	t.Run("handles Redis errors gracefully", func(t *testing.T) {
		// Create cache with closed client
		closedClient := redis.NewClient(&redis.Options{
			Addr: "localhost:99999", // Invalid address
		})
		cache := NewRedisPasswordCache(closedClient, "test:hibp")

		// Get should return false, false (cache miss)
		pwned, found := cache.Get(ctx, "ERROR", "TEST")
		assert.False(t, found)
		assert.False(t, pwned)

		// Set should return error
		err := cache.Set(ctx, "ERROR", "TEST", true, time.Hour)
		assert.Error(t, err)

		// Delete should return error
		err = cache.Delete(ctx, "ERROR", "TEST")
		assert.Error(t, err)
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		cache := NewRedisPasswordCache(redisClient, "test:hibp")

		// Concurrent writes
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				prefix := "CONCUR"
				suffix := string(rune('0' + id))
				_ = cache.Set(ctx, prefix, suffix, id%2 == 0, time.Hour)
				done <- true
			}(i)
		}

		// Wait for all writes
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all entries
		for i := 0; i < 10; i++ {
			prefix := "CONCUR"
			suffix := string(rune('0' + i))
			pwned, found := cache.Get(ctx, prefix, suffix)
			assert.True(t, found)
			assert.Equal(t, i%2 == 0, pwned)
		}
	})
}

// BenchmarkInMemoryCache benchmarks the in-memory cache performance.
func BenchmarkInMemoryCache(b *testing.B) {
	cache := NewInMemoryPasswordCache()
	ctx := context.Background()

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		prefix := string(rune('A' + (i % 26)))
		suffix := string(rune('0' + (i % 10)))
		_ = cache.Set(ctx, prefix, suffix, i%2 == 0, 0)
	}

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			prefix := string(rune('A' + (i % 26)))
			suffix := string(rune('0' + (i % 10)))
			_, _ = cache.Get(ctx, prefix, suffix)
		}
	})

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			prefix := string(rune('A' + (i % 26)))
			suffix := string(rune('0' + (i % 10)))
			_ = cache.Set(ctx, prefix, suffix, i%2 == 0, 0)
		}
	})
}
