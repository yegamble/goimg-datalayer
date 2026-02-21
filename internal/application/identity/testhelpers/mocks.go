package testhelpers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) NextID() identity.UserID {
	args := m.Called()
	return args.Get(0).(identity.UserID)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id identity.UserID) (*identity.User, error) {
	args := m.Called(ctx, id)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	if err := args.Error(1); err != nil {
		return user, fmt.Errorf("mock FindByID: %w", err)
	}
	return user, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	args := m.Called(ctx, email)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	if err := args.Error(1); err != nil {
		return user, fmt.Errorf("mock FindByEmail: %w", err)
	}
	return user, nil
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username identity.Username) (*identity.User, error) {
	args := m.Called(ctx, username)
	var user *identity.User
	if args.Get(0) != nil {
		user = args.Get(0).(*identity.User)
	}
	if err := args.Error(1); err != nil {
		return user, fmt.Errorf("mock FindByUsername: %w", err)
	}
	return user, nil
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	args := m.Called(ctx, id)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

func (m *MockUserRepository) FindExpiredGuests(ctx context.Context, asOf time.Time, limit int) ([]*identity.User, error) {
	args := m.Called(ctx, asOf, limit)
	var users []*identity.User
	if args.Get(0) != nil {
		users = args.Get(0).([]*identity.User)
	}
	if err := args.Error(1); err != nil {
		return users, fmt.Errorf("mock FindExpiredGuests: %w", err)
	}
	return users, nil
}

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateAccessToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateElevatedAccessToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*services.JWTClaims, error) {
	args := m.Called(tokenString)
	var claims *services.JWTClaims
	if args.Get(0) != nil {
		claims = args.Get(0).(*services.JWTClaims)
	}
	if err := args.Error(1); err != nil {
		return claims, fmt.Errorf("mock ValidateToken: %w", err)
	}
	return claims, nil
}

func (m *MockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	args := m.Called(tokenString)
	if err := args.Error(1); err != nil {
		return time.Time{}, fmt.Errorf("get token expiration: %w", err)
	}
	return args.Get(0).(time.Time), nil
}

type MockRefreshTokenService struct {
	mock.Mock
}

func (m *MockRefreshTokenService) GenerateToken(
	ctx context.Context,
	userID, sessionID, familyID, parentHash, ip, userAgent string,
) (string, *services.RefreshTokenMetadata, error) {
	args := m.Called(ctx, userID, sessionID, familyID, parentHash, ip, userAgent)
	var metadata *services.RefreshTokenMetadata
	if args.Get(1) != nil {
		metadata = args.Get(1).(*services.RefreshTokenMetadata)
	}
	return args.String(0), metadata, args.Error(2)
}

func (m *MockRefreshTokenService) ValidateToken(ctx context.Context, token string) (*services.RefreshTokenMetadata, error) {
	args := m.Called(ctx, token)
	var metadata *services.RefreshTokenMetadata
	if args.Get(0) != nil {
		metadata = args.Get(0).(*services.RefreshTokenMetadata)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}
	return metadata, nil
}

func (m *MockRefreshTokenService) MarkAsUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mark as used: %w", err)
	}
	return nil
}

func (m *MockRefreshTokenService) RevokeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenService) RevokeFamily(ctx context.Context, familyID string) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

func (m *MockRefreshTokenService) DetectAnomalies(metadata *services.RefreshTokenMetadata, currentIP, currentUserAgent string) bool {
	args := m.Called(metadata, currentIP, currentUserAgent)
	return args.Bool(0)
}

type MockTokenBlacklist struct {
	mock.Mock
}

func (m *MockTokenBlacklist) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	args := m.Called(ctx, tokenID, expiresAt)
	return args.Error(0)
}

func (m *MockTokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenBlacklist) Remove(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockTokenBlacklist) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTokenBlacklist) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *postgres.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*postgres.Session, error) {
	args := m.Called(ctx, id)
	var session *postgres.Session
	if args.Get(0) != nil {
		session = args.Get(0).(*postgres.Session)
	}
	return session, args.Error(1)
}

func (m *MockSessionRepository) GetByUserID(ctx context.Context, userID identity.UserID) ([]*postgres.Session, error) {
	args := m.Called(ctx, userID)
	var sessions []*postgres.Session
	if args.Get(0) != nil {
		sessions = args.Get(0).([]*postgres.Session)
	}
	return sessions, args.Error(1)
}

func (m *MockSessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, session services.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*services.Session, error) {
	args := m.Called(ctx, sessionID)
	var session *services.Session
	if args.Get(0) != nil {
		session = args.Get(0).(*services.Session)
	}
	return session, args.Error(1)
}

func (m *MockSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionStore) Revoke(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStore) RevokeAll(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*services.Session, error) {
	args := m.Called(ctx, userID)
	var sessions []*services.Session
	if args.Get(0) != nil {
		sessions = args.Get(0).([]*services.Session)
	}
	return sessions, args.Error(1)
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type MockAuthMetricsRecorder struct {
	mock.Mock
}

func (m *MockAuthMetricsRecorder) RecordLoginDelay(delaySeconds float64) {
	m.Called(delaySeconds)
}
