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

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// TestTokenBlacklist_Add tests adding a token to the blacklist.
func TestTokenBlacklist_Add(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)
	_ = expiresAt

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

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token to blacklist
	blacklistedJTI := uuid.New().String()
	notBlacklistedJTI := uuid.New().String()
	_ = notBlacklistedJTI

	blacklistKey := "goimg:blacklist:" + blacklistedJTI
	err := suite.RedisClient.Set(ctx, blacklistKey, "1", 15*time.Minute).Err()
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

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange - add token with short TTL
	tokenJTI := uuid.New().String()
	blacklistKey := "goimg:blacklist:" + tokenJTI

	// Add with 1 second TTL
	err := blacklist.Add(ctx, tokenJTI, time.Now().Add(1*time.Second))
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

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange - create multiple tokens
	token1JTI := uuid.New().String()
	token2JTI := uuid.New().String()
	token3JTI := uuid.New().String()

	expiresAt := time.Now().Add(15 * time.Minute)
	_ = expiresAt

	// Act - blacklist all tokens
	err := blacklist.Add(ctx, token1JTI, expiresAt)
	require.NoError(t, err)
	err = blacklist.Add(ctx, token2JTI, expiresAt)
	require.NoError(t, err)
	err = blacklist.Add(ctx, token3JTI, expiresAt)
	require.NoError(t, err)

	// Assert - all tokens are blacklisted
	key1 := "goimg:blacklist:" + token1JTI
	key2 := "goimg:blacklist:" + token2JTI
	key3 := "goimg:blacklist:" + token3JTI

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

	expiredKey := "goimg:blacklist:" + expiredTokenJTI
	validKey := "goimg:blacklist:" + validTokenJTI

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

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)
	_ = expiresAt

	// Act - blacklist token concurrently from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := blacklist.Add(ctx, tokenJTI, expiresAt)
			if err != nil {
				t.Error(err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Assert - token should be blacklisted (only once)
	blacklistKey := "goimg:blacklist:" + tokenJTI
	exists, err := suite.RedisClient.Exists(ctx, blacklistKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

// TestTokenBlacklist_InvalidJTI tests handling invalid JTI format.
func TestTokenBlacklist_InvalidJTI(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	tests := []struct {
		name      string
		jti       string
		expectErr bool
	}{
		{"empty string", "", true},
		{"very long string", string(make([]byte, 10000)), false},
		{"special characters", "!@#$%^&*(){}[]|\\:;\"'<>,.?/~`", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := blacklist.Add(ctx, tt.jti, time.Now().Add(15*time.Minute))

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify in Redis
				key := "goimg:blacklist:" + tt.jti
				exists, err := suite.RedisClient.Exists(ctx, key).Result()
				require.NoError(t, err)
				assert.Equal(t, int64(1), exists)
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
	blacklistKey := "goimg:blacklist:" + tokenJTI
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

// TestTokenBlacklist_Remove tests explicitly removing a token from the blacklist.
func TestTokenBlacklist_Remove(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange
	tokenJTI := uuid.New().String()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Add token to blacklist
	err := blacklist.Add(ctx, tokenJTI, expiresAt)
	require.NoError(t, err)

	// Verify it exists
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenJTI)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Act
	err = blacklist.Remove(ctx, tokenJTI)
	require.NoError(t, err)

	// Assert
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, tokenJTI)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

// TestTokenBlacklist_Count tests counting blacklisted tokens.
func TestTokenBlacklist_Count(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange - Ensure blacklist is clean
	err := blacklist.Clear(ctx)
	require.NoError(t, err)

	// Add a known number of tokens
	count := 5
	expiresAt := time.Now().Add(15 * time.Minute)
	for i := 0; i < count; i++ {
		err := blacklist.Add(ctx, uuid.New().String(), expiresAt)
		require.NoError(t, err)
	}

	// Act
	actualCount, err := blacklist.Count(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(count), actualCount)
}

// TestTokenBlacklist_Clear tests clearing the entire blacklist.
func TestTokenBlacklist_Clear(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Create token blacklist instance
	blacklist := jwt.NewTokenBlacklist(suite.RedisClient)

	// Arrange - Add some tokens
	for i := 0; i < 3; i++ {
		err := blacklist.Add(ctx, uuid.New().String(), time.Now().Add(15*time.Minute))
		require.NoError(t, err)
	}

	// Verify count is > 0
	count, err := blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Positive(t, count)

	// Act
	err = blacklist.Clear(ctx)
	require.NoError(t, err)

	// Assert
	count, err = blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
