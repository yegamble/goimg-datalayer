package security

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PasswordCache defines the interface for caching HIBP password check results.
// Implementations store whether a password hash has been found in breach databases
// to reduce API calls and improve response time.
type PasswordCache interface {
	// Get retrieves the cached result for a password hash.
	// Returns (pwned, found) where:
	//   - pwned: true if password was found in breaches, false if not
	//   - found: true if cache entry exists, false if cache miss
	Get(ctx context.Context, prefix, suffix string) (bool, bool)

	// Set stores the result of a password check in the cache.
	// Parameters:
	//   - prefix: First 5 characters of SHA-1 hash (for k-anonymity)
	//   - suffix: Remaining characters of SHA-1 hash
	//   - pwned: true if password was found in breaches
	//   - ttl: Time-to-live for cache entry (0 = no expiration)
	Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error

	// Delete removes a cache entry (primarily for testing).
	Delete(ctx context.Context, prefix, suffix string) error
}

// RedisPasswordCache implements PasswordCache using Redis as the backing store.
// Key pattern: goimg:hibp:{prefix}:{suffix}
// Value: "0" (not pwned) or "1" (pwned)
type RedisPasswordCache struct {
	client *redis.Client
	prefix string
}

// NewRedisPasswordCache creates a new Redis-backed password cache.
// The prefix parameter is prepended to all Redis keys (e.g., "goimg:hibp").
func NewRedisPasswordCache(client *redis.Client, keyPrefix string) *RedisPasswordCache {
	return &RedisPasswordCache{
		client: client,
		prefix: keyPrefix,
	}
}

// Get retrieves a cached password check result from Redis.
// Returns (pwned, found) where found=false indicates a cache miss.
func (c *RedisPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
	key := c.buildKey(prefix, suffix)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Cache miss - not an error
			return false, false
		}
		// Redis error - treat as cache miss to fail gracefully
		return false, false
	}

	// Value is "1" for pwned, "0" for not pwned
	pwned := val == "1"
	return pwned, true
}

// Set stores a password check result in Redis with the specified TTL.
// A TTL of 0 means the entry never expires (used for pwned passwords).
func (c *RedisPasswordCache) Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error {
	key := c.buildKey(prefix, suffix)

	// Encode boolean as "1" (pwned) or "0" (not pwned)
	value := "0"
	if pwned {
		value = "1"
	}

	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

// Delete removes a cache entry from Redis.
// Primarily used for testing; production code rarely needs to invalidate cache.
func (c *RedisPasswordCache) Delete(ctx context.Context, prefix, suffix string) error {
	key := c.buildKey(prefix, suffix)

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis del: %w", err)
	}

	return nil
}

// buildKey constructs the Redis key using the configured prefix and hash components.
// Format: {prefix}:{hash_prefix}:{hash_suffix}
// Example: goimg:hibp:2AC9A:6746ACA543AF8DFF39894CBC7B2D299FCAE
func (c *RedisPasswordCache) buildKey(hashPrefix, hashSuffix string) string {
	return fmt.Sprintf("%s:%s:%s", c.prefix, hashPrefix, hashSuffix)
}

// InMemoryPasswordCache is a simple in-memory cache for testing purposes.
// DO NOT use in production - no TTL support, unbounded memory growth.
//
// TODO(audit-2026-02-03): CRITICAL - This implementation has a race condition.
// The data map is accessed without mutex protection, causing data races in
// concurrent access scenarios. Additionally, the map grows unboundedly with
// no eviction policy. Either:
// 1. Add sync.RWMutex protection and LRU eviction
// 2. Mark this as explicitly test-only with build tags
// See: claude/audit_report_2026-02-03.md for full details.
type InMemoryPasswordCache struct {
	data map[string]bool
}

// NewInMemoryPasswordCache creates an in-memory password cache for testing.
func NewInMemoryPasswordCache() *InMemoryPasswordCache {
	return &InMemoryPasswordCache{
		data: make(map[string]bool),
	}
}

// Get retrieves a cached result from the in-memory map.
func (c *InMemoryPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
	key := prefix + ":" + suffix
	pwned, found := c.data[key]
	return pwned, found
}

// Set stores a result in the in-memory map (ignores TTL).
func (c *InMemoryPasswordCache) Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error {
	key := prefix + ":" + suffix
	c.data[key] = pwned
	return nil
}

// Delete removes an entry from the in-memory map.
func (c *InMemoryPasswordCache) Delete(ctx context.Context, prefix, suffix string) error {
	key := prefix + ":" + suffix
	delete(c.data, key)
	return nil
}
