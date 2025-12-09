package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// blacklistKeyPrefix is the Redis key prefix for blacklisted tokens.
	blacklistKeyPrefix = "goimg:blacklist:"

	// redisScanCount is the count parameter for Redis SCAN operations.
	redisScanCount = 100
)

// TokenBlacklist manages revoked JWT tokens using Redis.
// Tokens are stored in the blacklist until their natural expiration.
type TokenBlacklist struct {
	redis *redis.Client
}

// NewTokenBlacklist creates a new token blacklist service.
func NewTokenBlacklist(redisClient *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		redis: redisClient,
	}
}

// Add adds a token to the blacklist with a TTL matching the token's remaining lifetime.
// The token will be automatically removed from the blacklist when it expires.
func (b *TokenBlacklist) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	if tokenID == "" {
		return fmt.Errorf("token id cannot be empty")
	}

	if expiresAt.IsZero() {
		return fmt.Errorf("expiration time cannot be zero")
	}

	// Calculate TTL based on remaining token lifetime
	now := time.Now().UTC()
	ttl := expiresAt.Sub(now)

	// If token is already expired, no need to blacklist
	if ttl <= 0 {
		return nil
	}

	key := blacklistKeyPrefix + tokenID

	// Store token in blacklist with TTL
	// Value is the expiration timestamp for debugging purposes
	err := b.redis.Set(ctx, key, expiresAt.Unix(), ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

// IsBlacklisted checks if a token is in the blacklist.
// Returns true if the token is blacklisted, false otherwise.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	if tokenID == "" {
		return false, fmt.Errorf("token id cannot be empty")
	}

	key := blacklistKeyPrefix + tokenID

	exists, err := b.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}

// Remove removes a token from the blacklist.
// This is typically not needed as tokens expire automatically,
// but can be used in special cases (e.g., administrative unban).
func (b *TokenBlacklist) Remove(ctx context.Context, tokenID string) error {
	if tokenID == "" {
		return fmt.Errorf("token id cannot be empty")
	}

	key := blacklistKeyPrefix + tokenID

	err := b.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}

	return nil
}

// Count returns the number of blacklisted tokens (for monitoring/debugging).
// Note: This uses SCAN which may be slow for large blacklists.
func (b *TokenBlacklist) Count(ctx context.Context) (int64, error) {
	var count int64
	var cursor uint64

	// Use SCAN to count keys with the blacklist prefix
	for {
		var keys []string
		var err error

		keys, cursor, err = b.redis.Scan(ctx, cursor, blacklistKeyPrefix+"*", redisScanCount).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to scan blacklist keys: %w", err)
		}

		count += int64(len(keys))

		// Stop when cursor returns to 0
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// Clear removes all blacklisted tokens (for testing or administrative purposes).
// WARNING: This uses SCAN and DEL which may be slow for large blacklists.
func (b *TokenBlacklist) Clear(ctx context.Context) error {
	var cursor uint64

	// Use SCAN to find and delete all blacklist keys
	for {
		var keys []string
		var err error

		keys, cursor, err = b.redis.Scan(ctx, cursor, blacklistKeyPrefix+"*", redisScanCount).Result()
		if err != nil {
			return fmt.Errorf("failed to scan blacklist keys: %w", err)
		}

		if len(keys) > 0 {
			err = b.redis.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to delete blacklist keys: %w", err)
			}
		}

		// Stop when cursor returns to 0
		if cursor == 0 {
			break
		}
	}

	return nil
}
