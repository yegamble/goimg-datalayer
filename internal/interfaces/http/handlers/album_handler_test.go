package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestAlbumHandler_List(t *testing.T) {
	t.Parallel()

	t.Run("logged_in_user_uses_FindAllAccessible", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockRepo := new(testhelpers.MockAlbumRepository)
		listAlbumsHandler := queries.NewListAlbumsHandler(mockRepo)

		// Create handler with minimal dependencies
		handler := NewAlbumHandler(
			nil, nil, nil, nil, nil, nil, // commands
			listAlbumsHandler, // listAlbums
			nil, nil, nil,     // other queries
			zerolog.Nop(),
		)

		userID := identity.NewUserID()
		userUUID := userID.UUID()

		// Mock FindAllAccessible expectations
		// Expect FindAllAccessible to be called with the requesting UserID
		mockRepo.On("FindAllAccessible",
			mock.Anything,
			userID,
			mock.MatchedBy(func(p shared.Pagination) bool { return true }),
			(*gallery.Visibility)(nil),
		).Return([]*gallery.Album{}, int64(0), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums", nil)
		rec := httptest.NewRecorder()

		// Inject user into context
		ctx := middleware.SetUserContext(
			req.Context(),
			userUUID,
			"test@example.com",
			"user",
			uuid.New(),
			false,
		)
		req = req.WithContext(ctx)

		// Act
		handler.List(rec, req)

		// Assert
		require.Equal(t, http.StatusOK, rec.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("anonymous_user_uses_FindPublic", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockRepo := new(testhelpers.MockAlbumRepository)
		listAlbumsHandler := queries.NewListAlbumsHandler(mockRepo)

		// Create handler with minimal dependencies
		handler := NewAlbumHandler(
			nil, nil, nil, nil, nil, nil, // commands
			listAlbumsHandler, // listAlbums
			nil, nil, nil,     // other queries
			zerolog.Nop(),
		)

		// Mock FindPublic expectations
		mockRepo.On("FindPublic",
			mock.Anything,
			mock.MatchedBy(func(p shared.Pagination) bool { return true }),
		).Return([]*gallery.Album{}, int64(0), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums", nil)
		rec := httptest.NewRecorder()

		// No user context injection

		// Act
		handler.List(rec, req)

		// Assert
		require.Equal(t, http.StatusOK, rec.Code)
		mockRepo.AssertExpectations(t)
	})
}
