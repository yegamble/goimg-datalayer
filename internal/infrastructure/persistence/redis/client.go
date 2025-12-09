// Package redis provides Redis client and session store implementations.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Default Redis configuration values.
	defaultRedisPort      = 6379
	defaultPoolSize       = 10
	defaultMinIdle        = 5
	defaultMaxRetry       = 3
	defaultTimeoutSec     = 5
	poolTimeoutMultiplier = 2
	connMaxIdleTimeMin    = 5
	connMaxLifetimeMin    = 30
)

// Config holds Redis connection configuration.
type Config struct {
	Host     string        // Redis server host (e.g., "localhost")
	Port     int           // Redis server port (e.g., 6379)
	Password string        // Optional password for authentication
	DB       int           // Database number (0-15)
	PoolSize int           // Maximum number of socket connections
	MinIdle  int           // Minimum number of idle connections
	MaxRetry int           // Maximum number of retries before giving up
	Timeout  time.Duration // Connection timeout
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     defaultRedisPort,
		Password: "",
		DB:       0,
		PoolSize: defaultPoolSize,
		MinIdle:  defaultMinIdle,
		MaxRetry: defaultMaxRetry,
		Timeout:  defaultTimeoutSec * time.Second,
	}
}

// Client wraps redis.Client with additional methods for health checks.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client with the given configuration.
// Returns an error if the client cannot be created or initial health check fails.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("redis host cannot be empty")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid redis port: %d", cfg.Port)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdle,
		MaxRetries:   cfg.MaxRetry,
		DialTimeout:  cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,

		// Connection pool settings for optimal performance
		PoolTimeout:     cfg.Timeout * poolTimeoutMultiplier,
		ConnMaxIdleTime: connMaxIdleTimeMin * time.Minute,
		ConnMaxLifetime: connMaxLifetimeMin * time.Minute,
	})

	client := &Client{rdb: rdb}

	// Verify connection with a ping
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", addr, err)
	}

	return client, nil
}

// Ping checks if the Redis server is reachable.
// Returns an error if the server does not respond within the context deadline.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// HealthCheck performs a comprehensive health check on the Redis connection.
// Returns an error if any check fails.
func (c *Client) HealthCheck(ctx context.Context) error {
	// Check if we can ping the server
	if err := c.Ping(ctx); err != nil {
		return err
	}

	// Check pool stats for connection issues
	stats := c.rdb.PoolStats()
	if stats.TotalConns == 0 {
		return fmt.Errorf("redis: no connections in pool")
	}

	return nil
}

// Close closes the Redis client and releases all resources.
func (c *Client) Close() error {
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	return nil
}

// UnderlyingClient returns the underlying redis.Client for direct access.
// Use this when you need to perform operations not wrapped by this client.
func (c *Client) UnderlyingClient() *redis.Client {
	return c.rdb
}

// Set stores a key-value pair with an optional expiration.
// If expiration is 0, the key will not expire.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("redis set failed for key %s: %w", key, err)
	}
	return nil
}

// Get retrieves the value for the given key.
// Returns redis.Nil if the key does not exist.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get failed for key %s: %w", key, err)
	}
	return val, nil
}

// Del deletes one or more keys.
// Returns the number of keys that were removed.
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	deleted, err := c.rdb.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del failed: %w", err)
	}
	return deleted, nil
}

// Exists checks if one or more keys exist.
// Returns the number of keys that exist.
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	count, err := c.rdb.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis exists failed: %w", err)
	}
	return count, nil
}

// Expire sets a timeout on a key.
// After the timeout has expired, the key will automatically be deleted.
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.rdb.Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("redis expire failed for key %s: %w", key, err)
	}
	return nil
}

// TTL returns the remaining time to live of a key that has a timeout.
// Returns -2 if the key does not exist, -1 if the key exists but has no associated expire.
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ttl failed for key %s: %w", key, err)
	}
	return ttl, nil
}
