package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestGetUserBanStatusHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.GetUserBanStatusQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr string
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error)
	}{
		{
			name: "user is not banned",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(false, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.False(t, result.IsBanned)
				assert.Nil(t, result.BanID)
				assert.Nil(t, result.Reason)
				assert.Nil(t, result.BannedBy)
				assert.Nil(t, result.IsPermanent)
				assert.Nil(t, result.ExpiresAt)
				assert.Nil(t, result.CreatedAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "user has permanent ban",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t) // Permanent ban

				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(true, nil).Once()
				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidUserID, result.UserID)
				assert.True(t, result.IsBanned)
				assert.NotNil(t, result.BanID)
				assert.NotNil(t, result.Reason)
				assert.Equal(t, testhelpers.ValidBanReason, *result.Reason)
				assert.NotNil(t, result.BannedBy)
				assert.Equal(t, testhelpers.ValidAdminID, *result.BannedBy)
				assert.NotNil(t, result.IsPermanent)
				assert.True(t, *result.IsPermanent)
				assert.Nil(t, result.ExpiresAt) // Permanent ban has no expiration
				assert.NotNil(t, result.CreatedAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "user has temporary ban",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidTemporaryBan(t, 24*time.Hour)

				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(true, nil).Once()
				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.IsBanned)
				assert.NotNil(t, result.IsPermanent)
				assert.False(t, *result.IsPermanent)
				assert.NotNil(t, result.ExpiresAt)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid user id",
			query: queries.GetUserBanStatusQuery{
				UserID: "invalid-uuid",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid user id")
			},
		},
		{
			name: "empty user id",
			query: queries.GetUserBanStatusQuery{
				UserID: "",
			},
			setup:   func(t *testing.T, suite *testhelpers.TestSuite) {},
			wantErr: "invalid user id",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "invalid user id")
			},
		},
		{
			name: "repository error checking ban status",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(false, errors.New("database error")).Once()
			},
			wantErr: "check user ban status",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "check user ban status")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "user is banned but ban details not found returns banned without details",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(true, nil).Once()
				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(nil, moderation.ErrBanNotFound).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.IsBanned)
				// No details available due to FindByUserID error
				assert.Nil(t, result.BanID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "user has expired/revoked ban shows not banned",
			query: queries.GetUserBanStatusQuery{
				UserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				userID := testhelpers.ValidUserIDParsed()
				ban := testhelpers.ValidBan(t)
				// Revoke the ban to make it inactive
				adminID := testhelpers.ValidAdminIDParsed()
				_ = ban.Revoke(adminID)
				ban.ClearEvents()

				suite.BanRepo.On("IsUserBanned", mock.Anything, userID).Return(true, nil).Once()
				suite.BanRepo.On("FindByUserID", mock.Anything, userID).Return(ban, nil).Once()
			},
			wantErr: "",
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.UserBanStatusResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				// Ban exists but is not active, so should show not banned
				assert.False(t, result.IsBanned)
				suite.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := testhelpers.NewTestSuite(t)
			tt.setup(t, suite)

			handler := queries.NewGetUserBanStatusHandler(suite.BanRepo)
			result, err := handler.Handle(context.Background(), tt.query)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			tt.assert(t, suite, result, err)
		})
	}
}
