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
	if err := args.Error(1); err != nil {
		return user, fmt.Errorf("mock FindByID: %w", err)
	}
	return user, nil
}

// FindByEmail retrieves a user by email.
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

// FindByUsername retrieves a user by username.
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

// Save persists a user.
func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

// Delete removes a user.
func (m *MockUserRepository) Delete(ctx context.Context, id identity.UserID) error {
	args := m.Called(ctx, id)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

// FindExpiredGuests retrieves all guest users whose expiration date has passed.
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

// MockJWTService is a mock implementation of services.JWTService.
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

// GenerateElevatedAccessToken generates an access token with 2FA verification flag.
func (m *MockJWTService) GenerateElevatedAccessToken(userID, email, role, sessionID string) (string, error) {
	args := m.Called(userID, email, role, sessionID)
	return args.String(0), args.Error(1)
}

// ValidateToken validates a JWT token and returns claims.
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

// ExtractTokenID extracts the JWT ID without full validation.
func (m *MockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

// GetTokenExpiration extracts the expiration time without full validation.
func (m *MockJWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	args := m.Called(tokenString)
	if err := args.Error(1); err != nil {
		return time.Time{}, fmt.Errorf("get token expiration: %w", err)
	}
	return args.Get(0).(time.Time), nil
}

// MockRefreshTokenService is a mock implementation of services.RefreshTokenService.
type MockRefreshTokenService struct {
	mock.Mock
}

// GenerateToken generates a cryptographically secure refresh token.
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

// ValidateToken validates a refresh token and returns metadata.
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

// MarkAsUsed marks a refresh token as used.
func (m *MockRefreshTokenService) MarkAsUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mark as used: %w", err)
	}
	return nil
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
func (m *MockRefreshTokenService) DetectAnomalies(metadata *services.RefreshTokenMetadata, currentIP, currentUserAgent string) bool {
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

// MockSessionStore is a mock implementation of services.SessionStore.
type MockSessionStore struct {
	mock.Mock
}

// Create creates a new session in Redis.
func (m *MockSessionStore) Create(ctx context.Context, session services.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

// Get retrieves a session by ID.
func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*services.Session, error) {
	args := m.Called(ctx, sessionID)
	var session *services.Session
	if args.Get(0) != nil {
		session = args.Get(0).(*services.Session)
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
func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*services.Session, error) {
	args := m.Called(ctx, userID)
	var sessions []*services.Session
	if args.Get(0) != nil {
		sessions = args.Get(0).([]*services.Session)
	}
	return sessions, args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

// Publish publishes a domain event.
func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockAuthMetricsRecorder is a mock implementation of appidentity.AuthMetricsRecorder.
type MockAuthMetricsRecorder struct {
	mock.Mock
}

// RecordLoginDelay records the random delay applied to a login attempt.
func (m *MockAuthMetricsRecorder) RecordLoginDelay(delaySeconds float64) {
	m.Called(delaySeconds)
}

// MockFollowRepository is a mock implementation of identity.FollowRepository.
type MockFollowRepository struct {
	mock.Mock
}

// Save persists a follow relationship.
func (m *MockFollowRepository) Save(ctx context.Context, follow *identity.Follow) error {
	args := m.Called(ctx, follow)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

// Delete removes a follow relationship.
func (m *MockFollowRepository) Delete(ctx context.Context, followerID, followedID identity.UserID) error {
	args := m.Called(ctx, followerID, followedID)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

// Exists checks whether a follow relationship exists.
func (m *MockFollowRepository) Exists(ctx context.Context, followerID, followedID identity.UserID) (bool, error) {
	args := m.Called(ctx, followerID, followedID)
	return args.Bool(0), args.Error(1)
}

// FindFollowers retrieves all followers for a user.
func (m *MockFollowRepository) FindFollowers(ctx context.Context, userID identity.UserID, limit, offset int) ([]*identity.Follow, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	var follows []*identity.Follow
	if args.Get(0) != nil {
		follows = args.Get(0).([]*identity.Follow)
	}
	return follows, args.Int(1), args.Error(2)
}

// FindFollowing retrieves all users that a user is following.
func (m *MockFollowRepository) FindFollowing(ctx context.Context, userID identity.UserID, limit, offset int) ([]*identity.Follow, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	var follows []*identity.Follow
	if args.Get(0) != nil {
		follows = args.Get(0).([]*identity.Follow)
	}
	return follows, args.Int(1), args.Error(2)
}

// CountFollowers returns the number of followers for a user.
func (m *MockFollowRepository) CountFollowers(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// CountFollowing returns the number of users that a user is following.
func (m *MockFollowRepository) CountFollowing(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// MockOAuthAccountRepository is a mock implementation of identity.OAuthAccountRepository.
type MockOAuthAccountRepository struct {
	mock.Mock
}

// FindByID retrieves an OAuth account by its unique ID.
func (m *MockOAuthAccountRepository) FindByID(ctx context.Context, id identity.OAuthAccountID) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, id)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByID: %w", err)
	}
	return account, nil
}

// FindByProviderAndUserID retrieves an OAuth account by provider and provider user ID.
func (m *MockOAuthAccountRepository) FindByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, provider, providerUserID)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByProviderAndUserID: %w", err)
	}
	return account, nil
}

// FindByUserID retrieves all OAuth accounts for a user.
func (m *MockOAuthAccountRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]*identity.OAuthAccount, error) {
	args := m.Called(ctx, userID)
	var accounts []*identity.OAuthAccount
	if args.Get(0) != nil {
		accounts = args.Get(0).([]*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserID: %w", err)
	}
	return accounts, nil
}

// FindByUserIDAndProvider retrieves a specific OAuth account for a user and provider.
func (m *MockOAuthAccountRepository) FindByUserIDAndProvider(
	ctx context.Context,
	userID identity.UserID,
	provider identity.OAuthProvider,
) (*identity.OAuthAccount, error) {
	args := m.Called(ctx, userID, provider)
	var account *identity.OAuthAccount
	if args.Get(0) != nil {
		account = args.Get(0).(*identity.OAuthAccount)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserIDAndProvider: %w", err)
	}
	return account, nil
}

// Save persists an OAuth account.
func (m *MockOAuthAccountRepository) Save(ctx context.Context, account *identity.OAuthAccount) error {
	args := m.Called(ctx, account)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Save: %w", err)
	}
	return nil
}

// Delete removes an OAuth account.
func (m *MockOAuthAccountRepository) Delete(ctx context.Context, id identity.OAuthAccountID) error {
	args := m.Called(ctx, id)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock Delete: %w", err)
	}
	return nil
}

