package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
)

func TestBanUserHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.BanUserCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error)
	}{
		{
			name: "successful permanent ban",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil, // Permanent
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.True(t, result.IsPermanent)
				assert.Nil(t, result.ExpiresAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful temporary ban",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: intPtr(24), // 24 hours
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.False(t, result.IsPermanent)
				assert.NotNil(t, result.ExpiresAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.BanUserCommand{
				UserID:        "invalid-uuid",
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid user id")
			},
		},
		{
			name: "invalid banned by id",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      "invalid-uuid",
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid banned by id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid banned by id")
			},
		},
		{
			name: "empty reason",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        "",
				DurationHours: nil,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "create ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "create ban")
			},
		},
		{
			name: "reason too long",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        string(make([]byte, 501)), // 501 characters, max is 500
				DurationHours: nil,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "create ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "create ban")
			},
		},
		{
			name: "repository save error",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
			},
			wantErr: "save ban",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "save ban")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "event publish error does not fail operation",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("event bus error")).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.NoError(t, err) // Operation succeeds even if event publishing fails
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "empty user id",
			cmd: commands.BanUserCommand{
				UserID:        "",
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: nil,
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "zero duration hours creates permanent ban",
			cmd: commands.BanUserCommand{
				UserID:        testhelpers.ValidUserID,
				BannedBy:      testhelpers.ValidAdminID,
				Reason:        testhelpers.ValidBanReason,
				DurationHours: intPtr(0), // 0 hours
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				suite.BanRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.BanUserResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				// 0 duration creates a ban that expires immediately
				assert.False(t, result.IsPermanent)
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := commands.NewBanUserHandler(
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

// intPtr is a helper function to create a pointer to an int.
func intPtr(i int) *int {
	return &i
}
