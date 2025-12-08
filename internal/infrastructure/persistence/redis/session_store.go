package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// sessionKeyPrefix is the Redis key prefix for session data.
	sessionKeyPrefix = "goimg:session:"
	// userSessionsKeyPrefix is the Redis key prefix for tracking all sessions for a user.
	userSessionsKeyPrefix = "goimg:user:sessions:"
)

// Session represents user session metadata stored in Redis.
type Session struct {
	SessionID string    `json:"session_id"` // Unique session identifier
	UserID    string    `json:"user_id"`    // User UUID
	Email     string    `json:"email"`      // User email
	Role      string    `json:"role"`       // User role
	IP        string    `json:"ip"`         // IP address of the session
	UserAgent string    `json:"user_agent"` // User agent string
	CreatedAt time.Time `json:"created_at"` // Session creation timestamp
	ExpiresAt time.Time `json:"expires_at"` // Session expiration timestamp
}

// SessionStore manages user sessions in Redis.
type SessionStore struct {
	redis *redis.Client
}

// NewSessionStore creates a new session store.
func NewSessionStore(redisClient *redis.Client) *SessionStore {
	return &SessionStore{
		redis: redisClient,
	}
}

// Create creates a new session and stores it in Redis.
// The session TTL is set to match the expiration time.
func (s *SessionStore) Create(ctx context.Context, session Session) error {
	if session.SessionID == "" {
		return fmt.Errorf("session id cannot be empty")
	}

	if session.UserID == "" {
		return fmt.Errorf("user id cannot be empty")
	}

	if session.ExpiresAt.IsZero() {
		return fmt.Errorf("expiration time cannot be zero")
	}

	// Calculate TTL
	now := time.Now().UTC()
	ttl := session.ExpiresAt.Sub(now)

	if ttl <= 0 {
		return fmt.Errorf("session is already expired")
	}

	// Serialize session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	// Store session in Redis
	sessionKey := sessionKeyPrefix + session.SessionID
	err = s.redis.Set(ctx, sessionKey, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Add session ID to user's session set
	userSessionsKey := userSessionsKeyPrefix + session.UserID
	err = s.redis.SAdd(ctx, userSessionsKey, session.SessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to add session to user set: %w", err)
	}

	// Set expiration on user's session set
	err = s.redis.Expire(ctx, userSessionsKey, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration on user session set: %w", err)
	}

	return nil
}

// Get retrieves a session by its ID.
// Returns an error if the session does not exist.
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id cannot be empty")
	}

	sessionKey := sessionKeyPrefix + sessionID

	data, err := s.redis.Get(ctx, sessionKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	var session Session
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}

	return &session, nil
}

// Exists checks if a session exists in Redis.
func (s *SessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("session id cannot be empty")
	}

	sessionKey := sessionKeyPrefix + sessionID

	exists, err := s.redis.Exists(ctx, sessionKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return exists > 0, nil
}

// Revoke revokes a single session by its ID.
func (s *SessionStore) Revoke(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id cannot be empty")
	}

	// Get session to find user ID
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		// Session might already be expired or not exist
		return fmt.Errorf("get session: %w", err)
	}

	// Delete session
	sessionKey := sessionKeyPrefix + sessionID
	err = s.redis.Del(ctx, sessionKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove session ID from user's session set
	userSessionsKey := userSessionsKeyPrefix + session.UserID
	err = s.redis.SRem(ctx, userSessionsKey, sessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove session from user set: %w", err)
	}

	return nil
}

// RevokeAll revokes all sessions for a user.
// This is useful for logout all devices or security incidents.
func (s *SessionStore) RevokeAll(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user id cannot be empty")
	}

	userSessionsKey := userSessionsKeyPrefix + userID

	// Get all session IDs for the user
	sessionIDs, err := s.redis.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Delete each session
	for _, sessionID := range sessionIDs {
		sessionKey := sessionKeyPrefix + sessionID
		err := s.redis.Del(ctx, sessionKey).Err()
		if err != nil {
			return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
		}
	}

	// Delete user's session set
	err = s.redis.Del(ctx, userSessionsKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete user session set: %w", err)
	}

	return nil
}

// GetUserSessions retrieves all active sessions for a user.
func (s *SessionStore) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id cannot be empty")
	}

	userSessionsKey := userSessionsKeyPrefix + userID

	// Get all session IDs for the user
	sessionIDs, err := s.redis.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*Session{}, nil
	}

	sessions := make([]*Session, 0, len(sessionIDs))

	// Retrieve each session
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			// Session might have expired, skip it
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Count returns the number of active sessions (for monitoring/debugging).
// Note: This uses SCAN which may be slow for large session stores.
func (s *SessionStore) Count(ctx context.Context) (int64, error) {
	var count int64
	var cursor uint64

	// Use SCAN to count keys with the session prefix
	for {
		var keys []string
		var err error

		keys, cursor, err = s.redis.Scan(ctx, cursor, sessionKeyPrefix+"*", 100).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to scan session keys: %w", err)
		}

		count += int64(len(keys))

		// Stop when cursor returns to 0
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// Clear removes all sessions (for testing purposes).
// WARNING: This uses SCAN and DEL which may be slow for large session stores.
func (s *SessionStore) Clear(ctx context.Context) error {
	var cursor uint64

	// Clear session keys
	for {
		var keys []string
		var err error

		keys, cursor, err = s.redis.Scan(ctx, cursor, sessionKeyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan session keys: %w", err)
		}

		if len(keys) > 0 {
			err = s.redis.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to delete session keys: %w", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	// Clear user session set keys
	cursor = 0
	for {
		var keys []string
		var err error

		keys, cursor, err = s.redis.Scan(ctx, cursor, userSessionsKeyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan user session keys: %w", err)
		}

		if len(keys) > 0 {
			err = s.redis.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to delete user session keys: %w", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}
