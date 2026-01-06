package commands_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// TestLoginHandler_RecordsMetrics_Success verifies that login delay metrics are
// recorded on successful login.
func TestLoginHandler_RecordsMetrics_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	// Setup successful login
	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()
	suite.JWTService.On("GenerateAccessToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("access.token.jwt", nil).Once()
	suite.RefreshTokenService.On("GenerateToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("refresh.token.value", testhelpers.ValidRefreshTokenMetadata(), nil).Once()
	suite.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(nil).Once()
	suite.JWTService.On("GetTokenExpiration", mock.Anything).
		Return(time.Now().UTC().Add(15*time.Minute), nil).Once()

	// Expect metrics to be recorded with delay in valid range (0.1-0.3 seconds)
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_RecordsMetrics_InvalidCredentials verifies that login delay
// metrics are recorded even when credentials are invalid (timing attack mitigation).
func TestLoginHandler_RecordsMetrics_InvalidCredentials(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)

	// User not found
	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(nil, identity.ErrUserNotFound).Once()

	// Metrics should still be recorded for timing attack mitigation
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
	assert.Nil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_RecordsMetrics_WrongPassword verifies that metrics are
// recorded when password is wrong.
func TestLoginHandler_RecordsMetrics_WrongPassword(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()

	// Metrics should be recorded even on wrong password
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   "WrongPassword123!",
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
	assert.Nil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_RecordsMetrics_AccountSuspended verifies that metrics are
// recorded even when account is suspended.
func TestLoginHandler_RecordsMetrics_AccountSuspended(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidSuspendedUser()

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()

	// Metrics should be recorded for suspended accounts
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.ErrorIs(t, err, appidentity.ErrAccountSuspended)
	assert.Nil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_RecordsMetrics_TokenGenerationError verifies that metrics
// are recorded even when token generation fails.
func TestLoginHandler_RecordsMetrics_TokenGenerationError(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()

	// Token generation fails
	suite.JWTService.On("GenerateAccessToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", fmt.Errorf("JWT secret not configured")).Once()

	// Metrics should still be recorded
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate access token")
	assert.Nil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_RecordsMetrics_SessionCreationError verifies that metrics
// are recorded even when session creation fails.
func TestLoginHandler_RecordsMetrics_SessionCreationError(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()
	suite.JWTService.On("GenerateAccessToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("access.token.jwt", nil).Once()
	suite.RefreshTokenService.On("GenerateToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("refresh.token.value", testhelpers.ValidRefreshTokenMetadata(), nil).Once()

	// Session creation fails
	suite.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(fmt.Errorf("redis connection failed")).Once()

	// Metrics should still be recorded
	suite.AuthMetrics.On("RecordLoginDelay", mock.MatchedBy(func(delay float64) bool {
		return delay >= 0.1 && delay <= 0.3
	})).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
	assert.Nil(t, result)
	suite.AuthMetrics.AssertExpectations(t)
}

// TestLoginHandler_WithNilMetrics_UsesNoOp verifies that passing nil metrics
// uses the NoOp implementation without panicking.
func TestLoginHandler_WithNilMetrics_UsesNoOp(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()
	suite.JWTService.On("GenerateAccessToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("access.token.jwt", nil).Once()
	suite.RefreshTokenService.On("GenerateToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("refresh.token.value", testhelpers.ValidRefreshTokenMetadata(), nil).Once()
	suite.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(nil).Once()
	suite.JWTService.On("GetTokenExpiration", mock.Anything).
		Return(time.Now().UTC().Add(15*time.Minute), nil).Once()

	// Pass nil metrics - should use NoOp
	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		nil, // nil metrics
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act - should not panic
	result, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestLoginHandler_MetricsRecordedExactlyOnce verifies that metrics are
// recorded exactly once per login attempt (not multiple times).
func TestLoginHandler_MetricsRecordedExactlyOnce(t *testing.T) {
	t.Parallel()

	// Arrange
	suite := testhelpers.NewTestSuite(t)
	email, _ := identity.NewEmail(testhelpers.ValidEmail)
	user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

	suite.UserRepo.On("FindByEmail", mock.Anything, email).
		Return(user, nil).Once()
	suite.JWTService.On("GenerateAccessToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("access.token.jwt", nil).Once()
	suite.RefreshTokenService.On("GenerateToken",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("refresh.token.value", testhelpers.ValidRefreshTokenMetadata(), nil).Once()
	suite.SessionStore.On("Create", mock.Anything, mock.Anything).
		Return(nil).Once()
	suite.JWTService.On("GetTokenExpiration", mock.Anything).
		Return(time.Now().UTC().Add(15*time.Minute), nil).Once()

	// Expect metrics to be recorded exactly once
	suite.AuthMetrics.On("RecordLoginDelay", mock.Anything).Once()

	handler := commands.NewLoginHandler(
		suite.UserRepo,
		suite.JWTService,
		suite.RefreshTokenService,
		suite.SessionStore,
		suite.AuthMetrics,
		&suite.Logger,
	)

	cmd := commands.LoginCommand{
		Identifier: testhelpers.ValidEmail,
		Password:   testhelpers.ValidPassword,
		IPAddress:  testhelpers.ValidIPAddress,
		UserAgent:  testhelpers.ValidUserAgent,
	}

	// Act
	_, _ = handler.Handle(context.Background(), cmd)

	// Assert - will fail if called more or less than once
	suite.AuthMetrics.AssertExpectations(t)
	suite.AuthMetrics.AssertNumberOfCalls(t, "RecordLoginDelay", 1)
}
