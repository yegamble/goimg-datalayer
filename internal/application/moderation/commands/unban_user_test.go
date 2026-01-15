package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestUnbanUserHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.UnbanUserCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error)
	}{
		{
			name: "successful unban of permanent ban",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t) // Permanent ban

				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.Equal(t, "revoked", result.Status)
				assert.NotEmpty(t, result.BanID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful unban of temporary ban",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidTemporaryBan(t, 24*time.Hour)

				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.Equal(t, "revoked", result.Status)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.UnbanUserCommand{
				UserID:    "invalid-uuid",
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid user id")
			},
		},
		{
			name: "invalid revoked by id",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: "invalid-uuid",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid revoked by id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid revoked by id")
			},
		},
		{
			name: "ban not found",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(nil, moderation.ErrBanNotFound).Once()
			},
			wantErr: "find ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "find ban")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "ban already revoked",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t)
				// Manually mark as revoked
				adminID := testhelpers.ValidAdminIDParsed()
				_ = ban.Revoke(adminID)
				ban.ClearEvents()

				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
			},
			wantErr: "revoke ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "revoke ban")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository save error",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t)

				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
			},
			wantErr: "save ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "save ban")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "event publish error does not fail operation",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t)

				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error")).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.NoError(t, err) // Operation succeeds even if event publishing fails
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "empty user id",
			cmd: commands.UnbanUserCommand{
				UserID:    "",
				RevokedBy: testhelpers.ValidAdminID,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "empty revoked by id",
			cmd: commands.UnbanUserCommand{
				UserID:    testhelpers.ValidUserID,
				RevokedBy: "",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid revoked by id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UnbanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := commands.NewUnbanUserHandler(
				suite.BanRepo,
				suite.EventPublisher,
				&suite.Logger,
			)

			result, err := handler.Handle(context.Background(), tt.cmd)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
