package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestImageHandler_Routes_RateLimiting(t *testing.T) {
	// Setup miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Setup config
	cfg := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      zerolog.Nop(),
	}

	// Create ImageHandler with nil dependencies
	// We only care about middleware, and the handler will return 400 before accessing nil dependencies
	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zerolog.Nop(),
	)

	// Get router with rate limiting
	r := imageHandler.Routes(&cfg)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// Set user context (required for UploadRateLimiter)
	userID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", uuid.New(), false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(rec, req)

	// Verify rate limit headers are present
	// The Upload handler returns 400 (Bad Request) because of missing multipart form,
	// but the middleware should have already run and set the headers.
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))

	// Check specific limit value (50 for upload)
	assert.Equal(t, "50", rec.Header().Get("X-RateLimit-Limit"))
}

func TestImageHandler_Routes_NoRateLimiting(t *testing.T) {
	imageHandler := NewImageHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, zerolog.Nop(),
	)

	// Get router WITHOUT rate limiting
	r := imageHandler.Routes(nil)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// Set user context
	userID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "test@example.com", "user", uuid.New(), false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(rec, req)

	// Verify rate limit headers are NOT present
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, rec.Header().Get("X-RateLimit-Limit"))
}
