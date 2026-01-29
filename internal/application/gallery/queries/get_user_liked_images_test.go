package queries_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestGetUserLikedImagesHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval - user has liked images", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		image1 := testhelpers.ValidImage(t)
		image2 := testhelpers.ValidImage(t)
		imageID1, _ := uuid.Parse(image1.ID().String())
		imageID2, _ := uuid.Parse(image2.ID().String())
		imageIDs := []uuid.UUID{imageID1, imageID2}

		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(2), nil).Once()

		// Use mock.MatchedBy to match slice of IDs irrespective of how it was constructed
		mockImageRepo.On("FindByIDs", mock.Anything, mock.MatchedBy(func(ids []gallery.ImageID) bool {
			return len(ids) == 2 && ids[0].Equals(image1.ID()) && ids[1].Equals(image2.ID())
		})).Return([]*gallery.Image{image1, image2}, nil).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Pagination.Page())
		assert.Equal(t, 20, result.Pagination.PerPage())
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertExpectations(t)
	})

	t.Run("successful retrieval - user has no liked images", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return([]uuid.UUID{}, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(0), nil).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Images)
		assert.Equal(t, int64(0), result.Total)
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertNotCalled(t, "FindByIDs")
	})

	t.Run("pagination - multiple pages", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()

		// Create 10 images for first page
		imageIDs := make([]uuid.UUID, 10)
		images := make([]*gallery.Image, 10)
		for i := 0; i < 10; i++ {
			images[i] = testhelpers.ValidImage(t)
			id, _ := uuid.Parse(images[i].ID().String())
			imageIDs[i] = id
		}

		pagination, _ := shared.NewPagination(1, 10)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(35), nil).Once()

		mockImageRepo.On("FindByIDs", mock.Anything, mock.MatchedBy(func(ids []gallery.ImageID) bool {
			return len(ids) == 10
		})).Return(images, nil).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 10,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 10)
		assert.Equal(t, int64(35), result.Total)
		assert.Equal(t, 4, result.Pagination.TotalPages()) // 35 / 10 = 4 pages
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertExpectations(t)
	})

	t.Run("skips deleted images gracefully", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		image1 := testhelpers.ValidImage(t)
		deletedImageID := testhelpers.ValidImageIDParsed()
		image3 := testhelpers.ValidImage(t)
		imageID1, _ := uuid.Parse(image1.ID().String())
		imageID2, _ := uuid.Parse(deletedImageID.String())
		imageID3, _ := uuid.Parse(image3.ID().String())
		imageIDs := []uuid.UUID{imageID1, imageID2, imageID3}

		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(3), nil).Once()

		// FindByIDs returns only existing images (image1 and image3)
		mockImageRepo.On("FindByIDs", mock.Anything, mock.Anything).
			Return([]*gallery.Image{image1, image3}, nil).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 2)         // Only 2 images (deleted one skipped)
		assert.Equal(t, int64(3), result.Total) // Total still reflects all likes
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertExpectations(t)
	})

	t.Run("invalid user id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		query := queries.GetUserLikedImagesQuery{
			UserID:  "invalid-uuid",
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user id")
		assert.Nil(t, result)
	})

	t.Run("repository error - get liked image ids", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(nil, fmt.Errorf("database error")).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get liked image ids")
		assert.Nil(t, result)
		mockLikeRepo.AssertExpectations(t)
	})

	t.Run("repository error - count liked images", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		id, _ := uuid.Parse(testhelpers.ValidImageID)
		imageIDs := []uuid.UUID{id}

		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(0), fmt.Errorf("database error")).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count liked images")
		assert.Nil(t, result)
		mockLikeRepo.AssertExpectations(t)
	})

	t.Run("default pagination applied", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		image := testhelpers.ValidImage(t)
		id, _ := uuid.Parse(image.ID().String())
		imageIDs := []uuid.UUID{id}

		defaultPagination := shared.DefaultPagination()
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, defaultPagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(1), nil).Once()

		mockImageRepo.On("FindByIDs", mock.Anything, mock.MatchedBy(func(ids []gallery.ImageID) bool {
			return len(ids) == 1 && ids[0].Equals(image.ID())
		})).Return([]*gallery.Image{image}, nil).Once()

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    0, // Invalid, will use default
			PerPage: 0, // Invalid, will use default
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultPagination.Page(), result.Pagination.Page())
		assert.Equal(t, defaultPagination.PerPage(), result.Pagination.PerPage())
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertExpectations(t)
	})

	t.Run("returns error when image repository fails", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockLikeRepo := new(testhelpers.MockLikeRepository)
		mockImageRepo := new(testhelpers.MockImageRepository)
		handler := queries.NewGetUserLikedImagesHandler(mockLikeRepo, mockImageRepo)

		userID := testhelpers.ValidUserIDParsed()
		image1 := testhelpers.ValidImage(t)
		imageID1, _ := uuid.Parse(image1.ID().String())
		imageIDs := []uuid.UUID{imageID1}

		pagination, _ := shared.NewPagination(1, 20)
		mockLikeRepo.On("GetLikedImageIDs", mock.Anything, userID, pagination).
			Return(imageIDs, nil).Once()
		mockLikeRepo.On("CountLikedImagesByUser", mock.Anything, userID).
			Return(int64(1), nil).Once()

		mockImageRepo.On("FindByIDs", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("database error")).Once() // Repository error

		query := queries.GetUserLikedImagesQuery{
			UserID:  testhelpers.ValidUserID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find images by ids")
		assert.Nil(t, result)
		mockLikeRepo.AssertExpectations(t)
		mockImageRepo.AssertExpectations(t)
	})
}
