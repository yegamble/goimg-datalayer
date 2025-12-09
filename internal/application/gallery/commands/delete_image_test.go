package commands_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestDeleteImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     commands.DeleteImageCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error)
	}{
		{
			name: "successful deletion by owner",
			cmd: commands.DeleteImageCommand{
				ImageID:  testhelpers.ValidImageID,
				UserID:   testhelpers.ValidUserID,
				UserRole: "user",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
				suite.JobEnqueuer.On("EnqueueImageCleanup", mock.Anything,
					imageID.String(), mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.Equal(t, "enqueued", result.CleanupJobID)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful deletion by moderator",
			cmd: commands.DeleteImageCommand{
				ImageID:  testhelpers.ValidImageID,
				UserID:   "550e8400-e29b-41d4-a716-446655440001", // Different user
				UserRole: "moderator",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
				suite.JobEnqueuer.On("EnqueueImageCleanup", mock.Anything,
					imageID.String(), mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful deletion by admin",
			cmd: commands.DeleteImageCommand{
				ImageID:  testhelpers.ValidImageID,
				UserID:   "550e8400-e29b-41d4-a716-446655440001", // Different user
				UserRole: "admin",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
				suite.JobEnqueuer.On("EnqueueImageCleanup", mock.Anything,
					imageID.String(), mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid image id",
			cmd: commands.DeleteImageCommand{
				ImageID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, _ *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.DeleteImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  "invalid-uuid",
			},
			setup: func(t *testing.T, _ *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "image not found",
			cmd: commands.DeleteImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: gallery.ErrImageNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrImageNotFound)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - not owner and not moderator",
			cmd: commands.DeleteImageCommand{
				ImageID:  testhelpers.ValidImageID,
				UserID:   "550e8400-e29b-41d4-a716-446655440001", // Different user
				UserRole: "user",                                 // Not a moderator
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository save failure",
			cmd: commands.DeleteImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save image")
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "cleanup job enqueue failure - should still succeed",
			cmd: commands.DeleteImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				// Cleanup job fails (non-critical)
				suite.JobEnqueuer.On("EnqueueImageCleanup", mock.Anything,
					imageID.String(), mock.Anything, mock.Anything).
					Return(fmt.Errorf("queue unavailable")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.DeleteImageResult, err error) {
				// Should still succeed
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, "failed", result.CleanupJobID)
				suite.AssertExpectations(t)
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

			handler := commands.NewDeleteImageHandler(
				suite.ImageRepo,
				suite.JobEnqueuer,
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
		})
	}
}

// TestDeleteImageCommand_Interface verifies the command implements the interface.
