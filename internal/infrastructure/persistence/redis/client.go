// Package redis provides Redis client and session store implementations.
package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
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
	UseTLS   bool          // Use TLS for connection
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
		UseTLS:   false,
	}
}

// ConfigFromEnv creates a Config populated from environment variables, falling
// back to DefaultConfig() values for any variable that is unset. This lets the
// API dial a Redis service by hostname (e.g. a "redis" container in a compose
// stack) instead of always assuming localhost.
//
// Environment variables:
//   - REDIS_HOST (default: localhost)
//   - REDIS_PORT (default: 6379)
//   - REDIS_PASSWORD (default: empty)
//   - REDIS_USE_TLS (default: false; set to "true" to enable TLS)
//
// When none of these are set the returned Config is identical to
// DefaultConfig() (localhost:6379), preserving existing local-dev behavior.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if v := getEnv("REDIS_HOST", ""); v != "" {
		cfg.Host = v
	}
	if v := getEnvInt("REDIS_PORT", 0); v != 0 {
		cfg.Port = v
	}
	if v := getEnv("REDIS_PASSWORD", ""); v != "" {
		cfg.Password = v
	}
	if getEnv("REDIS_USE_TLS", "") == "true" {
		cfg.UseTLS = true
	}
	return cfg
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

	opts := &redis.Options{
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
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)

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

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value.
// Logs a warning if the environment variable contains an invalid integer value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			log.Warn().
				Str("key", key).
				Str("value", value).
				Int("default", defaultValue).
				Msg("invalid integer value for environment variable, using default")
			return defaultValue
		}
		return intVal
	}
	return defaultValue
}
