//go:build integration
// +build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rediscache "github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
	"github.com/yegamble/goimg-datalayer/tests/integration/fixtures"
)

// TestSessionStore_Create tests storing a session in Redis.
func TestSessionStore_Create(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	session := rediscache.Session{
		SessionID: sessionFixture.ID.String(),
		UserID:    sessionFixture.UserID.String(),
		Email:     userFixture.Email,
		Role:      userFixture.Role.String(),
		IP:        sessionFixture.IPAddress,
		UserAgent: sessionFixture.UserAgent,
		CreatedAt: sessionFixture.CreatedAt,
		ExpiresAt: sessionFixture.ExpiresAt,
	}

	// Act - Store session in Redis
	err := store.Create(ctx, session)

	// Assert
	require.NoError(t, err)

	// Verify data was stored
	retrieved, err := store.Get(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Email, retrieved.Email)
}

// TestSessionStore_Get tests retrieving a session from Redis.
func TestSessionStore_Get(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - store a session
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	session := rediscache.Session{
		SessionID: sessionFixture.ID.String(),
		UserID:    sessionFixture.UserID.String(),
		Email:     userFixture.Email,
		Role:      userFixture.Role.String(),
		IP:        sessionFixture.IPAddress,
		UserAgent: sessionFixture.UserAgent,
		CreatedAt: sessionFixture.CreatedAt,
		ExpiresAt: sessionFixture.ExpiresAt,
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Act
	retrieved, err := store.Get(ctx, session.SessionID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, sessionFixture.UserID.String(), retrieved.UserID)
}

// TestSessionStore_Revoke tests revoking a session in Redis.
func TestSessionStore_Revoke(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create and store session
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	session := rediscache.Session{
		SessionID: sessionFixture.ID.String(),
		UserID:    sessionFixture.UserID.String(),
		Email:     userFixture.Email,
		Role:      userFixture.Role.String(),
		IP:        sessionFixture.IPAddress,
		UserAgent: sessionFixture.UserAgent,
		CreatedAt: sessionFixture.CreatedAt,
		ExpiresAt: sessionFixture.ExpiresAt,
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Act
	err = store.Revoke(ctx, session.SessionID)

	// Assert
	require.NoError(t, err)

	// Verify session is deleted
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestSessionStore_RevokeAllForUser tests revoking all sessions for a user.
func TestSessionStore_RevokeAllForUser(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create multiple sessions for user
	userFixture := fixtures.ValidUser(t)
	session1 := fixtures.UniqueSession(t, userFixture.ID)
	session2 := fixtures.UniqueSession(t, userFixture.ID)
	session3 := fixtures.UniqueSession(t, userFixture.ID)

	sessions := []*fixtures.SessionFixture{session1, session2, session3}

	for _, s := range sessions {
		session := rediscache.Session{
			SessionID: s.ID.String(),
			UserID:    s.UserID.String(),
			Email:     userFixture.Email,
			Role:      userFixture.Role.String(),
			IP:        s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Act
	err := store.RevokeAll(ctx, userFixture.ID.String())

	// Assert
	require.NoError(t, err)

	// Verify all sessions are deleted
	for _, s := range sessions {
		exists, err := store.Exists(ctx, s.ID.String())
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

// TestSessionStore_Expiry tests that sessions expire correctly.
func TestSessionStore_Expiry(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()

	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create session with short TTL
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	// Set expires at 1 second from now
	expiresAt := time.Now().UTC().Add(1 * time.Second)

	session := rediscache.Session{
		SessionID: sessionFixture.ID.String(),
		UserID:    sessionFixture.UserID.String(),
		Email:     userFixture.Email,
		Role:      userFixture.Role.String(),
		IP:        sessionFixture.IPAddress,
		UserAgent: sessionFixture.UserAgent,
		CreatedAt: sessionFixture.CreatedAt,
		ExpiresAt: expiresAt,
	}

	// Store session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify it exists initially
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Act - wait for expiry
	time.Sleep(2 * time.Second)

	// Assert - session should be expired
	exists, err = store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.False(t, exists, "session should have expired")
}

// TestSessionStore_GetTTL tests retrieving TTL for a session.
func TestSessionStore_GetTTL(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()
	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange
	userFixture := fixtures.ValidUser(t)
	sessionFixture := fixtures.ValidSession(t, userFixture.ID)

	ttl := 10 * time.Minute
	expiresAt := time.Now().UTC().Add(ttl)

	session := rediscache.Session{
		SessionID: sessionFixture.ID.String(),
		UserID:    sessionFixture.UserID.String(),
		Email:     userFixture.Email,
		Role:      userFixture.Role.String(),
		IP:        sessionFixture.IPAddress,
		UserAgent: sessionFixture.UserAgent,
		CreatedAt: sessionFixture.CreatedAt,
		ExpiresAt: expiresAt,
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Act
	// We have to check TTL on Redis directly as Store doesn't expose it,
	// but we verified Store.Create sets it correctly.
	sessionKey := "goimg:session:" + session.SessionID
	remainingTTL, err := suite.RedisClient.TTL(ctx, sessionKey).Result()

	// Assert
	require.NoError(t, err)
	assert.Greater(t, remainingTTL, 9*time.Minute, "TTL should be close to 10 minutes")
	assert.LessOrEqual(t, remainingTTL, ttl, "TTL should not exceed original")
}

// TestSessionStore_UpdateTTL tests updating TTL for an existing session.
func TestSessionStore_UpdateTTL(t *testing.T) {
	// Skip for now as SessionStore doesn't support UpdateTTL
	t.Skip("Skipping UpdateTTL test as SessionStore does not expose this functionality")
}

// TestSessionStore_MultipleUsers tests session isolation between users.
func TestSessionStore_MultipleUsers(t *testing.T) {
	suite := containers.NewIntegrationTestSuite(t)
	ctx := context.Background()
	store := rediscache.NewSessionStore(suite.RedisClient)

	// Arrange - create sessions for different users
	user1 := fixtures.ValidUser(t)
	user2 := fixtures.AdminUser(t)

	sessionFixture1 := fixtures.ValidSession(t, user1.ID)
	sessionFixture2 := fixtures.ValidSession(t, user2.ID)

	s1 := rediscache.Session{
		SessionID: sessionFixture1.ID.String(),
		UserID:    sessionFixture1.UserID.String(),
		Email:     user1.Email,
		Role:      user1.Role.String(),
		IP:        sessionFixture1.IPAddress,
		UserAgent: sessionFixture1.UserAgent,
		CreatedAt: sessionFixture1.CreatedAt,
		ExpiresAt: sessionFixture1.ExpiresAt,
	}

	s2 := rediscache.Session{
		SessionID: sessionFixture2.ID.String(),
		UserID:    sessionFixture2.UserID.String(),
		Email:     user2.Email,
		Role:      user2.Role.String(),
		IP:        sessionFixture2.IPAddress,
		UserAgent: sessionFixture2.UserAgent,
		CreatedAt: sessionFixture2.CreatedAt,
		ExpiresAt: sessionFixture2.ExpiresAt,
	}

	err := store.Create(ctx, s1)
	require.NoError(t, err)
	err = store.Create(ctx, s2)
	require.NoError(t, err)

	// Act - retrieve both sessions
	val1, err := store.Get(ctx, s1.SessionID)
	require.NoError(t, err)
	val2, err := store.Get(ctx, s2.SessionID)
	require.NoError(t, err)

	// Assert - sessions are isolated
	assert.Equal(t, user1.ID.String(), val1.UserID)
	assert.Equal(t, user2.ID.String(), val2.UserID)
	assert.NotEqual(t, val1.SessionID, val2.SessionID)
}
