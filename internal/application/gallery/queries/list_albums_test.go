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

func TestListAlbumsHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("successful list - public albums", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		albums := []*gallery.Album{
			testhelpers.ValidAlbum(t),
			testhelpers.ValidAlbum(t),
		}

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return(albums, int64(2), nil).Once()

		query := queries.ListAlbumsQuery{
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 2)
		assert.Equal(t, int64(2), result.TotalCount)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		assert.Equal(t, 1, result.TotalPages)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("successful list - by owner", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		ownerID := testhelpers.ValidUserIDParsed()
		albums := []*gallery.Album{
			testhelpers.ValidAlbum(t),
		}

		mockAlbumRepo.On("FindByOwner", mock.Anything, ownerID).
			Return(albums, nil).Once()

		query := queries.ListAlbumsQuery{
			OwnerUserID: testhelpers.ValidUserID,
			Page:        1,
			PerPage:     20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 1)
		assert.Equal(t, int64(1), result.TotalCount)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("successful list - empty result", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return([]*gallery.Album{}, int64(0), nil).Once()

		query := queries.ListAlbumsQuery{
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 0)
		assert.Equal(t, int64(0), result.TotalCount)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("filter by visibility - public", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		// Create albums with different visibilities
		publicAlbum := testhelpers.ValidAlbum(t)
		privateAlbum := testhelpers.ValidAlbum(t)
		require.NoError(t, privateAlbum.UpdateVisibility(gallery.VisibilityPrivate))

		albums := []*gallery.Album{publicAlbum, privateAlbum}

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return(albums, int64(2), nil).Once()

		query := queries.ListAlbumsQuery{
			Visibility: "public",
			Page:       1,
			PerPage:    20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 1)
		assert.Equal(t, int64(1), result.TotalCount) // Only public albums
		assert.Equal(t, "public", result.Albums[0].Visibility)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("filter by visibility - private", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		ownerID := testhelpers.ValidUserIDParsed()

		// Create albums with different visibilities
		publicAlbum := testhelpers.ValidAlbum(t)
		privateAlbum := testhelpers.ValidAlbum(t)
		require.NoError(t, privateAlbum.UpdateVisibility(gallery.VisibilityPrivate))

		albums := []*gallery.Album{publicAlbum, privateAlbum}

		mockAlbumRepo.On("FindByOwner", mock.Anything, ownerID).
			Return(albums, nil).Once()

		query := queries.ListAlbumsQuery{
			OwnerUserID: testhelpers.ValidUserID,
			Visibility:  "private",
			Page:        1,
			PerPage:     20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 1)
		assert.Equal(t, int64(1), result.TotalCount) // Only private albums
		assert.Equal(t, "private", result.Albums[0].Visibility)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("pagination - multiple pages", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		albums := make([]*gallery.Album, 10)
		for i := 0; i < 10; i++ {
			albums[i] = testhelpers.ValidAlbum(t)
		}

		pagination, _ := shared.NewPagination(1, 10)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return(albums, int64(25), nil).Once()

		query := queries.ListAlbumsQuery{
			Page:    1,
			PerPage: 10,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 10)
		assert.Equal(t, int64(25), result.TotalCount)
		assert.Equal(t, 3, result.TotalPages) // 25 items / 10 per page = 3 pages
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("pagination - by owner with manual pagination", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		ownerID := testhelpers.ValidUserIDParsed()

		// Create 25 albums
		albums := make([]*gallery.Album, 25)
		for i := 0; i < 25; i++ {
			albums[i] = testhelpers.ValidAlbum(t)
		}

		mockAlbumRepo.On("FindByOwner", mock.Anything, ownerID).
			Return(albums, nil).Once()

		query := queries.ListAlbumsQuery{
			OwnerUserID: testhelpers.ValidUserID,
			Page:        2,
			PerPage:     10,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Albums, 10)              // Second page: items 10-19
		assert.Equal(t, int64(25), result.TotalCount) // Total count across all pages
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("invalid owner user id", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		query := queries.ListAlbumsQuery{
			OwnerUserID: "invalid-uuid",
			Page:        1,
			PerPage:     20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid owner user id")
		assert.Nil(t, result)
	})

	t.Run("invalid visibility", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return([]*gallery.Album{testhelpers.ValidAlbum(t)}, int64(1), nil).Once()

		query := queries.ListAlbumsQuery{
			Visibility: "invalid-visibility",
			Page:       1,
			PerPage:    20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid visibility")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("repository error - find public", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		pagination, _ := shared.NewPagination(1, 20)
		mockAlbumRepo.On("FindPublic", mock.Anything, pagination).
			Return(nil, int64(0), fmt.Errorf("database error")).Once()

		query := queries.ListAlbumsQuery{
			Page:    1,
			PerPage: 20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find public albums")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("repository error - find by owner", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		ownerID := testhelpers.ValidUserIDParsed()
		mockAlbumRepo.On("FindByOwner", mock.Anything, ownerID).
			Return(nil, fmt.Errorf("database error")).Once()

		query := queries.ListAlbumsQuery{
			OwnerUserID: testhelpers.ValidUserID,
			Page:        1,
			PerPage:     20,
		}

		// Act
		result, err := handler.Handle(context.Background(), query)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find albums by owner")
		assert.Nil(t, result)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("default pagination applied", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockAlbumRepo := new(testhelpers.MockAlbumRepository)
		handler := queries.NewListAlbumsHandler(mockAlbumRepo)

		albums := []*gallery.Album{testhelpers.ValidAlbum(t)}

		// Default pagination is used when page/perPage are invalid
		defaultPagination := shared.DefaultPagination()
		mockAlbumRepo.On("FindPublic", mock.Anything, defaultPagination).
			Return(albums, int64(1), nil).Once()

		query := queries.ListAlbumsQuery{
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
	})
}
