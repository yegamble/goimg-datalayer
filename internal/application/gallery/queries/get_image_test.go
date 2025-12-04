package queries_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestGetImageHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   queries.GetImageQuery
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		wantErr error
		assert  func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error)
	}{
		{
			name: "successful retrieval - public image",
			query: queries.GetImageQuery{
				ImageID:           testhelpers.ValidImageID,
				RequestingUserID:  "", // Anonymous
				IncrementViewOnly: false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ID)
				assert.Equal(t, "public", result.Visibility)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "successful retrieval - owner viewing own private image",
			query: queries.GetImageQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID, // Owner
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				// Make it private
				require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, testhelpers.ValidImageID, result.ID)
				assert.Equal(t, "private", result.Visibility)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "unauthorized - private image, not owner",
			query: queries.GetImageQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "550e8400-e29b-41d4-a716-446655440001", // Different user
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				// Make it private
				require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrUnauthorizedAccess,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "increment view count",
			query: queries.GetImageQuery{
				ImageID:           testhelpers.ValidImageID,
				RequestingUserID:  "",   // Anonymous
				IncrementViewOnly: true, // Only increment views
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "invalid image id",
			query: queries.GetImageQuery{
				ImageID:          "invalid-uuid",
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image id")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid requesting user id",
			query: queries.GetImageQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: "invalid-uuid",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No mocks - should fail validation
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid requesting user id")
				assert.Nil(t, result)
			},
		},
		{
			name: "image not found",
			query: queries.GetImageQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				imageID := testhelpers.ValidImageIDParsed()
				suite.ImageRepo.On("FindByID", mock.Anything, imageID).
					Return(nil, gallery.ErrImageNotFound).Once()
			},
			wantErr: gallery.ErrImageNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrImageNotFound)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "image not viewable - deleted",
			query: queries.GetImageQuery{
				ImageID:          testhelpers.ValidImageID,
				RequestingUserID: testhelpers.ValidUserID,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				// Mark as deleted
				require.NoError(t, image.MarkAsDeleted())
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
			},
			wantErr: gallery.ErrImageNotFound,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, gallery.ErrImageNotFound)
				assert.Nil(t, result)
				suite.AssertExpectations(t)
			},
		},
		{
			name: "view count save failure - should not fail query",
			query: queries.GetImageQuery{
				ImageID:           testhelpers.ValidImageID,
				RequestingUserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
				IncrementViewOnly: false,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				image := testhelpers.ValidImage(t)
				imageID := testhelpers.ValidImageIDParsed()

				suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()
				// Save fails but query should still succeed
				suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error")).Maybe()
			},
			wantErr: nil,
			assert: func(t *testing.T, suite *testhelpers.TestSuite, result *queries.ImageDTO, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
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

			handler := queries.NewGetImageHandler(
				suite.ImageRepo,
				&suite.Logger,
			)

			// Act
			result, err := handler.Handle(context.Background(), tt.query)

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

// TestGetImageQuery_Interface verifies the query implements the interface.

// TestImageToDTO tests the DTO conversion function.
func TestImageToDTO(t *testing.T) {
	t.Parallel()

	image := testhelpers.ValidImage(t)

	// Add some tags
	tag1 := testhelpers.ValidTag(t, "nature")
	tag2 := testhelpers.ValidTag(t, "landscape")
	require.NoError(t, image.AddTag(tag1))
	require.NoError(t, image.AddTag(tag2))

	// Convert to DTO using the handler's internal logic via a query
	suite := testhelpers.NewTestSuite(t)
	imageID := testhelpers.ValidImageIDParsed()
	suite.ImageRepo.On("FindByID", mock.Anything, imageID).Return(image, nil).Once()

	handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)
	dto, err := handler.Handle(context.Background(), queries.GetImageQuery{
		ImageID: image.ID().String(),
	})

	require.NoError(t, err)
	require.NotNil(t, dto)

	// Verify DTO fields
	assert.Equal(t, image.ID().String(), dto.ID)
	assert.Equal(t, image.OwnerID().String(), dto.OwnerID)
	assert.Equal(t, image.Metadata().Title(), dto.Title)
	assert.Equal(t, image.Visibility().String(), dto.Visibility)
	assert.Equal(t, image.Status().String(), dto.Status)
	assert.Len(t, dto.Tags, 2)
}
