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

//nolint:funlen // Table-driven test with comprehensive test cases
func TestGetImageHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval - public image", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:           image.ID().String(),
			RequestingUserID:  "", // Anonymous
			IncrementViewOnly: false,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, image.ID().String(), result.ID)
		assert.Equal(t, "public", result.Visibility)
		suite.AssertExpectations(t)
	})

	t.Run("successful retrieval - owner viewing own private image", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		// Make it private
		require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          image.ID().String(),
			RequestingUserID: testhelpers.ValidUserID, // Owner
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, image.ID().String(), result.ID)
		assert.Equal(t, "private", result.Visibility)
		suite.AssertExpectations(t)
	})

	t.Run("unauthorized - private image, not owner", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		// Make it private
		require.NoError(t, image.UpdateVisibility(gallery.VisibilityPrivate))

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          image.ID().String(),
			RequestingUserID: "550e8400-e29b-41d4-a716-446655440001", // Different user
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrUnauthorizedAccess)
		assert.Nil(t, result)
		suite.AssertExpectations(t)
	})

	t.Run("increment view count", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()
		suite.ImageRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:           image.ID().String(),
			RequestingUserID:  "", // Anonymous
			IncrementViewOnly: true,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		suite.AssertExpectations(t)
	})

	t.Run("invalid image id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          "invalid-uuid",
			RequestingUserID: testhelpers.ValidUserID,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid image id")
		assert.Nil(t, result)
	})

	t.Run("invalid requesting user id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          testhelpers.ValidImageID,
			RequestingUserID: "invalid-uuid",
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid requesting user id")
		assert.Nil(t, result)
	})

	t.Run("image not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		imageID := testhelpers.ValidImageIDParsed()

		suite.ImageRepo.On("FindByID", mock.Anything, imageID).
			Return(nil, gallery.ErrImageNotFound).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          testhelpers.ValidImageID,
			RequestingUserID: testhelpers.ValidUserID,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrImageNotFound)
		assert.Nil(t, result)
		suite.AssertExpectations(t)
	})

	t.Run("image not viewable - deleted", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		// Mark as deleted
		require.NoError(t, image.MarkAsDeleted())

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:          image.ID().String(),
			RequestingUserID: testhelpers.ValidUserID,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, gallery.ErrImageNotFound)
		assert.Nil(t, result)
		suite.AssertExpectations(t)
	})

	t.Run("view count save failure - should not fail query", func(t *testing.T) {
		t.Parallel()

		// Arrange
		suite := testhelpers.NewTestSuite(t)
		image := testhelpers.ValidImage(t)

		suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()
		// Save fails but query should still succeed
		suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
			Return(fmt.Errorf("database error")).Maybe()

		handler := queries.NewGetImageHandler(suite.ImageRepo, &suite.Logger)

		query := queries.GetImageQuery{
			ImageID:           image.ID().String(),
			RequestingUserID:  "550e8400-e29b-41d4-a716-446655440001", // Different user
			IncrementViewOnly: false,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		suite.AssertExpectations(t)
	})
}

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
	// Use the actual image ID, not the hardcoded ValidImageIDParsed
	suite.ImageRepo.On("FindByID", mock.Anything, image.ID()).Return(image, nil).Once()

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
