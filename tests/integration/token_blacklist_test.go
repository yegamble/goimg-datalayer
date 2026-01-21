//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rediscache "github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// TestTokenBlacklist_Add tests adding a token to the blacklist.
func TestTokenBlacklist_Add(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Act
	err := blacklist.Add(ctx, tokenJTI, expiresAt)

	// Assert
	require.NoError(t, err)

	// Verify token was added to Redis
	blacklistKey := "goimg:blacklist:" + tokenJTI
	exists, err := suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

// TestTokenBlacklist_IsBlacklisted tests checking if a token is blacklisted.
func TestTokenBlacklist_IsBlacklisted(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token to blacklist
	blacklistedJTI := uuid.New().String()
	notBlacklistedJTI := uuid.New().String()

	err := blacklist.Add(ctx, blacklistedJTI, time.Now().Add(15*time.Minute))
	require.NoError(t, err)

	// Act & Assert - check blacklisted token
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, blacklistedJTI)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Act & Assert - check non-blacklisted token
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, notBlacklistedJTI)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

// TestTokenBlacklist_Expiry tests that blacklisted tokens expire correctly.
func TestTokenBlacklist_Expiry(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token with short TTL
	tokenJTI := uuid.New().String()

	// Add with 1 second TTL
	err := blacklist.Add(ctx, tokenJTI, time.Now().Add(1*time.Second))
	require.NoError(t, err)

	// Verify it exists initially
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenJTI)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Act - wait for expiry
	time.Sleep(2 * time.Second)

	// Assert - token should be expired (no longer blacklisted)
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, tokenJTI)
	require.NoError(t, err)
	assert.False(t, isBlacklisted, "blacklisted token should have expired")
}

// TestTokenBlacklist_MultipleTokens tests blacklisting multiple tokens.
func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - create multiple tokens
	token1JTI := uuid.New().String()
	token2JTI := uuid.New().String()
	token3JTI := uuid.New().String()

	expiresAt := time.Now().Add(15 * time.Minute)

	// Act - blacklist all tokens
	err := blacklist.Add(ctx, token1JTI, expiresAt)
	require.NoError(t, err)
	err = blacklist.Add(ctx, token2JTI, expiresAt)
	require.NoError(t, err)
	err = blacklist.Add(ctx, token3JTI, expiresAt)
	require.NoError(t, err)

	// Assert - all tokens are blacklisted
	for _, tokenID := range []string{token1JTI, token2JTI, token3JTI} {
		isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
		require.NoError(t, err)
		assert.True(t, isBlacklisted)
	}
}

// TestTokenBlacklist_RemoveExpiredTokens tests cleanup of expired blacklist entries.
// This would be used by a background job to clean up expired entries.
func TestTokenBlacklist_RemoveExpiredTokens(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add tokens with different TTLs
	expiredTokenJTI := uuid.New().String()
	validTokenJTI := uuid.New().String()

	// Add expired token (1 second TTL)
	err := blacklist.Add(ctx, expiredTokenJTI, time.Now().Add(1*time.Second))
	require.NoError(t, err)

	// Add valid token (10 minute TTL)
	err = blacklist.Add(ctx, validTokenJTI, time.Now().Add(10*time.Minute))
	require.NoError(t, err)

	// Wait for expired token to expire
	time.Sleep(2 * time.Second)

	// Act - check expired token is gone
	isExpiredBlacklisted, err := blacklist.IsBlacklisted(ctx, expiredTokenJTI)
	require.NoError(t, err)

	// Assert - expired token should be auto-removed by Redis
	assert.False(t, isExpiredBlacklisted)

	// Valid token should still exist
	isValidBlacklisted, err := blacklist.IsBlacklisted(ctx, validTokenJTI)
	require.NoError(t, err)
	assert.True(t, isValidBlacklisted)
}

// TestTokenBlacklist_RaceCondition tests concurrent blacklist operations.
func TestTokenBlacklist_RaceCondition(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Act - blacklist token concurrently from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := blacklist.Add(ctx, tokenJTI, expiresAt)
			require.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Assert - token should be blacklisted (only once)
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenJTI)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)
}

// TestTokenBlacklist_InvalidJTI tests handling invalid JTI format.
func TestTokenBlacklist_InvalidJTI(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

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
			err := blacklist.Add(ctx, tt.jti, time.Now().Add(15*time.Minute))

			// Redis should handle all of these, but validate behavior
			if tt.jti == "" {
				// Empty JTI should be rejected
				require.Error(t, err)
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
	blacklist := rediscache.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	ttl := 10 * time.Minute

	err := blacklist.Add(ctx, tokenJTI, time.Now().Add(ttl))
	require.NoError(t, err)

	// Act
	blacklistKey := "goimg:blacklist:" + tokenJTI
	remainingTTL, err := suite.RedisClient.TTL(ctx, blacklistKey).Result()

	// Assert
	require.NoError(t, err)
	assert.Greater(t, remainingTTL, 9*time.Minute, "TTL should be close to 10 minutes")
	assert.LessOrEqual(t, remainingTTL, ttl, "TTL should not exceed original")
}
