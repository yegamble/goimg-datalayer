package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestAlbumHandler_List(t *testing.T) {
	mockAlbumRepo := new(testhelpers.MockAlbumRepository)
	listAlbumsHandler := queries.NewListAlbumsHandler(mockAlbumRepo)
	logger := zerolog.Nop()

	handler := handlers.NewAlbumHandler(
		nil, nil, nil, nil, nil,
		nil,
		listAlbumsHandler,
		nil, nil, nil,
		logger,
	)

	t.Run("authenticated user should trigger FindAllAccessible", func(t *testing.T) {
		userID := uuid.New()
		userEmail := "test@example.com"
		userRole := "user"
		sessionID := uuid.New()

		ctx := context.Background()
		ctx = middleware.SetUserContext(ctx, userID, userEmail, userRole, sessionID, false, true)

		req, err := http.NewRequestWithContext(ctx, "GET", "/api/v1/albums", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()

		idUserID, err := identity.ParseUserID(userID.String())
		require.NoError(t, err)

		albums := []*gallery.Album{}
		mockAlbumRepo.On("FindAllAccessible", mock.Anything, idUserID, mock.Anything, mock.Anything).
			Return(albums, int64(0), nil).Once()

		handler.List(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAlbumRepo.AssertExpectations(t)
	})

	t.Run("anonymous user should trigger FindPublic", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/albums", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()

		albums := []*gallery.Album{}
		mockAlbumRepo.On("FindPublic", mock.Anything, mock.Anything).
			Return(albums, int64(0), nil).Once()

		handler.List(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAlbumRepo.AssertExpectations(t)
	})
}
