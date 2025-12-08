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

func TestUpdateImageHandler_Handle(t *testing.T) {
	t.Parallel()

	newTitle := "Updated Title"
	newDescription := "Updated Description"
	newVisibility := "private"

	tests := []struct {
		name    string
		cmd     commands.UpdateImageCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error)
	}{
		{
			name: "successful update - all fields",
			cmd: commands.UpdateImageCommand{
				ImageID:     testhelpers.ValidImageID,
				UserID:      testhelpers.ValidUserID,
				Title:       &newTitle,
				Description: &newDescription,
				Visibility:  &newVisibility,
				Tags:        []string{"updated", "tags"},
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ImageID)
				assert.Equal(t, 2, result.TagsAdded)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful update - title only",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
				suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "no fields updated",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
				// No fields to update
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Contains(t, result.Message, "No changes made")
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid image id",
			cmd: commands.UpdateImageCommand{
				ImageID: "invalid-uuid",
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid user id",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  "invalid-uuid",
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "image not found",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: gallery.ErrImageNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrImageNotFound)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - not owner",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid visibility",
			cmd: commands.UpdateImageCommand{
				ImageID:    testhelpers.ValidImageID,
				UserID:     testhelpers.ValidUserID,
				Visibility: stringPtr("invalid"),
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrInvalidVisibility,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, gallery.ErrInvalidVisibility)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid tag in update",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
				Tags:    []string{"a"}, // Too short
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrTagTooShort,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrTagTooShort)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "repository save failure",
			cmd: commands.UpdateImageCommand{
				ImageID: testhelpers.ValidImageID,
				UserID:  testhelpers.ValidUserID,
				Title:   &newTitle,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *commands.UpdateImageResult, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "save image")
				assert.Nil(t, result)
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

			handler := commands.NewUpdateImageHandler(
				suite.ImageRepo,
				suite.EventPublisher,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.cmd)

			// Assert
			if tt.assert != nil {
				tt.assert(t, suite, result, err)
			} else if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// TestUpdateImageCommand_Interface verifies the command implements the interface.

// Helper function to get string pointer
func stringPtr(s string) *string {
	return &s
}
