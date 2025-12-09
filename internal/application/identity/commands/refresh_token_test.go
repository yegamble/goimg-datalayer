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
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestRefreshTokenHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.RefreshTokenCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error)
	}{
		{
			name: "successful token refresh",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.refresh.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.UserID = user.ID().String()
				metadata.Used = false // Not used yet

				// Token validation succeeds
				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.refresh.token").
					Return(metadata, nil).Once()

				// Session exists
				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				// User lookup succeeds
				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				// No anomalies detected
				suite.RefreshTokenService.On("DetectAnomalies", metadata, testhelpers.ValidIPAddress, testhelpers.ValidUserAgent).
					Return(false).Once()

				// Mark old token as used
				suite.RefreshTokenService.On("MarkAsUsed", mock.Anything, "valid.refresh.token").
					Return(nil).Once()

				// Generate new access token
				suite.JWTService.On("GenerateAccessToken",
					user.ID().String(),
					user.Email().String(),
					string(user.Role()),
					metadata.SessionID,
				).Return("new.access.token", nil).Once()

				// Generate new refresh token
				newMetadata := testhelpers.ValidRefreshTokenMetadata()
				suite.RefreshTokenService.On("GenerateToken",
					mock.Anything,
					user.ID().String(),
					metadata.SessionID,
					metadata.FamilyID,
					metadata.TokenHash,
					testhelpers.ValidIPAddress,
					testhelpers.ValidUserAgent,
				).Return("new.refresh.token", newMetadata, nil).Once()

				// Update session
				suite.SessionStore.On("Create", mock.Anything, mock.Anything).
					Return(nil).Once()

				// Token expiration
				suite.JWTService.On("GetTokenExpiration", "new.access.token").
					Return(time.Now().UTC().Add(15*time.Minute), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				// Verify token was marked as used
				suite.RefreshTokenService.AssertCalled(t, "MarkAsUsed", mock.Anything, "valid.refresh.token")
			},
		},
		{
			name: "invalid token format",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "malformed.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Token validation fails
				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "malformed.token").
					Return(nil, fmt.Errorf("invalid token format")).Once()
			},
			wantErr: appidentity.ErrInvalidToken,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, appidentity.ErrInvalidToken)
				assert.Nil(t, result)
			},
		},
		{
			name: "expired refresh token",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "expired.refresh.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Expired metadata
				metadata := testhelpers.ExpiredRefreshTokenMetadata()

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "expired.refresh.token").
					Return(metadata, nil).Once()
			},
			wantErr: appidentity.ErrTokenExpired,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrTokenExpired)
				assert.Nil(t, result)
			},
		},
		{
			name: "token replay detected - revokes family",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "already.used.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.Used = true // Token already used

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "already.used.token").
					Return(metadata, nil).Once()

				// Should revoke entire token family
				suite.RefreshTokenService.On("RevokeFamily", mock.Anything, metadata.FamilyID).
					Return(nil).Once()

				// Should revoke session
				suite.SessionStore.On("Revoke", mock.Anything, metadata.SessionID).
					Return(nil).Once()
			},
			wantErr: appidentity.ErrTokenReplayDetected,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrTokenReplayDetected)
				assert.Nil(t, result)
				// Verify family was revoked
				suite.RefreshTokenService.AssertCalled(t, "RevokeFamily", mock.Anything, mock.Anything)
				suite.SessionStore.AssertCalled(t, "Revoke", mock.Anything, mock.Anything)
			},
		},
		{
			name: "session not found",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token.no.session",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token.no.session").
					Return(metadata, nil).Once()

				// Session doesn't exist
				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(false, nil).Once()
			},
			wantErr: appidentity.ErrSessionNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrSessionNotFound)
				assert.Nil(t, result)
			},
		},
		{
			name: "user not found",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token.deleted.user",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token.deleted.user").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				// User not found
				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: appidentity.ErrInvalidToken,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrInvalidToken)
				assert.Nil(t, result)
			},
		},
		{
			name: "account suspended during refresh",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token.suspended.user",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.Used = false
				user := testhelpers.ValidSuspendedUser()

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token.suspended.user").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				// Should revoke session
				suite.SessionStore.On("Revoke", mock.Anything, metadata.SessionID).
					Return(nil).Maybe()
			},
			wantErr: appidentity.ErrAccountSuspended,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.ErrorIs(t, err, appidentity.ErrAccountSuspended)
				assert.Nil(t, result)
			},
		},
		{
			name: "anomaly detected - IP change - continues with warning",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token",
				IPAddress:    "10.0.0.1", // Different IP
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.UserID = user.ID().String()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				// Anomaly detected (different IP)
				suite.RefreshTokenService.On("DetectAnomalies", metadata, "10.0.0.1", testhelpers.ValidUserAgent).
					Return(true).Once()

				// Still continues with refresh
				suite.RefreshTokenService.On("MarkAsUsed", mock.Anything, "valid.token").
					Return(nil).Once()

				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("new.access.token", nil).Once()

				newMetadata := testhelpers.ValidRefreshTokenMetadata()
				suite.RefreshTokenService.On("GenerateToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, "10.0.0.1", mock.Anything).
					Return("new.refresh.token", newMetadata, nil).Once()

				suite.SessionStore.On("Create", mock.Anything, mock.Anything).
					Return(nil).Once()

				suite.JWTService.On("GetTokenExpiration", mock.Anything).
					Return(time.Now().UTC().Add(15*time.Minute), nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				// Should succeed despite anomaly detection
				require.NoError(t, err)
				require.NotNil(t, result)
			},
		},
		{
			name: "mark as used fails",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.UserID = user.ID().String()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				suite.RefreshTokenService.On("DetectAnomalies", mock.Anything, mock.Anything, mock.Anything).
					Return(false).Once()

				// MarkAsUsed fails
				suite.RefreshTokenService.On("MarkAsUsed", mock.Anything, "valid.token").
					Return(fmt.Errorf("redis write failed")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "mark token as used")
				assert.Nil(t, result)
			},
		},
		{
			name: "new access token generation fails",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.UserID = user.ID().String()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				suite.RefreshTokenService.On("DetectAnomalies", mock.Anything, mock.Anything, mock.Anything).
					Return(false).Once()

				suite.RefreshTokenService.On("MarkAsUsed", mock.Anything, "valid.token").
					Return(nil).Once()

				// Access token generation fails
				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("JWT signing failed")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "generate access token")
				assert.Nil(t, result)
			},
		},
		{
			name: "new refresh token generation fails",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				user := testhelpers.ValidActiveUser()
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.UserID = user.ID().String()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()

				userID, _ := identity.ParseUserID(metadata.UserID)
				suite.UserRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil).Once()

				suite.RefreshTokenService.On("DetectAnomalies", mock.Anything, mock.Anything, mock.Anything).
					Return(false).Once()

				suite.RefreshTokenService.On("MarkAsUsed", mock.Anything, "valid.token").
					Return(nil).Once()

				suite.JWTService.On("GenerateAccessToken",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("new.access.token", nil).Once()

				// New refresh token generation fails
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
			name: "session check error",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "valid.token",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := testhelpers.ValidRefreshTokenMetadata()
				metadata.Used = false

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "valid.token").
					Return(metadata, nil).Once()

				// Session check fails
				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(false, fmt.Errorf("redis connection failed")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "check session existence")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid user ID in metadata",
			cmd: commands.RefreshTokenCommand{
				RefreshToken: "token.with.bad.userid",
				IPAddress:    testhelpers.ValidIPAddress,
				UserAgent:    testhelpers.ValidUserAgent,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				metadata := &services.RefreshTokenMetadata{
					TokenHash:  "hash",
					UserID:     "not-a-valid-uuid",
					SessionID:  testhelpers.ValidSessionID.String(),
					FamilyID:   testhelpers.ValidFamilyID,
					IssuedAt:   time.Now().UTC(),
					ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour),
					IP:         testhelpers.ValidIPAddress,
					UserAgent:  testhelpers.ValidUserAgent,
					ParentHash: "",
					Used:       false,
				}

				suite.RefreshTokenService.On("ValidateToken", mock.Anything, "token.with.bad.userid").
					Return(metadata, nil).Once()

				suite.SessionStore.On("Exists", mock.Anything, metadata.SessionID).
					Return(true, nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result interface{}, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user ID")
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

			handler := commands.NewRefreshTokenHandler(
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
