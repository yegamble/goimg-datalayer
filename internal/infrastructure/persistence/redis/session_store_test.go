package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionStore(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())

	assert.NotNil(t, store)
	assert.NotNil(t, store.redis)
}

func createTestSession(userID string) Session {
	return Session{
		SessionID: uuid.New().String(),
		UserID:    userID,
		Email:     "test@example.com",
		Role:      "user",
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
}

func TestSessionStore_Create(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() {
		if err := store.Clear(ctx); err != nil {
			t.Logf("failed to clear sessions: %v", err)
		}
	}()

	session := createTestSession("user-123")

	// Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify session exists
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Retrieve session
	retrieved, err := store.Get(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Email, retrieved.Email)
	assert.Equal(t, session.Role, retrieved.Role)
	assert.Equal(t, session.IP, retrieved.IP)
	assert.Equal(t, session.UserAgent, retrieved.UserAgent)
}

func TestSessionStore_Create_InvalidSession(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	})

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	tests := []struct {
		name      string
		session   Session
		wantError string
	}{
		{
			name: "empty session id",
			session: Session{
				SessionID: "",
				UserID:    "user-123",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantError: "session id cannot be empty",
		},
		{
			name: "empty user id",
			session: Session{
				SessionID: "session-123",
				UserID:    "",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			wantError: "user id cannot be empty",
		},
		{
			name: "zero expiration",
			session: Session{
				SessionID: "session-123",
				UserID:    "user-123",
				ExpiresAt: time.Time{},
			},
			wantError: "expiration time cannot be zero",
		},
		{
			name: "already expired",
			session: Session{
				SessionID: "session-123",
				UserID:    "user-123",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			wantError: "session is already expired",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := store.Create(ctx, tt.session)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestSessionStore_Get(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() {
		if err := store.Clear(ctx); err != nil {
			t.Logf("failed to clear sessions: %v", err)
		}
	}()

	session := createTestSession("user-123")

	// Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Get session
	retrieved, err := store.Get(ctx, session.SessionID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "nonexistent-session")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	assert.Nil(t, retrieved)
}

func TestSessionStore_Get_EmptySessionID(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "session id cannot be empty")
	assert.Nil(t, retrieved)
}

func TestSessionStore_Exists(t *testing.T) {
	client := getTestClient(t)
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	}()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() {
		if err := store.Clear(ctx); err != nil {
			t.Logf("failed to clear sessions: %v", err)
		}
	}()

	session := createTestSession("user-123")

	// Session should not exist initially
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create session
	err = store.Create(ctx, session)
	require.NoError(t, err)

	// Session should exist now
	exists, err = store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSessionStore_Exists_EmptySessionID(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	defer func() { _ = client.Close() }()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	exists, err := store.Exists(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "session id cannot be empty")
	assert.False(t, exists)
}

func TestSessionStore_Revoke(t *testing.T) {
	client := getTestClient(t)
	defer func() { _ = client.Close() }()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() { _ = store.Clear(ctx) }()

	session := createTestSession("user-123")

	// Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify session exists
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Revoke session
	err = store.Revoke(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify session no longer exists
	exists, err = store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSessionStore_Revoke_NonexistentSession(t *testing.T) {
	client := getTestClient(t)
	defer func() { _ = client.Close() }()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Revoking nonexistent session should not error
	err := store.Revoke(ctx, "nonexistent-session")
	require.NoError(t, err)
}

func TestSessionStore_RevokeAll(t *testing.T) {
	client := getTestClient(t)
	defer func() { _ = client.Close() }()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() { _ = store.Clear(ctx) }()

	userID := "user-123"

	// Create multiple sessions for the same user
	sessions := make([]Session, 3)
	for i := 0; i < 3; i++ {
		sessions[i] = createTestSession(userID)
		err := store.Create(ctx, sessions[i])
		require.NoError(t, err)
	}

	// Verify all sessions exist
	for _, session := range sessions {
		exists, err := store.Exists(ctx, session.SessionID)
		require.NoError(t, err)
		assert.True(t, exists)
	}

	// Revoke all sessions for the user
	err := store.RevokeAll(ctx, userID)
	require.NoError(t, err)

	// Verify all sessions are revoked
	for _, session := range sessions {
		exists, err := store.Exists(ctx, session.SessionID)
		require.NoError(t, err)
		assert.False(t, exists, "session %s should be revoked", session.SessionID)
	}
}

func TestSessionStore_RevokeAll_EmptyUserID(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	err := store.RevokeAll(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user id cannot be empty")
}

func TestSessionStore_GetUserSessions(t *testing.T) {
	client := getTestClient(t)
	defer func() { _ = client.Close() }()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer func() { _ = store.Clear(ctx) }()

	userID := "user-123"

	// Create multiple sessions for the same user
	expectedSessions := 3
	for i := 0; i < expectedSessions; i++ {
		session := createTestSession(userID)
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Get all sessions for the user
	sessions, err := store.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, expectedSessions)

	// Verify all sessions belong to the user
	for _, session := range sessions {
		assert.Equal(t, userID, session.UserID)
	}
}

func TestSessionStore_GetUserSessions_NoSessions(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	sessions, err := store.GetUserSessions(ctx, "user-without-sessions")

	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionStore_GetUserSessions_EmptyUserID(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	sessions, err := store.GetUserSessions(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user id cannot be empty")
	assert.Nil(t, sessions)
}

func TestSessionStore_Count(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	err := store.Clear(ctx)
	require.NoError(t, err)

	// Clean up after test
	defer store.Clear(ctx)

	// Count should be 0 initially
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create some sessions
	expectedCount := 5
	for i := 0; i < expectedCount; i++ {
		session := createTestSession(uuid.New().String())
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Count should match
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(expectedCount), count)
}

func TestSessionStore_Clear(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Create some sessions
	for i := 0; i < 3; i++ {
		session := createTestSession(uuid.New().String())
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Verify sessions exist
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Positive(t, count)

	// Clear all sessions
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Count should be 0
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestSessionStore_SessionExpiration(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer store.Clear(ctx)

	session := createTestSession("user-123")
	// Set a short expiration for testing
	session.ExpiresAt = time.Now().UTC().Add(2 * time.Second)

	// Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify session exists
	exists, err := store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Wait for expiration
	time.Sleep(2500 * time.Millisecond)

	// Session should be expired (automatically removed by Redis)
	exists, err = store.Exists(ctx, session.SessionID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSessionStore_MultipleUsers(t *testing.T) {
	client := getTestClient(t)
	defer client.Close()

	store := NewSessionStore(client.UnderlyingClient())
	ctx := context.Background()

	// Clear sessions before test
	defer store.Clear(ctx)

	user1ID := "user-1"
	user2ID := "user-2"

	// Create sessions for user 1
	for i := 0; i < 2; i++ {
		session := createTestSession(user1ID)
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Create sessions for user 2
	for i := 0; i < 3; i++ {
		session := createTestSession(user2ID)
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Get sessions for user 1
	user1Sessions, err := store.GetUserSessions(ctx, user1ID)
	require.NoError(t, err)
	assert.Len(t, user1Sessions, 2)

	// Get sessions for user 2
	user2Sessions, err := store.GetUserSessions(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 3)

	// Revoke all sessions for user 1
	err = store.RevokeAll(ctx, user1ID)
	require.NoError(t, err)

	// User 1 should have no sessions
	user1Sessions, err = store.GetUserSessions(ctx, user1ID)
	require.NoError(t, err)
	assert.Empty(t, user1Sessions)

	// User 2 should still have their sessions
	user2Sessions, err = store.GetUserSessions(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 3)
}
