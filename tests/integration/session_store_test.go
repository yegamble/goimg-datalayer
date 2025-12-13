//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

// TestSessionStore_Create tests storing a session in Redis.
func TestSessionStore_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create session store instance once infrastructure layer is implemented
	// store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	sessionKey := "session:" + sessionFixture.ID.String()
	sessionData := map[string]interface{}{
		"user_id":    sessionFixture.UserID.String(),
		"ip_address": sessionFixture.IPAddress,
		"user_agent": sessionFixture.UserAgent,
	}

	// Act - Store session in Redis
	// err := store.Set(ctx, sessionKey, sessionData, 7*24*time.Hour)

	// Assert
	// require.NoError(t, err)

	// Verify data was stored
	exists, err := suite.RedisClient.Exists(ctx, sessionKey).Result()
	require.NoError(t, err)
	// assert.Equal(t, int64(1), exists)

	t.Skip("Skipping until SessionStore implementation is available")
}

// TestSessionStore_Get tests retrieving a session from Redis.
func TestSessionStore_Get(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create session store instance
	// store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - store a session
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	sessionKey := "session:" + sessionFixture.ID.String()
	sessionData := map[string]interface{}{
		"user_id":    sessionFixture.UserID.String(),
		"ip_address": sessionFixture.IPAddress,
		"user_agent": sessionFixture.UserAgent,
	}

	// Store directly in Redis for test
	pipe := suite.RedisClient.Pipeline()
	pipe.HSet(ctx, sessionKey, sessionData)
	pipe.Expire(ctx, sessionKey, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	require.NoError(t, err)

	// Act
	// retrieved, err := store.Get(ctx, sessionKey)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, sessionFixture.UserID.String(), retrieved["user_id"])

	t.Skip("Skipping until SessionStore implementation is available")
}

// TestSessionStore_Revoke tests revoking a session in Redis.
func TestSessionStore_Revoke(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create session store instance
	// store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create and store session
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)
	sessionKey := "session:" + sessionFixture.ID.String()

	// Store session
	err := suite.RedisClient.Set(ctx, sessionKey, "active", 7*24*time.Hour).Err()
	require.NoError(t, err)

	// Act
	// err = store.Revoke(ctx, sessionKey)

	// Assert
	// require.NoError(t, err)

	// Verify session is deleted
	exists, err := suite.RedisClient.Exists(ctx, sessionKey).Result()
	require.NoError(t, err)
	// assert.Equal(t, int64(0), exists)

	t.Skip("Skipping until SessionStore implementation is available")
}

// TestSessionStore_RevokeAllForUser tests revoking all sessions for a user.
func TestSessionStore_RevokeAllForUser(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create session store instance
	// store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create multiple sessions for user
	userFixture := fixtures.ValidUser(t)
	session1 := fixtures.UniqueSession(t, userFixture.ID)
	session2 := fixtures.UniqueSession(t, userFixture.ID)
	session3 := fixtures.UniqueSession(t, userFixture.ID)

	// Store sessions
	sessions := []string{
		"session:" + session1.ID.String(),
		"session:" + session2.ID.String(),
		"session:" + session3.ID.String(),
	}

	for _, key := range sessions {
		err := suite.RedisClient.Set(ctx, key, "active", 7*24*time.Hour).Err()
		require.NoError(t, err)
	}

	// Act
	// count, err := store.RevokeAllForUser(ctx, userFixture.ID)

	// Assert
	// require.NoError(t, err)
	// assert.Equal(t, 3, count)

	// Verify all sessions are deleted
	for _, key := range sessions {
		exists, err := suite.RedisClient.Exists(ctx, key).Result()
		require.NoError(t, err)
		// assert.Equal(t, int64(0), exists)
	}

	t.Skip("Skipping until SessionStore implementation is available")
}

// TestSessionStore_Expiry tests that sessions expire correctly.
func TestSessionStore_Expiry(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// TODO: Create session store instance
	// store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create session with short TTL
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)
	sessionKey := "session:" + sessionFixture.ID.String()

	// Store session with 1 second TTL
	err := suite.RedisClient.Set(ctx, sessionKey, "active", 1*time.Second).Err()
	require.NoError(t, err)

	// Verify it exists initially
	exists, err := suite.RedisClient.Exists(ctx, sessionKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	// Act - wait for expiry
	time.Sleep(2 * time.Second)

	// Assert - session should be expired
	exists, err = suite.RedisClient.Exists(ctx, sessionKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "session should have expired")
}

// TestSessionStore_GetTTL tests retrieving TTL for a session.
func TestSessionStore_GetTTL(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Arrange
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)
	sessionKey := "session:" + sessionFixture.ID.String()

	ttl := 10 * time.Minute
	err := suite.RedisClient.Set(ctx, sessionKey, "active", ttl).Err()
	require.NoError(t, err)

	// Act
	remainingTTL, err := suite.RedisClient.TTL(ctx, sessionKey).Result()

	// Assert
	require.NoError(t, err)
	assert.Greater(t, remainingTTL, 9*time.Minute, "TTL should be close to 10 minutes")
	assert.LessOrEqual(t, remainingTTL, ttl, "TTL should not exceed original")
}

// TestSessionStore_UpdateTTL tests updating TTL for an existing session.
func TestSessionStore_UpdateTTL(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Arrange - create session with initial TTL
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)
	sessionKey := "session:" + sessionFixture.ID.String()

	err := suite.RedisClient.Set(ctx, sessionKey, "active", 5*time.Minute).Err()
	require.NoError(t, err)

	// Act - update TTL
	newTTL := 15 * time.Minute
	err = suite.RedisClient.Expire(ctx, sessionKey, newTTL).Err()

	// Assert
	require.NoError(t, err)

	// Verify new TTL
	remainingTTL, err := suite.RedisClient.TTL(ctx, sessionKey).Result()
	require.NoError(t, err)
	assert.Greater(t, remainingTTL, 14*time.Minute, "TTL should be close to 15 minutes")
}

// TestSessionStore_MultipleUsers tests session isolation between users.
func TestSessionStore_MultipleUsers(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	// Arrange - create sessions for different users
	user1 := fixtures.ValidUser(t)
	user2 := fixtures.AdminUser(t)

	session1 := fixtures.ValidSession(t, user1.ID)
	session2 := fixtures.ValidSession(t, user2.ID)

	key1 := "session:" + session1.ID.String()
	key2 := "session:" + session2.ID.String()

	err := suite.RedisClient.Set(ctx, key1, user1.ID.String(), 7*24*time.Hour).Err()
	require.NoError(t, err)
	err = suite.RedisClient.Set(ctx, key2, user2.ID.String(), 7*24*time.Hour).Err()
	require.NoError(t, err)

	// Act - retrieve both sessions
	val1, err := suite.RedisClient.Get(ctx, key1).Result()
	require.NoError(t, err)
	val2, err := suite.RedisClient.Get(ctx, key2).Result()
	require.NoError(t, err)

	// Assert - sessions are isolated
	assert.Equal(t, user1.ID.String(), val1)
	assert.Equal(t, user2.ID.String(), val2)
	assert.NotEqual(t, val1, val2)
}
