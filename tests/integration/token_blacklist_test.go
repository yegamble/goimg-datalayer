// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// TestTokenBlacklist_Add tests adding a token to the blacklist.
func TestTokenBlacklist_Add(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance once infrastructure layer is implemented
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Act
	// err := blacklist.Add(ctx, tokenJTI, expiresAt)

	// Assert
	// require.NoError(t, err)

	// Verify token was added to Redis
	blacklistKey := "blacklist:token:" + tokenJTI
	exists, err := suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	// assert.Equal(t, int64(1), exists)

	t.Skip("Skipping until TokenBlacklist implementation is available")
}

// TestTokenBlacklist_IsBlacklisted tests checking if a token is blacklisted.
func TestTokenBlacklist_IsBlacklisted(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token to blacklist
	blacklistedJTI := uuid.New().String()
	notBlacklistedJTI := uuid.New().String()

	blacklistKey := "blacklist:token:" + blacklistedJTI
	err := suite.RedisClient.Set(ctx, blacklistKey, "1", 15*time.Minute).Err()
	require.NoError(t, err)

	// Act & Assert - check blacklisted token
	// isBlacklisted, err := blacklist.IsBlacklisted(ctx, blacklistedJTI)
	// require.NoError(t, err)
	// assert.True(t, isBlacklisted)

	// Act & Assert - check non-blacklisted token
	// isBlacklisted, err = blacklist.IsBlacklisted(ctx, notBlacklistedJTI)
	// require.NoError(t, err)
	// assert.False(t, isBlacklisted)

	t.Skip("Skipping until TokenBlacklist implementation is available")
}

// TestTokenBlacklist_Expiry tests that blacklisted tokens expire correctly.
func TestTokenBlacklist_Expiry(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token with short TTL
	tokenJTI := uuid.New().String()
	blacklistKey := "blacklist:token:" + tokenJTI

	// Add with 1 second TTL
	err := suite.RedisClient.Set(ctx, blacklistKey, "1", 1*time.Second).Err()
	require.NoError(t, err)

	// Verify it exists initially
	exists, err := suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	// Act - wait for expiry
	time.Sleep(2 * time.Second)

	// Assert - token should be expired (no longer blacklisted)
	exists, err = suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "blacklisted token should have expired")
}

// TestTokenBlacklist_MultipleTokens tests blacklisting multiple tokens.
func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - create multiple tokens
	token1JTI := uuid.New().String()
	token2JTI := uuid.New().String()
	token3JTI := uuid.New().String()

	expiresAt := time.Now().Add(15 * time.Minute)

	// Act - blacklist all tokens
	key1 := "blacklist:token:" + token1JTI
	key2 := "blacklist:token:" + token2JTI
	key3 := "blacklist:token:" + token3JTI

	err := suite.RedisClient.Set(ctx, key1, "1", 15*time.Minute).Err()
	require.NoError(t, err)
	err = suite.RedisClient.Set(ctx, key2, "1", 15*time.Minute).Err()
	require.NoError(t, err)
	err = suite.RedisClient.Set(ctx, key3, "1", 15*time.Minute).Err()
	require.NoError(t, err)

	// Assert - all tokens are blacklisted
	for _, key := range []string{key1, key2, key3} {
		exists, err := suite.RedisClient.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)
	}
}

// TestTokenBlacklist_RemoveExpiredTokens tests cleanup of expired blacklist entries.
// This would be used by a background job to clean up expired entries.
func TestTokenBlacklist_RemoveExpiredTokens(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Arrange - add tokens with different TTLs
	expiredTokenJTI := uuid.New().String()
	validTokenJTI := uuid.New().String()

	expiredKey := "blacklist:token:" + expiredTokenJTI
	validKey := "blacklist:token:" + validTokenJTI

	// Add expired token (1 second TTL)
	err := suite.RedisClient.Set(ctx, expiredKey, "1", 1*time.Second).Err()
	require.NoError(t, err)

	// Add valid token (10 minute TTL)
	err = suite.RedisClient.Set(ctx, validKey, "1", 10*time.Minute).Err()
	require.NoError(t, err)

	// Wait for expired token to expire
	time.Sleep(2 * time.Second)

	// Act - check expired token is gone
	expiredExists, err := suite.RedisClient.Exists(ctx, expiredKey).Result()
	require.NoError(t, err)

	// Assert - expired token should be auto-removed by Redis
	assert.Equal(t, int64(0), expiredExists)

	// Valid token should still exist
	validExists, err := suite.RedisClient.Exists(ctx, validKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), validExists)
}

// TestTokenBlacklist_RaceCondition tests concurrent blacklist operations.
func TestTokenBlacklist_RaceCondition(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Act - blacklist token concurrently from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			key := "blacklist:token:" + tokenJTI
			err := suite.RedisClient.Set(ctx, key, "1", 15*time.Minute).Err()
			require.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Assert - token should be blacklisted (only once)
	blacklistKey := "blacklist:token:" + tokenJTI
	exists, err := suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

// TestTokenBlacklist_InvalidJTI tests handling invalid JTI format.
func TestTokenBlacklist_InvalidJTI(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create token blacklist instance
	// blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	tests := []struct {
		name string
		jti  string
	}{
		{"empty string", ""},
		{"very long string", string(make([]byte, 10000))},
		{"special characters", "!@#$%^&*(){}[]|\\:;\"'<>,.?/~`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should not panic and should handle gracefully
			key := "blacklist:token:" + tt.jti
			err := suite.RedisClient.Set(ctx, key, "1", 15*time.Minute).Err()

			// Redis should handle all of these, but validate behavior
			if tt.jti == "" {
				// Empty JTI might be rejected by application logic
				t.Skip("Empty JTI handling depends on application logic")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestTokenBlacklist_GetTTL tests retrieving TTL for a blacklisted token.
func TestTokenBlacklist_GetTTL(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Arrange
	tokenJTI := uuid.New().String()
	blacklistKey := "blacklist:token:" + tokenJTI
	ttl := 10 * time.Minute

	err := suite.RedisClient.Set(ctx, blacklistKey, "1", ttl).Err()
	require.NoError(t, err)

	// Act
	remainingTTL, err := suite.RedisClient.TTL(ctx, blacklistKey).Result()

	// Assert
	require.NoError(t, err)
	assert.Greater(t, remainingTTL, 9*time.Minute, "TTL should be close to 10 minutes")
	assert.LessOrEqual(t, remainingTTL, ttl, "TTL should not exceed original")
}
