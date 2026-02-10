package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSessionStore creates a miniredis instance and session store for testing.
func setupSessionStore(t *testing.T) (*miniredis.Miniredis, *SessionStore) {
	t.Helper()

	mr := miniredis.RunT(t)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewSessionStore(redisClient)

	t.Cleanup(func() {
		_ = redisClient.Close()
		mr.Close()
	})

	return mr, store
}

func TestNewSessionStore_Success(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewSessionStore(redisClient)

	require.NotNil(t, store)
	assert.NotNil(t, store.redis)
}

func TestSessionStore_Create_Success(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	session := Session{
		SessionID: uuid.New().String(),
		UserID:    "user-123",
		Email:     "test@example.com",
		Role:      "user",
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify session exists in Redis
	sessionKey := sessionKeyPrefix + session.SessionID
	exists := mr.Exists(sessionKey)
	assert.True(t, exists)

	// Verify session data
	data, err := mr.Get(sessionKey)
	require.NoError(t, err)

	var storedSession Session
	err = json.Unmarshal([]byte(data), &storedSession)
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, storedSession.SessionID)
	assert.Equal(t, session.UserID, storedSession.UserID)

	// Verify user session set
	userSessionsKey := userSessionsKeyPrefix + session.UserID
	isMember, err := mr.IsMember(userSessionsKey, session.SessionID)
	require.NoError(t, err)
	assert.True(t, isMember)
}

func TestSessionStore_Create_ValidationErrors(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
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
			name: "zero expiration time",
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := store.Create(ctx, tt.session)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestSessionStore_Get_Success(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	session := Session{
		SessionID: uuid.New().String(),
		UserID:    "user-123",
		Email:     "test@example.com",
		Role:      "admin",
		IP:        "10.0.0.1",
		UserAgent: "TestAgent/1.0",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
	}

	// Store session directly in miniredis
	sessionKey := sessionKeyPrefix + session.SessionID
	data, err := json.Marshal(session)
	require.NoError(t, err)
	mr.Set(sessionKey, string(data))

	// Get session
	retrieved, err := store.Get(ctx, session.SessionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, session.SessionID, retrieved.SessionID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Email, retrieved.Email)
	assert.Equal(t, session.Role, retrieved.Role)
	assert.Equal(t, session.IP, retrieved.IP)
	assert.Equal(t, session.UserAgent, retrieved.UserAgent)
}

func TestSessionStore_Get_NotFound_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "nonexistent-session-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	assert.Nil(t, retrieved)
}

func TestSessionStore_Get_EmptySessionID_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session id cannot be empty")
	assert.Nil(t, retrieved)
}

