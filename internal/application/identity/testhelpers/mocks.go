package testhelpers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// MockUserRepository is a mock implementation of identity.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

// NextID generates a new UserID.
func (m *MockUserRepository) NextID() identity.UserID {
	args := m.Called()
	return args.Get(0).(identity.UserID)
}

// FindByID retrieves a user by ID.
func (m *MockUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	args := m.Called(ctx, id)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	return user, args.Error(1)
}

// FindByEmail retrieves a user by email.
func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	args := m.Called(ctx, email)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	return user, args.Error(1)
}

// FindByUsername retrieves a user by username.
func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	args := m.Called(ctx, username)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	return user, args.Error(1)
}

// Save persists a user.
func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// Delete removes a user.
func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockJWTService is a mock implementation of jwt.Service.
type MockJWTService struct {
	mock.Mock
}

// GenerateAccessToken generates a new access token.
func (m *MockJWTService) GenerateAccessToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

// GenerateRefreshToken generates a new refresh token.
func (m *MockJWTService) GenerateRefreshToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

// ValidateToken validates a JWT token and returns claims.
func (m *MockJWTService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	args := m.Called(tokenString)
	var claims *jwt.Claims
	if args.Get(0) != nil {
		claims = args.Get(0).(*jwt.Claims)
	}
	return claims, args.Error(1)
}

// ExtractTokenID extracts the JWT ID without full validation.
func (m *MockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

// GetTokenExpiration extracts the expiration time without full validation.
func (m *MockJWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	args := m.Called(tokenString)
	return args.Get(0).(time.Time), args.Error(1)
}

// MockRefreshTokenService is a mock implementation of jwt.RefreshTokenService.
type MockRefreshTokenService struct {
	mock.Mock
}

// GenerateToken generates a cryptographically secure refresh token.
func (m *MockRefreshTokenService) GenerateToken(
	ctx context.Context,
	userID, sessionID, familyID, parentHash, ip, userAgent string,
) (string, *jwt.RefreshTokenMetadata, error) {
	args := m.Called(ctx, userID, sessionID, familyID, parentHash, ip, userAgent)
	var metadata *jwt.RefreshTokenMetadata
	if args.Get(1) != nil {
		metadata = args.Get(1).(*jwt.RefreshTokenMetadata)
	}
	return args.String(0), metadata, args.Error(2)
}

// ValidateToken validates a refresh token and returns metadata.
func (m *MockRefreshTokenService) ValidateToken(ctx context.Context, token string) (*jwt.RefreshTokenMetadata, error) {
	args := m.Called(ctx, token)
	var metadata *jwt.RefreshTokenMetadata
	if args.Get(0) != nil {
		metadata = args.Get(0).(*jwt.RefreshTokenMetadata)
	}
	return metadata, args.Error(1)
}

// MarkAsUsed marks a refresh token as used.
func (m *MockRefreshTokenService) MarkAsUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// RevokeToken revokes a single refresh token.
func (m *MockRefreshTokenService) RevokeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// RevokeFamily revokes all tokens in a token family.
func (m *MockRefreshTokenService) RevokeFamily(ctx context.Context, familyID string) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

// DetectAnomalies checks for suspicious behavior in token usage.
func (m *MockRefreshTokenService) DetectAnomalies(metadata *jwt.RefreshTokenMetadata, currentIP, currentUserAgent string) bool {
	args := m.Called(metadata, currentIP, currentUserAgent)
	return args.Bool(0)
}

// MockTokenBlacklist is a mock implementation of jwt.TokenBlacklist.
type MockTokenBlacklist struct {
	mock.Mock
}

// Add adds a token to the blacklist.
func (m *MockTokenBlacklist) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	args := m.Called(ctx, tokenID, expiresAt)
	return args.Error(0)
}

// IsBlacklisted checks if a token is blacklisted.
func (m *MockTokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

// Remove removes a token from the blacklist.
func (m *MockTokenBlacklist) Remove(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

// Count returns the number of blacklisted tokens.
func (m *MockTokenBlacklist) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// Clear removes all blacklisted tokens.
func (m *MockTokenBlacklist) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSessionRepository is a mock implementation of postgres.SessionRepository.
type MockSessionRepository struct {
	mock.Mock
}

// Create creates a new session.
func (m *MockSessionRepository) Create(ctx context.Context, session *postgres.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

// GetByID retrieves a session by ID.
func (m *MockSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*postgres.Session, error) {
	args := m.Called(ctx, id)
	var session *postgres.Session
	if args.Get(0) != nil {
		session = args.Get(0).(*postgres.Session)
	}
	return session, args.Error(1)
}

// GetByUserID retrieves all active sessions for a user.
func (m *MockSessionRepository) GetByUserID(ctx context.Context, userID identity.UserID) ([]*postgres.Session, error) {
	args := m.Called(ctx, userID)
	var sessions []*postgres.Session
	if args.Get(0) != nil {
		sessions = args.Get(0).([]*postgres.Session)
	}
	return sessions, args.Error(1)
}

// Revoke revokes a session.
func (m *MockSessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// DeleteExpired deletes expired sessions.
func (m *MockSessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockSessionStore is a mock implementation of redis.SessionStore.
type MockSessionStore struct {
	mock.Mock
}

// Create creates a new session in Redis.
func (m *MockSessionStore) Create(ctx context.Context, session redis.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

// Get retrieves a session by ID.
func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*redis.Session, error) {
	args := m.Called(ctx, sessionID)
	var session *redis.Session
	if args.Get(0) != nil {
		session = args.Get(0).(*redis.Session)
	}
	return session, args.Error(1)
}

// Exists checks if a session exists.
func (m *MockSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

// Revoke revokes a single session.
func (m *MockSessionStore) Revoke(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// RevokeAll revokes all sessions for a user.
func (m *MockSessionStore) RevokeAll(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// GetUserSessions retrieves all active sessions for a user.
func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*redis.Session, error) {
	args := m.Called(ctx, userID)
	var sessions []*redis.Session
	if args.Get(0) != nil {
		sessions = args.Get(0).([]*redis.Session)
	}
	return sessions, args.Error(1)
}

// Count returns the number of active sessions.
func (m *MockSessionStore) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// Clear removes all sessions.
func (m *MockSessionStore) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
