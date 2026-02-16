package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestImageUploadRateLimit(t *testing.T) {
	// 1. Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 2. Setup Config
	logger := zerolog.Nop()
	// Use a very low limit for testing (2 uploads per hour)
	rlConfig := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      logger,
		WindowSize:  time.Hour,
	}

	// 3. Setup Mocks for ImageHandler
	// We need these to instantiate the handler, but they won't be called if rate limit hits first
	mockRepo := new(testhelpers.MockImageRepository)
	mockAppStorage := new(testhelpers.MockStorageProvider)
	mockJobEnqueuer := new(testhelpers.MockJobEnqueuer)
	mockEventPublisher := new(testhelpers.MockEventPublisher)

	// Create UploadImageHandler (Command)
	uploadImageHandler := commands.NewUploadImageHandler(
		mockRepo,
		mockAppStorage,
		mockJobEnqueuer,
		mockEventPublisher,
		&logger,
	)

	// Create ImageHandler
	// Note: We pass nil for other dependencies as they are not used for upload
	imageHandler := handlers.NewImageHandler(
		uploadImageHandler,
		nil, nil, nil, nil, nil, nil,
		nil, // StorageProvider for GET
		"",
		logger,
	)

	// 4. Create Router with ImageHandler Routes AND Rate Limit Config
	r := chi.NewRouter()
	r.Mount("/images", imageHandler.Routes(&rlConfig))

	// 5. Setup User Context
	userID := identity.NewUserID()
	email := "test@example.com"
	role := "user"
	sessionID := uuid.New()

	// Helper to send upload request
	sendUploadRequest := func() *httptest.ResponseRecorder {
		// We don't need a real body because rate limiting happens before parsing body
		req := httptest.NewRequest(http.MethodPost, "/images/", nil)

		// Set user context
		ctx := middleware.SetUserContext(req.Context(), userID.UUID(), email, role, sessionID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 6. Execute requests

	// The default upload limit is 50 per hour (hardcoded in UploadRateLimiter middleware for now)
	// See internal/interfaces/http/middleware/rate_limit.go: func UploadRateLimiter
	// It ignores the limit in config for UploadRateLimiter and uses hardcoded 50.
	// Wait, checking the code...

	/*
	func UploadRateLimiter(cfg RateLimiterConfig) func(http.Handler) http.Handler {
		// Default upload limit: 50 uploads per hour
		uploadLimit := 50
		uploadWindow := time.Hour
		...
	*/

	// It uses hardcoded limit 50. So we need to send 50 requests to hit the limit.
	// This is a bit many for a unit test, but miniredis is fast.

	limit := 50

	for i := 0; i < limit; i++ {
		rec := sendUploadRequest()
		// It might fail with 400 Bad Request (missing multipart), but NOT 429
		// We expect 400 because we send empty body
		// As long as it is NOT 429, we are good.
		assert.NotEqual(t, http.StatusTooManyRequests, rec.Code, "Request %d should be allowed", i+1)

		// Also verify rate limit headers are present
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
	}

	// 7. Execute 51st request - Expect Rate Limit Exceeded
	rec := sendUploadRequest()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "Request should be rate limited")

	if rec.Code == http.StatusTooManyRequests {
		assert.NotEmpty(t, rec.Header().Get("Retry-After"))
		assert.Contains(t, rec.Body.String(), "exceeded the upload limit")
	}
}
