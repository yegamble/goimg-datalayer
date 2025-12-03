package testhelpers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// TestSuite encapsulates all mocks and test dependencies for application layer tests.
// Use NewTestSuite to create a properly initialized test suite with all mocks.
type TestSuite struct {
	// Domain Layer Mocks
	UserRepo *MockUserRepository

	// JWT Security Mocks
	JWTService          *MockJWTService
	RefreshTokenService *MockRefreshTokenService
	TokenBlacklist      *MockTokenBlacklist

	// Session Management Mocks
	SessionRepo  *MockSessionRepository // Postgres sessions
	SessionStore *MockSessionStore      // Redis sessions

	// Event Publishing Mock
	EventPublisher *MockEventPublisher

	// Logger for handlers (no-op logger for tests)
	Logger zerolog.Logger

	// Testing context
	t *testing.T
}

// NewTestSuite creates a new test suite with all mocks initialized.
// This is the recommended way to set up tests for application layer handlers.
//
// Example:
//
//	func TestMyCommand(t *testing.T) {
//	    suite := testhelpers.NewTestSuite(t)
//	    // Configure mocks
//	    suite.UserRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
//	    // Run test
//	    // ...
//	    // Verify expectations
//	    suite.AssertExpectations()
//	}
func NewTestSuite(t *testing.T) *TestSuite {
	return &TestSuite{
		// Initialize all mocks
		UserRepo:            new(MockUserRepository),
		JWTService:          new(MockJWTService),
		RefreshTokenService: new(MockRefreshTokenService),
		TokenBlacklist:      new(MockTokenBlacklist),
		SessionRepo:         new(MockSessionRepository),
		SessionStore:        new(MockSessionStore),
		EventPublisher:      new(MockEventPublisher),
		Logger:              zerolog.Nop(), // No-op logger for tests
		t:                   t,
	}
}

// AssertExpectations asserts that all mocks had their expected methods called.
// Call this at the end of each test to verify all mock expectations were met.
func (s *TestSuite) AssertExpectations() {
	s.UserRepo.AssertExpectations(s.t)
	s.JWTService.AssertExpectations(s.t)
	s.RefreshTokenService.AssertExpectations(s.t)
	s.TokenBlacklist.AssertExpectations(s.t)
	s.SessionRepo.AssertExpectations(s.t)
	s.SessionStore.AssertExpectations(s.t)
	s.EventPublisher.AssertExpectations(s.t)
}

// Helper methods for common test setups

// SetupSuccessfulUserCreation configures mocks for a successful user registration.
func (s *TestSuite) SetupSuccessfulUserCreation() {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)
}

// SetupSuccessfulLogin configures mocks for a successful login flow.
func (s *TestSuite) SetupSuccessfulLogin(user *identity.User) {
	sessionID := ValidSessionID.String()

	// User lookup succeeds
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(user, nil)

	// JWT generation succeeds
	s.JWTService.On("GenerateAccessToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("access.token.value", nil)

	s.JWTService.On("GenerateRefreshToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("refresh.token.value", nil)

	// Refresh token service generates token
	s.RefreshTokenService.On("GenerateToken",
		mock.Anything,
		user.ID().String(),
		sessionID,
		mock.Anything, // familyID
		"",            // parentHash
		ValidIPAddress,
		ValidUserAgent,
	).Return("refresh.token.value", ValidRefreshTokenMetadata(), nil)

	// Session creation succeeds
	s.SessionRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil)

	s.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(nil)
}

// SetupSuccessfulTokenRefresh configures mocks for a successful token refresh.
func (s *TestSuite) SetupSuccessfulTokenRefresh(user *identity.User, metadata *jwt.RefreshTokenMetadata) {
	sessionID := metadata.SessionID

	// Refresh token validation succeeds
	s.RefreshTokenService.On("ValidateToken", mock.Anything, mock.Anything).
		Return(metadata, nil)

	// Anomaly detection passes
	s.RefreshTokenService.On("DetectAnomalies", metadata, ValidIPAddress, ValidUserAgent).
		Return(false)

	// Mark old token as used
	s.RefreshTokenService.On("MarkAsUsed", mock.Anything, mock.Anything).
		Return(nil)

	// User lookup succeeds
	userID, _ := identity.ParseUserID(metadata.UserID)
	s.UserRepo.On("FindByID", mock.Anything, userID).
		Return(user, nil)

	// Generate new tokens
	s.JWTService.On("GenerateAccessToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("new.access.token", nil)

	// Generate new refresh token
	s.RefreshTokenService.On("GenerateToken",
		mock.Anything,
		user.ID().String(),
		sessionID,
		metadata.FamilyID,
		metadata.TokenHash, // parent hash
		ValidIPAddress,
		ValidUserAgent,
	).Return("new.refresh.token", ValidRefreshTokenMetadata(), nil)
}

// SetupSuccessfulLogout configures mocks for a successful logout.
func (s *TestSuite) SetupSuccessfulLogout(tokenID string, sessionID string) {
	// Blacklist access token
	s.TokenBlacklist.On("Add", mock.Anything, tokenID, mock.Anything).
		Return(nil)

	// Revoke refresh token
	s.RefreshTokenService.On("RevokeToken", mock.Anything, mock.Anything).
		Return(nil)

	// Revoke sessions
	sessionUUID, _ := uuid.Parse(sessionID)
	s.SessionRepo.On("Revoke", mock.Anything, sessionUUID).
		Return(nil)

	s.SessionStore.On("Revoke", mock.Anything, sessionID).
		Return(nil)
}

// SetupUserNotFound configures mocks to return "user not found" error.
func (s *TestSuite) SetupUserNotFound() {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByID", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
}

// SetupEmailAlreadyExists configures mocks to simulate duplicate email.
func (s *TestSuite) SetupEmailAlreadyExists(existingUser *identity.User) {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(existingUser, nil)
}

// SetupUsernameAlreadyExists configures mocks to simulate duplicate username.
func (s *TestSuite) SetupUsernameAlreadyExists(existingUser *identity.User) {
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(existingUser, nil)
}
