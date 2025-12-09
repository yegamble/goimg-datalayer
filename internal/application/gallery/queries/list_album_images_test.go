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
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestListAlbumImagesHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful list - public album with images", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		images := []*gallery.Image{
			testhelpers.ValidImage(t),
			testhelpers.ValidImage(t),
		}

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), pagination).
			Return(images, int64(2), nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "", // Anonymous can view public album
			Page:             1,
			PerPage:          20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 2)
		assert.Equal(t, int64(2), result.TotalCount)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		assert.Equal(t, 1, result.TotalPages)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})

	t.Run("successful list - owner viewing private album", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityPrivate))

		images := []*gallery.Image{testhelpers.ValidImage(t)}

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), pagination).
			Return(images, int64(1), nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: album.OwnerID().String(), // Owner
			Page:             1,
			PerPage:          20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 1)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})

	t.Run("successful list - empty album", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), pagination).
			Return([]*gallery.Image{}, int64(0), nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID: album.ID().String(),
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Images)
		assert.Equal(t, int64(0), result.TotalCount)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})

	t.Run("pagination - multiple pages", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		images := make([]*gallery.Image, 10)
		for i := 0; i < 10; i++ {
			images[i] = testhelpers.ValidImage(t)
		}

		pagination, _ := shared.NewPagination(1, 10)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), pagination).
			Return(images, int64(35), nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID: album.ID().String(),
			Page:    1,
			PerPage: 10,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Images, 10)
		assert.Equal(t, int64(35), result.TotalCount)
		assert.Equal(t, 4, result.TotalPages) // 35 / 10 = 4 pages
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})

	t.Run("unauthorized - private album, not owner", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityPrivate))

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "550e8400-e29b-41d4-a716-446655440001", // Different user
			Page:             1,
			PerPage:          20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertNotCalled(t, "FindImagesInAlbum")
	})

	t.Run("unauthorized - private album, anonymous user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityPrivate))

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "", // Anonymous
			Page:             1,
			PerPage:          20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("invalid album id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		query := queries.ListAlbumImagesQuery{
			AlbumID: "invalid-uuid",
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid album id")
		assert.Nil(t, result)
	})

	t.Run("invalid requesting user id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "invalid-uuid",
			Page:             1,
			PerPage:          20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid requesting user id")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("album not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		albumID := testhelpers.ValidAlbumIDParsed()
		mockAlbumRepo.On("FindByID", mock.Anything, albumID).
			Return(nil, gallery.ErrAlbumNotFound).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID: testhelpers.ValidAlbumID,
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find album")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("repository error - find images", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		pagination, _ := shared.NewPagination(1, 20)

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), pagination).
			Return(nil, int64(0), fmt.Errorf("database error")).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID: album.ID().String(),
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find images in album")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})

	t.Run("default pagination applied", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		mockAlbumImageRepo := new(testhelpers.MockAlbumImageRepository)
		handler := queries.NewListAlbumImagesHandler(mockAlbumRepo, mockAlbumImageRepo)

		album := testhelpers.ValidAlbum(t)
		images := []*gallery.Image{testhelpers.ValidImage(t)}

		defaultPagination := shared.DefaultPagination()
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()
		mockAlbumImageRepo.On("FindImagesInAlbum", mock.Anything, album.ID(), defaultPagination).
			Return(images, int64(1), nil).Once()

		query := queries.ListAlbumImagesQuery{
			AlbumID: album.ID().String(),
			Page:    0, // Invalid, will use default
			PerPage: 0, // Invalid, will use default
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, defaultPagination.Page(), result.Page)
		assert.Equal(t, defaultPagination.PerPage(), result.PerPage)
		mockAlbumRepo.AssertExpectations(t)
		mockAlbumImageRepo.AssertExpectations(t)
	})
}
