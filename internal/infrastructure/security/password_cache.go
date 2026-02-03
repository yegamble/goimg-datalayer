package security

import (
	"container/list"
	"context"
	"fmt"
	"sync"
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
// It uses an LRU (Least Recently Used) eviction policy to maintain cache effectiveness
// while preventing unbounded memory growth. This implementation is primarily intended
// for testing or ephemeral environments.
type InMemoryPasswordCache struct {
	mu   sync.RWMutex
	data map[string]*list.Element
	// lruList maintains access order with most recently used at the back
	lruList *list.List
	// maxEntries limits the number of entries to prevent unbounded growth.
	maxEntries int
}

// lruEntry represents a cache entry with its key and value
type lruEntry struct {
	key   string
	pwned bool
}

// Default max entries for in-memory cache
const defaultMaxEntries = 1000

// NewInMemoryPasswordCache creates an in-memory password cache with LRU eviction.
func NewInMemoryPasswordCache() *InMemoryPasswordCache {
	return &InMemoryPasswordCache{
		data:       make(map[string]*list.Element),
		lruList:    list.New(),
		maxEntries: defaultMaxEntries,
	}
}

// Get retrieves a cached result from the in-memory map.
// On a cache hit, the entry is moved to the back of the LRU list (marked as recently used).
// Note: Uses write lock for all Get operations (not read lock) to maintain LRU order.
// This is acceptable for testing/ephemeral use, but production should use Redis.
func (c *InMemoryPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := prefix + ":" + suffix
	element, found := c.data[key]
	if !found {
		return false, false
	}

	// Move to back (most recently used)
	c.lruList.MoveToBack(element)
	entry := element.Value.(*lruEntry)
	return entry.pwned, true
}

// Set stores a result in the in-memory map (ignores TTL).
// Uses LRU eviction when the cache is full - removes the least recently used entry.
func (c *InMemoryPasswordCache) Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := prefix + ":" + suffix

	// If key exists, update it and move to back
	if element, exists := c.data[key]; exists {
		c.lruList.MoveToBack(element)
		entry := element.Value.(*lruEntry)
		entry.pwned = pwned
		return nil
	}

	// If at capacity, evict least recently used (front of list)
	if len(c.data) >= c.maxEntries {
		oldest := c.lruList.Front()
		if oldest != nil {
			oldEntry := oldest.Value.(*lruEntry)
			delete(c.data, oldEntry.key)
			c.lruList.Remove(oldest)
		}
	}

	// Add new entry to back (most recently used)
	entry := &lruEntry{key: key, pwned: pwned}
	element := c.lruList.PushBack(entry)
	c.data[key] = element

	return nil
}

// Delete removes an entry from the in-memory map and LRU list.
func (c *InMemoryPasswordCache) Delete(ctx context.Context, prefix, suffix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := prefix + ":" + suffix
	if element, exists := c.data[key]; exists {
		c.lruList.Remove(element)
		delete(c.data, key)
	}

	return nil
}
