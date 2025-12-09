package commands_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestRegisterUserHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.RegisterUserCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error)
	}{
		{
			name: "successful registration",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				username, _ := identity.NewUsername(testhelpers.ValidUsername)

				// Email uniqueness check passes
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()

				// Username uniqueness check passes
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(nil, identity.ErrUserNotFound).Once()

				// Save succeeds
				suite.UserRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
					return u.Email().String() == testhelpers.ValidEmail &&
						u.Username().String() == testhelpers.ValidUsername
				})).Return(nil).Once()

				// Event publishing succeeds (non-critical)
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
					Return(nil).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				// Verify user was created
				suite.UserRepo.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			},
		},
		{
			name: "email already exists",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  "newusername",
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				existingUser := testhelpers.ValidUser()

				// Email already exists
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(existingUser, nil).Once()
			},
			wantErr: appidentity.ErrEmailAlreadyExists,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrEmailAlreadyExists)
				assert.Nil(t, result)
				// Save should not be called
				suite.UserRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
			},
		},
		{
			name: "username already exists",
			cmd: commands.RegisterUserCommand{
				Email:     "newemail@example.com",
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail("newemail@example.com")
				username, _ := identity.NewUsername(testhelpers.ValidUsername)
				existingUser := testhelpers.ValidUser()

				// Email check passes
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()

				// Username already exists
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(existingUser, nil).Once()
			},
			wantErr: appidentity.ErrUsernameAlreadyExists,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrUsernameAlreadyExists)
				assert.Nil(t, result)
				suite.UserRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
			},
		},
		{
			name: "invalid email format - missing @",
			cmd: commands.RegisterUserCommand{
				Email:     "notanemail",
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls should be made
			},
			wantErr: nil, // Wrapped error, so we check the message
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid email")
				assert.Nil(t, result)
				suite.UserRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything)
			},
		},
		{
			name: "invalid email format - empty",
			cmd: commands.RegisterUserCommand{
				Email:     "",
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid email")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid email format - disposable",
			cmd: commands.RegisterUserCommand{
				Email:     "user@mailinator.com",
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid email")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid username - too short",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  "ab",
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls - validation fails early
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid username")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid username - invalid characters",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  "user@name",
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls - validation fails early
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid username")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid username - reserved word",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  "admin",
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls - validation fails early
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid username")
				assert.Nil(t, result)
			},
		},
		{
			name: "weak password - too short",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  "Short1!",
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				username, _ := identity.NewUsername(testhelpers.ValidUsername)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid password")
				assert.Nil(t, result)
			},
		},
		{
			name: "repository save error",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				username, _ := identity.NewUsername(testhelpers.ValidUsername)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(nil, identity.ErrUserNotFound).Once()

				// Database error
				suite.UserRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database connection failed")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save user")
				assert.Nil(t, result)
			},
		},
		{
			name: "event publisher error - should still succeed",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				username, _ := identity.NewUsername(testhelpers.ValidUsername)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("Save", mock.Anything, mock.Anything).
					Return(nil).Once()

				// Event publishing fails (non-critical)
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
					Return(fmt.Errorf("event bus unavailable")).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				// Should still succeed even if event publishing fails
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
		{
			name: "email uniqueness check database error",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)

				// Database error during email check
				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, fmt.Errorf("database timeout")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "check email uniqueness")
				assert.Nil(t, result)
			},
		},
		{
			name: "username uniqueness check database error",
			cmd: commands.RegisterUserCommand{
				Email:     testhelpers.ValidEmail,
				Username:  testhelpers.ValidUsername,
				Password:  testhelpers.ValidPassword,
				IPAddress: testhelpers.ValidIPAddress,
				UserAgent: testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				email, _ := identity.NewEmail(testhelpers.ValidEmail)
				username, _ := identity.NewUsername(testhelpers.ValidUsername)

				suite.UserRepo.On("FindByEmail", mock.Anything, email).
					Return(nil, identity.ErrUserNotFound).Once()

				// Database error during username check
				suite.UserRepo.On("FindByUsername", mock.Anything, username).
					Return(nil, fmt.Errorf("database connection lost")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "check username uniqueness")
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

			handler := commands.NewRegisterUserHandler(
				suite.UserRepo,
				suite.EventPublisher,
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