func TestSessionStore_Get_InvalidJSON(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	sessionID := uuid.New().String()
	sessionKey := sessionKeyPrefix + sessionID

	// Store invalid JSON
	mr.Set(sessionKey, "invalid-json{")

	retrieved, err := store.Get(ctx, sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize session")
	assert.Nil(t, retrieved)
}

func TestSessionStore_Exists_Success(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	sessionID := uuid.New().String()

	// Session should not exist initially
	exists, err := store.Exists(ctx, sessionID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create session
	sessionKey := sessionKeyPrefix + sessionID
	mr.Set(sessionKey, "data")

	// Session should exist now
	exists, err = store.Exists(ctx, sessionID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSessionStore_Exists_EmptySessionID_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	exists, err := store.Exists(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session id cannot be empty")
	assert.False(t, exists)
}

func TestSessionStore_Revoke_Success(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	session := Session{
		SessionID: uuid.New().String(),
		UserID:    "user-123",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	// Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Verify session exists
	sessionKey := sessionKeyPrefix + session.SessionID
	exists := mr.Exists(sessionKey)
	assert.True(t, exists)

	// Revoke session
	err = store.Revoke(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify session is deleted
	exists = mr.Exists(sessionKey)
	assert.False(t, exists)

	// Verify session removed from user set
	userSessionsKey := userSessionsKeyPrefix + session.UserID
	isMember, _ := mr.IsMember(userSessionsKey, session.SessionID)
	// Even if set doesn't exist, member should not be present
	assert.False(t, isMember)
}

func TestSessionStore_Revoke_EmptySessionID(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	err := store.Revoke(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session id cannot be empty")
}

func TestSessionStore_Revoke_NonexistentSession_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	// Revoking nonexistent session should be idempotent.
	err := store.Revoke(ctx, "nonexistent-session")
	require.NoError(t, err)
}

func TestSessionStore_RevokeAll_Success(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	userID := "user-123"

	// Create multiple sessions for the user
	sessions := make([]Session, 3)
	for i := 0; i < 3; i++ {
		sessions[i] = Session{
			SessionID: uuid.New().String(),
			UserID:    userID,
			Email:     "test@example.com",
			Role:      "user",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, sessions[i])
		require.NoError(t, err)
	}

	// Verify all sessions exist
	for _, session := range sessions {
		exists, err := store.Exists(ctx, session.SessionID)
		require.NoError(t, err)
		assert.True(t, exists)
	}

	// Revoke all sessions
	err := store.RevokeAll(ctx, userID)
	require.NoError(t, err)

	// Verify all sessions are deleted
	for _, session := range sessions {
		exists, err := store.Exists(ctx, session.SessionID)
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestSessionStore_RevokeAll_EmptyUserID_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	err := store.RevokeAll(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user id cannot be empty")
}

func TestSessionStore_RevokeAll_NoSessions(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	// Revoking all for user with no sessions should succeed
	err := store.RevokeAll(ctx, "user-without-sessions")
	require.NoError(t, err)
}

func TestSessionStore_GetUserSessions_Success(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	userID := "user-456"

	// Create multiple sessions for the user
	expectedCount := 3
	for i := 0; i < expectedCount; i++ {
		session := Session{
			SessionID: uuid.New().String(),
			UserID:    userID,
			Email:     "test@example.com",
			Role:      "user",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Get user sessions
	sessions, err := store.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, expectedCount)

	// Verify all sessions belong to the user
	for _, session := range sessions {
		assert.Equal(t, userID, session.UserID)
	}
}

func TestSessionStore_GetUserSessions_EmptyUserID_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	sessions, err := store.GetUserSessions(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user id cannot be empty")
	assert.Nil(t, sessions)
}

func TestSessionStore_GetUserSessions_NoSessions_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	sessions, err := store.GetUserSessions(ctx, "user-without-sessions")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionStore_GetUserSessions_WithExpiredSessions(t *testing.T) {
	t.Parallel()

	mr, store := setupSessionStore(t)
	ctx := context.Background()

	userID := "user-789"

	// Create sessions
	validSession := Session{
		SessionID: uuid.New().String(),
		UserID:    userID,
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	err := store.Create(ctx, validSession)
	require.NoError(t, err)

	// Add an expired session ID to user's session set
	expiredSessionID := uuid.New().String()
	userSessionsKey := userSessionsKeyPrefix + userID
	mr.SetAdd(userSessionsKey, expiredSessionID)
	// Note: Don't create the actual session, so it appears expired/deleted

	// Get user sessions - should only return valid session
	sessions, err := store.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, validSession.SessionID, sessions[0].SessionID)
}

func TestSessionStore_Count_Success(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	// Initially no sessions
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create some sessions
	expectedCount := 5
	for i := 0; i < expectedCount; i++ {
		session := Session{
			SessionID: uuid.New().String(),
			UserID:    uuid.New().String(),
			Email:     "test@example.com",
			Role:      "user",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Count should match
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(expectedCount), count)
}

func TestSessionStore_Clear_Success(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	// Create some sessions
	for i := 0; i < 5; i++ {
		session := Session{
			SessionID: uuid.New().String(),
			UserID:    uuid.New().String(),
			Email:     "test@example.com",
			Role:      "user",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Verify sessions exist
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// Clear all sessions
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Count should be 0
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestSessionStore_SessionLifecycle(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	userID := "lifecycle-user"
	sessionID := uuid.New().String()

	session := Session{
		SessionID: sessionID,
		UserID:    userID,
		Email:     "lifecycle@example.com",
		Role:      "user",
		IP:        "127.0.0.1",
		UserAgent: "Test/1.0",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	// 1. Create session
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// 2. Verify session exists
	exists, err := store.Exists(ctx, sessionID)
	require.NoError(t, err)
	assert.True(t, exists)

	// 3. Get session
	retrieved, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrieved.SessionID)
	assert.Equal(t, userID, retrieved.UserID)

	// 4. Get user sessions
	userSessions, err := store.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userSessions, 1)

	// 5. Count sessions
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// 6. Revoke session
	err = store.Revoke(ctx, sessionID)
	require.NoError(t, err)

	// 7. Verify session no longer exists
	exists, err = store.Exists(ctx, sessionID)
	require.NoError(t, err)
	assert.False(t, exists)

	// 8. Count should be 0
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestSessionStore_MultipleUsers_Miniredis(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)
	ctx := context.Background()

	user1ID := "user-1"
	user2ID := "user-2"

	// Create 2 sessions for user1
	for i := 0; i < 2; i++ {
		session := Session{
			SessionID: uuid.New().String(),
			UserID:    user1ID,
			Email:     "user1@example.com",
			Role:      "user",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Create 3 sessions for user2
	for i := 0; i < 3; i++ {
		session := Session{
			SessionID: uuid.New().String(),
			UserID:    user2ID,
			Email:     "user2@example.com",
			Role:      "admin",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Verify user1 has 2 sessions
	user1Sessions, err := store.GetUserSessions(ctx, user1ID)
	require.NoError(t, err)
	assert.Len(t, user1Sessions, 2)

	// Verify user2 has 3 sessions
	user2Sessions, err := store.GetUserSessions(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 3)

	// Total count should be 5
	count, err := store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Revoke all sessions for user1
	err = store.RevokeAll(ctx, user1ID)
	require.NoError(t, err)

	// User1 should have no sessions
	user1Sessions, err = store.GetUserSessions(ctx, user1ID)
	require.NoError(t, err)
	assert.Empty(t, user1Sessions)

	// User2 should still have 3 sessions
	user2Sessions, err = store.GetUserSessions(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 3)

	// Total count should be 3
	count, err = store.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestSessionStore_ContextCancellation(t *testing.T) {
	t.Parallel()

	_, store := setupSessionStore(t)

	session := Session{
		SessionID: uuid.New().String(),
		UserID:    "user-123",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}

	tests := []struct {
		name      string
		operation func(ctx context.Context) error
	}{
		{
			name: "Create with canceled context",
			operation: func(ctx context.Context) error {
				return store.Create(ctx, session)
			},
		},
		{
			name: "Get with canceled context",
			operation: func(ctx context.Context) error {
				_, err := store.Get(ctx, session.SessionID)
				return err
			},
		},
		{
			name: "Exists with canceled context",
			operation: func(ctx context.Context) error {
				_, err := store.Exists(ctx, session.SessionID)
				return err
			},
		},
		{
			name: "Revoke with canceled context",
			operation: func(ctx context.Context) error {
				return store.Revoke(ctx, session.SessionID)
			},
		},
		{
			name: "RevokeAll with canceled context",
			operation: func(ctx context.Context) error {
				return store.RevokeAll(ctx, session.UserID)
			},
		},
		{
			name: "GetUserSessions with canceled context",
			operation: func(ctx context.Context) error {
				_, err := store.GetUserSessions(ctx, session.UserID)
				return err
			},
		},
		{
			name: "Count with canceled context",
			operation: func(ctx context.Context) error {
				_, err := store.Count(ctx)
				return err
			},
		},
		{
			name: "Clear with canceled context",
			operation: func(ctx context.Context) error {
				return store.Clear(ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := tt.operation(ctx)
			require.Error(t, err)
		})
	}
}