// ExistsByProviderAndUserID checks if an OAuth account exists for the provider and provider user ID.
func (m *MockOAuthAccountRepository) ExistsByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (bool, error) {
	args := m.Called(ctx, provider, providerUserID)
	return args.Bool(0), args.Error(1)
}

// MockTOTPRepository is a mock implementation of queries.TOTPRepository.
type MockTOTPRepository struct {
	mock.Mock
}

// FindByUserID retrieves a TOTP secret by user ID.
func (m *MockTOTPRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*identity.TOTPSecret, error) {
	args := m.Called(ctx, userID)
	var secret *identity.TOTPSecret
	if args.Get(0) != nil {
		secret = args.Get(0).(*identity.TOTPSecret)
	}
	if err := args.Error(1); err != nil {
		return nil, fmt.Errorf("mock FindByUserID: %w", err)
	}
	return secret, nil
}

// IsEnabled checks if 2FA is enabled for a user.
func (m *MockTOTPRepository) IsEnabled(ctx context.Context, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

// Save persists a TOTP secret.
func (m *MockTOTPRepository) Save(ctx context.Context, userID identity.UserID, secret identity.TOTPSecret) error {
	args := m.Called(ctx, userID, secret)
	return args.Error(0)
}

// Delete removes a TOTP secret.
func (m *MockTOTPRepository) Delete(ctx context.Context, userID identity.UserID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// DeleteByUserID deletes all backup codes for a user (alias for DeleteAll).
func (m *MockBackupCodeRepository) DeleteByUserID(ctx context.Context, userID identity.UserID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockBackupCodeRepository is a mock implementation of queries.BackupCodeRepository.
type MockBackupCodeRepository struct {
	mock.Mock
}

// CountUnused returns the number of unused backup codes for a user.
func (m *MockBackupCodeRepository) CountUnused(ctx context.Context, userID identity.UserID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// SaveAll persists a batch of backup codes.
func (m *MockBackupCodeRepository) SaveAll(ctx context.Context, userID identity.UserID, codes []identity.BackupCode) error {
	args := m.Called(ctx, userID, codes)
	return args.Error(0)
}

// FindByUserID retrieves all backup codes for a user.
func (m *MockBackupCodeRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.BackupCode), args.Error(1)
}

// FindUnusedByUserID retrieves unused backup codes for a user.
func (m *MockBackupCodeRepository) FindUnusedByUserID(ctx context.Context, userID identity.UserID) ([]identity.BackupCode, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]identity.BackupCode), args.Error(1)
}

// DeleteAll deletes all backup codes for a user.
func (m *MockBackupCodeRepository) DeleteAll(ctx context.Context, userID identity.UserID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// FindByUserIDAndCode retrieves a specific backup code for a user.
// This might be needed for VerifyLogin with backup code.
func (m *MockBackupCodeRepository) FindByUserIDAndCode(ctx context.Context, userID identity.UserID, code string) (*identity.BackupCode, error) {
	args := m.Called(ctx, userID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.BackupCode), args.Error(1)
}

// Save persists a single backup code.
func (m *MockBackupCodeRepository) Save(ctx context.Context, code *identity.BackupCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}
