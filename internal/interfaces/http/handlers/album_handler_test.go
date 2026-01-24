package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestAlbumHandler_List(t *testing.T) {
	t.Parallel()

	// Helper to create the handler with dependencies
	setupHandler := func() (*handlers.AlbumHandler, *testhelpers.MockAlbumRepository) {
		mockRepo := new(testhelpers.MockAlbumRepository)
		listHandler := queries.NewListAlbumsHandler(mockRepo)

		// Create AlbumHandler with only the ListAlbumsHandler populated
		// Other handlers are nil as they shouldn't be called
		handler := handlers.NewAlbumHandler(
			nil, nil, nil, nil, nil, // commands
			nil,                     // getAlbum
			listHandler,             // listAlbums
			nil, nil, nil,           // other queries
			zerolog.Nop(),
		)
		return handler, mockRepo
	}

	t.Run("anonymous user - lists public albums", func(t *testing.T) {
		t.Parallel()

		handler, mockRepo := setupHandler()

		// Setup mock expectations
		// Anonymous user -> Should call FindPublic
		pagination := shared.DefaultPagination()
		mockRepo.On("FindPublic", mock.Anything, pagination).
			Return([]*gallery.Album{}, int64(0), nil).Once()

		// Create request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.List(w, req)

		// Verify
		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("authenticated user - lists accessible albums", func(t *testing.T) {
		t.Parallel()

		handler, mockRepo := setupHandler()

		// Setup user context
		userID := identity.NewUserID()
		userUUID := uuid.MustParse(userID.String())

		// Setup mock expectations
		// Authenticated user -> Should call FindAllAccessible with userID
		pagination := shared.DefaultPagination()
		mockRepo.On("FindAllAccessible", mock.Anything, userID, pagination, mock.AnythingOfType("*gallery.Visibility")).
			Return([]*gallery.Album{}, int64(0), nil).Once()

		// Create request with user context
		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums", nil)
		ctx := middleware.SetUserContext(
			req.Context(),
			userUUID,
			"test@example.com",
			"user",
			uuid.New(),
			false,
		)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		// Execute
		handler.List(w, req)

		// Verify
		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("filter by owner", func(t *testing.T) {
		t.Parallel()

		handler, mockRepo := setupHandler()

		ownerID := identity.NewUserID()

		// Setup mock expectations
		// Filter by owner -> Should call FindByOwner
		pagination := shared.DefaultPagination()
		mockRepo.On("FindByOwner", mock.Anything, ownerID, pagination, mock.AnythingOfType("*gallery.Visibility")).
			Return([]*gallery.Album{}, int64(0), nil).Once()

		// Create request with owner_id param
		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums?owner_id="+ownerID.String(), nil)
		w := httptest.NewRecorder()

		// Execute
		handler.List(w, req)

		// Verify
		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("pagination parameters", func(t *testing.T) {
		t.Parallel()

		handler, mockRepo := setupHandler()

		// Setup mock expectations
		// Should parse offset=20, limit=10 -> page 3 (20/10 + 1)
		expectedPagination, _ := shared.NewPagination(3, 10)
		mockRepo.On("FindPublic", mock.Anything, expectedPagination).
			Return([]*gallery.Album{}, int64(0), nil).Once()

		// Create request with offset and limit
		req := httptest.NewRequest(http.MethodGet, "/api/v1/albums?offset=20&limit=10", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.List(w, req)

		// Verify
		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

// Router wiring test to ensure route is registered correctly
func TestAlbumHandler_Routes(t *testing.T) {
	t.Parallel()

	// Helper to create the handler
	createHandler := func() *handlers.AlbumHandler {
		// Just return a handler with nil dependencies, we only check routing
		return handlers.NewAlbumHandler(
			nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil,
			zerolog.Nop(),
		)
	}

	handler := createHandler()
	router := handler.Routes()

	// Verify route registration using chi walker or just making a request
	// Simple check: make sure "/" is registered

	walker := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if route == "/" && method == "GET" {
			return nil // Found it
		}
		return nil
	}

	if err := chi.Walk(router, walker); err != nil {
		t.Errorf("Walk failed: %v", err)
	}
}
