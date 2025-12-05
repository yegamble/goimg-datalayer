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

func TestGetAlbumHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval - public album by anonymous user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "", // Anonymous
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, album.ID().String(), result.ID)
		assert.Equal(t, album.OwnerID().String(), result.OwnerID)
		assert.Equal(t, album.Title(), result.Title)
		assert.Equal(t, "public", result.Visibility)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("successful retrieval - owner viewing own private album", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		// Make it private
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityPrivate))

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: album.OwnerID().String(), // Owner
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, album.ID().String(), result.ID)
		assert.Equal(t, "private", result.Visibility)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("unauthorized - private album, not owner", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		// Make it private
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityPrivate))

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "550e8400-e29b-41d4-a716-446655440001", // Different user
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("unauthorized - unlisted album, anonymous user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		// Make it unlisted
		require.NoError(t, album.UpdateVisibility(gallery.VisibilityUnlisted))

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "", // Anonymous
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
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		query := queries.GetAlbumQuery{
			AlbumID:          "invalid-uuid",
			RequestingUserID: testhelpers.ValidUserID,
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
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		// The handler fetches the album before parsing the requesting user ID
		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: "invalid-uuid",
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
		albumID := testhelpers.ValidAlbumIDParsed()

		mockAlbumRepo.On("FindByID", mock.Anything, albumID).
			Return(nil, gallery.ErrAlbumNotFound).Once()

		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		query := queries.GetAlbumQuery{
			AlbumID:          testhelpers.ValidAlbumID,
			RequestingUserID: testhelpers.ValidUserID,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find album")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		albumID := testhelpers.ValidAlbumIDParsed()

		mockAlbumRepo.On("FindByID", mock.Anything, albumID).
			Return(nil, fmt.Errorf("database connection failed")).Once()

		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		query := queries.GetAlbumQuery{
			AlbumID:          testhelpers.ValidAlbumID,
			RequestingUserID: testhelpers.ValidUserID,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find album")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("album with cover image", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewGetAlbumHandler(mockAlbumRepo)

		album := testhelpers.ValidAlbum(t)
		coverImageID := testhelpers.ValidImageIDParsed()
		album.SetCoverImage(&coverImageID)

		mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

		query := queries.GetAlbumQuery{
			AlbumID:          album.ID().String(),
			RequestingUserID: album.OwnerID().String(),
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.CoverImageID)
		assert.Equal(t, coverImageID.String(), *result.CoverImageID)
		mockAlbumRepo.AssertExpectations(t)
	})
}

func TestAlbumToDTO(t *testing.T) {
	t.Parallel()

	album := testhelpers.ValidAlbum(t)

	// Add a cover image
	coverImageID := testhelpers.ValidImageIDParsed()
	album.SetCoverImage(&coverImageID)

	// Get the album using the handler to test DTO conversion
	mockAlbumRepo := new(testhelpers.MockAlbumRepository)
	mockAlbumRepo.On("FindByID", mock.Anything, album.ID()).Return(album, nil).Once()

	handler := queries.NewGetAlbumHandler(mockAlbumRepo)
	dto, err := handler.Handle(context.Background(), queries.GetAlbumQuery{
		AlbumID: album.ID().String(),
	})

	require.NoError(t, err)
	require.NotNil(t, dto)

	// Verify DTO fields
	assert.Equal(t, album.ID().String(), dto.ID)
	assert.Equal(t, album.OwnerID().String(), dto.OwnerID)
	assert.Equal(t, album.Title(), dto.Title)
	assert.Equal(t, album.Description(), dto.Description)
	assert.Equal(t, album.Visibility().String(), dto.Visibility)
	assert.NotNil(t, dto.CoverImageID)
	assert.Equal(t, coverImageID.String(), *dto.CoverImageID)
	assert.NotEmpty(t, dto.CreatedAt)
	assert.NotEmpty(t, dto.UpdatedAt)
}
