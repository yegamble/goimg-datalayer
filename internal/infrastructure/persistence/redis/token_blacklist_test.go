package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedisForBlacklist creates a miniredis instance for blacklist testing.
func setupTestRedisForBlacklist(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0,
	})

	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})

	return mr, client
}

func TestNewTokenBlacklist(t *testing.T) {
	t.Parallel()

	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)

	assert.NotNil(t, blacklist)
	assert.Equal(t, client, blacklist.redis)
}

func TestTokenBlacklist_Add(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	tokenID := "test-token-1"
	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// Clean up after test
	defer func() {
		_ = blacklist.Remove(ctx, tokenID)
	}()

	// Add token to blacklist
	err := blacklist.Add(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// Verify token is blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestTokenBlacklist_Add_EmptyTokenID(t *testing.T) {
	t.Parallel()

	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	err := blacklist.Add(ctx, "", time.Now().Add(5*time.Minute))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token id cannot be empty")
}

func TestTokenBlacklist_Add_ZeroExpiration(t *testing.T) {
	t.Parallel()

	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	err := blacklist.Add(ctx, "token-id", time.Time{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiration time cannot be zero")
}

func TestTokenBlacklist_Add_AlreadyExpired(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	tokenID := "expired-token"
	expiresAt := time.Now().UTC().Add(-1 * time.Hour) // Already expired

	// Should not error, but should not add to blacklist
	err := blacklist.Add(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// Verify token is not blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestTokenBlacklist_Add_WithTTL(t *testing.T) {
	mr, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	tokenID := "ttl-token"
	expiresAt := time.Now().UTC().Add(2 * time.Second)

	// Clean up after test
	defer func() {
		_ = blacklist.Remove(ctx, tokenID)
	}()

	// Add token to blacklist
	err := blacklist.Add(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// Verify token is blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Fast-forward time in miniredis to trigger expiration
	mr.FastForward(3 * time.Second)

	// Verify token is no longer blacklisted (expired from Redis)
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestTokenBlacklist_IsBlacklisted(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	tokenID := "check-token"

	// Token should not be blacklisted initially
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	// Add token to blacklist
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	err = blacklist.Add(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// Clean up after test
	defer func() {
		_ = blacklist.Remove(ctx, tokenID)
	}()

	// Token should be blacklisted now
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestTokenBlacklist_IsBlacklisted_EmptyTokenID(t *testing.T) {
	t.Parallel()

	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	isBlacklisted, err := blacklist.IsBlacklisted(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token id cannot be empty")
	assert.False(t, isBlacklisted)
}

func TestTokenBlacklist_Remove(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	tokenID := "remove-token"
	expiresAt := time.Now().UTC().Add(5 * time.Minute)

	// Add token to blacklist
	err := blacklist.Add(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// Verify token is blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Remove token from blacklist
	err = blacklist.Remove(ctx, tokenID)
	require.NoError(t, err)

	// Verify token is no longer blacklisted
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestTokenBlacklist_Remove_EmptyTokenID(t *testing.T) {
	t.Parallel()

	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	err := blacklist.Remove(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token id cannot be empty")
}

func TestTokenBlacklist_Count(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	// Clear existing blacklist
	err := blacklist.Clear(ctx)
	require.NoError(t, err)

	// Count should be 0
	count, err := blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add some tokens
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	for i := 0; i < 5; i++ {
		tokenID := fmt.Sprintf("count-token-%d", i)
		err := blacklist.Add(ctx, tokenID, expiresAt)
		require.NoError(t, err)
	}

	// Clean up after test
	defer func() {
		_ = blacklist.Clear(ctx)
	}()

	// Count should be 5
	count, err = blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestTokenBlacklist_Clear(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	// Add some tokens
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	for i := 0; i < 3; i++ {
		tokenID := fmt.Sprintf("clear-token-%d", i)
		err := blacklist.Add(ctx, tokenID, expiresAt)
		require.NoError(t, err)
	}

	// Verify tokens are blacklisted
	count, err := blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Positive(t, count)

	// Clear blacklist
	err = blacklist.Clear(ctx)
	require.NoError(t, err)

	// Count should be 0
	count, err = blacklist.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	_, client := setupTestRedisForBlacklist(t)

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	// Clear existing blacklist
	err := blacklist.Clear(ctx)
	require.NoError(t, err)

	// Clean up after test
	defer func() {
		_ = blacklist.Clear(ctx)
	}()

	// Add multiple tokens
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	tokenIDs := []string{"token-1", "token-2", "token-3"}

	for _, tokenID := range tokenIDs {
		err := blacklist.Add(ctx, tokenID, expiresAt)
		require.NoError(t, err)
	}

	// Verify all tokens are blacklisted
	for _, tokenID := range tokenIDs {
		isBlacklisted, err := blacklist.IsBlacklisted(ctx, tokenID)
		require.NoError(t, err)
		assert.True(t, isBlacklisted, "token %s should be blacklisted", tokenID)
	}

	// Remove one token
	err = blacklist.Remove(ctx, "token-2")
	require.NoError(t, err)

	// Verify token-2 is not blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, "token-2")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)

	// Verify other tokens are still blacklisted
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, "token-1")
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	isBlacklisted, err = blacklist.IsBlacklisted(ctx, "token-3")
	require.NoError(t, err)
	assert.True(t, isBlacklisted)
}
