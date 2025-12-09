package commands_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestLoginHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.LoginCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error)
	}{
		{
			name: "successful login with email",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)
				sessionID := uuid.New().String()

				// User lookup succeeds
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()

				// JWT generation succeeds
				suite.JWTService.On("GenerateAccessToken",
					user.ID().String(),
					user.Email().String(),
					string(user.Role()),
					mock.AnythingOfType("string"),
				).Return("access.token.jwt", nil).Once()

				// Refresh token generation
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.SessionID = sessionID
				suite.RefreshTokenService.On("GenerateToken",
					mock.Anything,
					user.ID().String(),
					mock.AnythingOfType("string"), // sessionID
					mock.AnythingOfType("string"), // familyID
					"",                            // parentHash
					testhelpers.ValidIPAddress,
					testhelpers.ValidUserAgent,
				).Return("refresh.token.value", metadata, nil).Once()

				// Session creation succeeds
				suite.SessionStore.On("Create", mock.Anything, mock.Anything).
					Return(nil).Once()

				// Token expiration extraction
				suite.JWTService.On("GetTokenExpiration", "access.token.jwt").
					Return(time.Now().UTC().Add(15*time.Minute), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				// Verify all services were called
				suite.UserRepo.AssertCalled(t, "FindByEmail", mock.Anything, mock.Anything)
				suite.JWTService.AssertCalled(t, "GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				suite.SessionStore.AssertCalled(t, "Create", mock.Anything, mock.Anything)
			},
		},
		{
			name: "successful login with username",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidUsername,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				username, _ := identity.NewUsername(testhelpers.ValidUsername)
				user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

				// User lookup by username succeeds
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(user, nil).Once()

				// JWT generation
				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("access.token.jwt", nil).Once()

				// Refresh token generation
				suite.RefreshTokenService.On("GenerateToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("refresh.token.value", testhelpers.ValidRefreshTokenMetadata(), nil).Once()

				// Session creation
				suite.SessionStore.On("Create", mock.Anything, mock.Anything).
					Return(nil).Once()

				suite.JWTService.On("GetTokenExpiration", mock.Anything).
					Return(time.Now().UTC().Add(15*time.Minute), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
		{
			name: "user not found - returns generic error",
			cmd: commands.LoginCommand{
				Identifier: "nonexistent@example.com",
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail("nonexistent@example.com")

				// User not found
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: appidentity.ErrInvalidCredentials,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
				assert.Nil(t, result)
				// No token generation should happen
				suite.JWTService.AssertNotCalled(t, "GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "wrong password - returns generic error",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   "WrongPassword123!",
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				// User exists with different password
				user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()
			},
			wantErr: appidentity.ErrInvalidCredentials,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
				assert.Nil(t, result)
				// No token generation should happen
				suite.JWTService.AssertNotCalled(t, "GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "account suspended",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				// User is suspended
				user := testhelpers.ValidSuspendedUser()

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()
			},
			wantErr: appidentity.ErrAccountSuspended,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrAccountSuspended)
				assert.Nil(t, result)
				// No token generation should happen
				suite.JWTService.AssertNotCalled(t, "GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "account deleted - returns generic error",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				// Use deleted user fixture
				user := testhelpers.ValidDeletedUser()

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()
			},
			wantErr: appidentity.ErrAccountDeleted,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrAccountDeleted)
				assert.Nil(t, result)
			},
		},
		{
			name: "session creation error",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
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
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "create session")
				assert.Nil(t, result)
			},
		},
		{
			name: "token generation error",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()

				// Token generation fails
				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("JWT secret not configured")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "generate access token")
				assert.Nil(t, result)
			},
		},
		{
			name: "refresh token generation error",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				user := testhelpers.ValidUserWithPassword(testhelpers.ValidPassword)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(user, nil).Once()

				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("access.token.jwt", nil).Once()

				// Refresh token generation fails
				suite.RefreshTokenService.On("GenerateToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", nil, fmt.Errorf("redis unavailable")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "generate refresh token")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid identifier format",
			cmd: commands.LoginCommand{
				Identifier: "not-email-or-username!@#$%",
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls - identifier validation fails
			},
			wantErr: appidentity.ErrInvalidCredentials,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
				assert.Nil(t, result)
			},
		},
		{
			name: "database error during user lookup",
			cmd: commands.LoginCommand{
				Identifier: testhelpers.ValidEmail,
				Password:   testhelpers.ValidPassword,
				IPAddress:  testhelpers.ValidIPAddress,
				UserAgent:  testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)

				// Database error
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, fmt.Errorf("database connection timeout")).Once()
			},
			wantErr: appidentity.ErrInvalidCredentials,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				// Handler returns generic error to prevent user enumeration
				require.ErrorIs(t, err, appidentity.ErrInvalidCredentials)
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			suite := testhelpers.NewTestSuite(t)
			if tt.setup != nil {
				tt.setup(t, suite)
			}

			handler := commands.NewLoginHandler(
				suite.UserRepo,
				suite.JWTService,
				suite.RefreshTokenService,
				suite.SessionStore,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			switch {
			case tt.assert != nil:
				tt.assert(t, suite, result, err)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			default:
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			suite.AssertExpectations()
		})
	}
}
