package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

func TestImageHandler_Routes_UploadRateLimiter(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	// Create a mock repo for ListImagesHandler to prevent panic
	mockRepo := new(MockImageRepository)
	// Allow FindPublic to be called (it will return empty list)
	mockRepo.On("FindPublic", mock.Anything, mock.Anything).Return([]*gallery.Image{}, int64(0), nil).Maybe()

	listImagesHandler := queries.NewListImagesHandler(mockRepo, &logger)

	// We construct ImageHandler with listImagesHandler
	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, listImagesHandler, nil, nil, logger,
	)

	// Create a mock middleware that sets a header
	mockMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "executed")
			next.ServeHTTP(w, r)
		})
	}

	// Act
	router := imageHandler.Routes(mockMiddleware)

	// 1. Verify middleware is applied to POST / (Upload)
	// Note: Upload handler will fail (401 or similar) because we didn't set up context/deps
	// But we only care about middleware execution.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Assert middleware execution
	assert.Equal(t, "executed", rec.Header().Get("X-Test-Middleware"))

	// 2. Verify middleware is NOT applied to GET / (List)
	reqGet := httptest.NewRequest(http.MethodGet, "/", nil)
	recGet := httptest.NewRecorder()

	router.ServeHTTP(recGet, reqGet)
	assert.Empty(t, recGet.Header().Get("X-Test-Middleware"))
}
