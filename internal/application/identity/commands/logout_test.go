//nolint:goconst // Test strings don't need constants
package commands_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestLogoutHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.LogoutCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, err error)
	}{
		{
			name: "successful single session logout",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				// Extract token ID
				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				// Get token expiration
				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				// Blacklist access token
				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(nil).Once()

				// Revoke refresh token
				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				// Revoke session
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.NoError(t, err)
				// Verify all cleanup steps were performed
				suite.TokenBlacklist.AssertCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
				suite.RefreshTokenService.AssertCalled(t, "RevokeToken", mock.Anything, "valid.refresh.token")
				suite.SessionStore.AssertCalled(t, "Revoke", mock.Anything, testhelpers.ValidSessionID.String())
			},
		},
		{
			name: "successful logout all sessions",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "",
				RefreshToken: "",
				LogoutAll:    true,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()

				// Find user
				suite.UserRepo.On("FindByID", mock.Anything, testhelpers.ValidUserID).
					Return(user, nil).Once()

				// Get all user sessions
				sessions := []*services.Session{
					{
						SessionID: "session1",
						UserID:    testhelpers.ValidUserID.String(),
					},
					{
						SessionID: "session2",
						UserID:    testhelpers.ValidUserID.String(),
					},
				}
				suite.SessionStore.On("GetUserSessions", mock.Anything, testhelpers.ValidUserID.String()).
					Return(sessions, nil).Once()

				// Revoke all sessions
				suite.SessionStore.On("RevokeAll", mock.Anything, testhelpers.ValidUserID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.NoError(t, err)
				suite.SessionStore.AssertCalled(t, "RevokeAll", mock.Anything, testhelpers.ValidUserID.String())
			},
		},
		{
			name: "logout succeeds even if token extraction fails",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "malformed.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Token extraction fails (non-critical)
				suite.JWTService.On("ExtractTokenID", "malformed.token").
					Return("", fmt.Errorf("malformed token")).Once()

				// Refresh token revocation proceeds
				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				// Session revocation proceeds
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				// Should still succeed even if token extraction fails
				require.NoError(t, err)
				suite.SessionStore.AssertCalled(t, "Revoke", mock.Anything, testhelpers.ValidSessionID.String())
			},
		},
		{
			name: "logout succeeds even if session not found - idempotent",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(nil).Once()

				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				// Session doesn't exist (already revoked) - should be idempotent
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "logout continues even if blacklist fails",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				// Blacklist fails (non-critical)
				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(fmt.Errorf("redis connection failed")).Once()

				// Other operations continue
				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				// Should still succeed
				require.NoError(t, err)
			},
		},
		{
			name: "logout continues even if refresh token revocation fails",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(nil).Once()

				// Refresh token revocation fails (non-critical)
				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(fmt.Errorf("token not found")).Once()

				// Session revocation continues
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				// Should still succeed
				require.NoError(t, err)
			},
		},
		{
			name: "logout continues even if session revocation fails",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(nil).Once()

				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				// Session revocation fails (non-critical)
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(fmt.Errorf("redis connection lost")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				// Should still succeed
				require.NoError(t, err)
			},
		},
		{
			name: "logout with empty refresh token",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "", // No refresh token provided
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"
				expiresAt := time.Now().UTC().Add(15 * time.Minute)

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(expiresAt, nil).Once()

				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, expiresAt).
					Return(nil).Once()

				// Refresh token revocation should NOT be called
				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.NoError(t, err)
				// Verify refresh token revocation was not called
				suite.RefreshTokenService.AssertNotCalled(t, "RevokeToken", mock.Anything, mock.Anything)
			},
		},
		{
			name: "invalid user ID",
			cmd: commands.LogoutCommand{
				UserID:       "not-a-valid-uuid",
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No calls should be made
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user ID")
			},
		},
		{
			name: "logout all - user not found",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "",
				RefreshToken: "",
				LogoutAll:    true,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// User not found
				suite.UserRepo.On("FindByID", mock.Anything, testhelpers.ValidUserID).
					Return(nil, fmt.Errorf("user not found")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "load user")
			},
		},
		{
			name: "logout all - get sessions error",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "",
				RefreshToken: "",
				LogoutAll:    true,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()

				suite.UserRepo.On("FindByID", mock.Anything, testhelpers.ValidUserID).
					Return(user, nil).Once()

				// Get sessions fails
				suite.SessionStore.On("GetUserSessions", mock.Anything, testhelpers.ValidUserID.String()).
					Return(nil, fmt.Errorf("redis connection failed")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "get user sessions")
			},
		},
		{
			name: "logout all - revoke all error",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "",
				RefreshToken: "",
				LogoutAll:    true,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()

				suite.UserRepo.On("FindByID", mock.Anything, testhelpers.ValidUserID).
					Return(user, nil).Once()

				sessions := []*services.Session{
					{SessionID: "session1"},
				}
				suite.SessionStore.On("GetUserSessions", mock.Anything, testhelpers.ValidUserID.String()).
					Return(sessions, nil).Once()

				// Revoke all fails
				suite.SessionStore.On("RevokeAll", mock.Anything, testhelpers.ValidUserID.String()).
					Return(fmt.Errorf("redis unavailable")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "revoke all sessions")
			},
		},
		{
			name: "token expiration extraction fails - uses default TTL",
			cmd: commands.LogoutCommand{
				UserID:       testhelpers.ValidUserID.String(),
				SessionID:    testhelpers.ValidSessionID.String(),
				AccessToken:  "valid.access.token",
				RefreshToken: "valid.refresh.token",
				LogoutAll:    false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				tokenID := "token-jti-123"

				suite.JWTService.On("ExtractTokenID", "valid.access.token").
					Return(tokenID, nil).Once()

				// Get token expiration fails
				suite.JWTService.On("GetTokenExpiration", "valid.access.token").
					Return(time.Time{}, fmt.Errorf("invalid token")).Once()

				// Should use default 15 min TTL
				suite.TokenBlacklist.On("Add", mock.Anything, tokenID, mock.MatchedBy(func(t time.Time) bool {
					// Should be approximately 15 minutes from now
					diff := t.Sub(time.Now().UTC())
					return diff >= 14*time.Minute && diff <= 16*time.Minute
				})).Return(nil).Once()

				suite.RefreshTokenService.On("RevokeToken", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				suite.SessionStore.On("Revoke", mock.Anything, testhelpers.ValidSessionID.String()).
					Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, err error) {
				// Should still succeed with default TTL
				require.NoError(t, err)
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

			handler := commands.NewLogoutHandler(
				suite.UserRepo,
				suite.JWTService,
				suite.RefreshTokenService,
				suite.SessionStore,
				suite.TokenBlacklist,
				&suite.Logger,
			)

			// Act
			err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			switch {
			case tt.assert != nil:
				tt.assert(t, suite, err)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			default:
				require.NoError(t, err)
			}

			suite.AssertExpectations()
		})
	}
}
