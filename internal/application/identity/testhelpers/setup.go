package testhelpers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

type TestSuite struct {
	UserRepo  *MockUserRepository
	TokenRepo *MockTokenRepository

	JWTService          *MockJWTService
	RefreshTokenService *MockRefreshTokenService
	TokenBlacklist      *MockTokenBlacklist

	SessionRepo  *MockSessionRepository
	SessionStore *MockSessionStore

	EventPublisher *MockEventPublisher

	AuthMetrics *MockAuthMetricsRecorder

	EmailSender *MockEmailSender

	Logger zerolog.Logger

	t *testing.T
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()
	return &TestSuite{
		UserRepo:            new(MockUserRepository),
		TokenRepo:           new(MockTokenRepository),
		JWTService:          new(MockJWTService),
		RefreshTokenService: new(MockRefreshTokenService),
		TokenBlacklist:      new(MockTokenBlacklist),
		SessionRepo:         new(MockSessionRepository),
		SessionStore:        new(MockSessionStore),
		EventPublisher:      new(MockEventPublisher),
		AuthMetrics:         new(MockAuthMetricsRecorder),
		EmailSender:         new(MockEmailSender),
		Logger:              zerolog.Nop(),
		t:                   t,
	}
}

func (s *TestSuite) AssertExpectations() {
	s.UserRepo.AssertExpectations(s.t)
	s.TokenRepo.AssertExpectations(s.t)
	s.JWTService.AssertExpectations(s.t)
	s.RefreshTokenService.AssertExpectations(s.t)
	s.TokenBlacklist.AssertExpectations(s.t)
	s.SessionRepo.AssertExpectations(s.t)
	s.SessionStore.AssertExpectations(s.t)
	s.EventPublisher.AssertExpectations(s.t)
	s.AuthMetrics.AssertExpectations(s.t)
	s.EmailSender.AssertExpectations(s.t)
}

func (s *TestSuite) SetupSuccessfulUserCreation() {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)
}

func (s *TestSuite) SetupSuccessfulLogin(user *identity.User) {
	sessionID := ValidSessionID.String()

	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(user, nil)

	s.JWTService.On(
		"GenerateAccessToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("access.token.value", nil)

	s.JWTService.On(
		"GenerateRefreshToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("refresh.token.value", nil)

	s.RefreshTokenService.On(
		"GenerateToken",
		mock.Anything,
		user.ID().String(),
		sessionID,
		mock.Anything,
		"",
		ValidIPAddress,
		ValidUserAgent,
	).Return("refresh.token.value", ValidRefreshTokenMetadata(), nil)

	s.SessionRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil)

	s.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(nil)
}

func (s *TestSuite) SetupSuccessfulTokenRefresh(user *identity.User, metadata *jwt.RefreshTokenMetadata) {
	sessionID := metadata.SessionID

	s.RefreshTokenService.On("ValidateToken", mock.Anything, mock.Anything).
		Return(metadata, nil)

	s.RefreshTokenService.On("DetectAnomalies", metadata, ValidIPAddress, ValidUserAgent).
		Return(false)

	s.RefreshTokenService.On("MarkAsUsed", mock.Anything, mock.Anything).
		Return(nil)

	userID, _ := identity.ParseUserID(metadata.UserID)
	s.UserRepo.On("FindByID", mock.Anything, userID).
		Return(user, nil)

	s.JWTService.On(
		"GenerateAccessToken",
		user.ID().String(),
		user.Email().String(),
		string(user.Role()),
		sessionID,
	).Return("new.access.token", nil)

	s.RefreshTokenService.On(
		"GenerateToken",
		mock.Anything,
		user.ID().String(),
		sessionID,
		metadata.FamilyID,
		metadata.TokenHash,
		ValidIPAddress,
		ValidUserAgent,
	).Return("new.refresh.token", ValidRefreshTokenMetadata(), nil)
}

func (s *TestSuite) SetupSuccessfulLogout(tokenID string, sessionID string) {
	s.TokenBlacklist.On("Add", mock.Anything, tokenID, mock.Anything).
		Return(nil)

	s.RefreshTokenService.On("RevokeToken", mock.Anything, mock.Anything).
		Return(nil)

	sessionUUID, _ := uuid.Parse(sessionID)
	s.SessionRepo.On("Revoke", mock.Anything, sessionUUID).
		Return(nil)

	s.SessionStore.On("Revoke", mock.Anything, sessionID).
		Return(nil)
}

func (s *TestSuite) SetupUserNotFound() {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByID", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
}

func (s *TestSuite) SetupEmailAlreadyExists(existingUser *identity.User) {
	s.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(existingUser, nil)
}

func (s *TestSuite) SetupUsernameAlreadyExists(existingUser *identity.User) {
	s.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(existingUser, nil)
}
