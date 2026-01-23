package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestReportRateLimiter(t *testing.T) {
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
		// Other fields use defaults or are not relevant for ReportRateLimiter specific logic
		// which hardcodes limit to 10 and window to 1h
	}

	limiter := middleware.ReportRateLimiter(cfg)

	// Create a dummy handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter(nextHandler)

	// Create a user context
	userID := uuid.New()
	sessionID := uuid.New()

	// Test case: 10 allowed requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/reports", nil)
		ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false)
		ctx = middleware.SetRequestID(ctx, "req-id-"+strconv.Itoa(i))
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)

		// Verify headers
		assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, strconv.Itoa(10-(i+1)), rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	}

	// Test case: 11th request should be blocked
	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false)
	ctx = middleware.SetRequestID(ctx, "req-id-blocked")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "11th request should be blocked")
	assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))

	// Fast forward time to reset limit (1 hour + 1 second)
	mr.FastForward(time.Hour + time.Second)

	// Test case: Request after window reset should be allowed
	req = httptest.NewRequest(http.MethodPost, "/reports", nil)
	ctx = middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false)
	ctx = middleware.SetRequestID(ctx, "req-id-after-reset")
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "request after window reset should be allowed")
	assert.Equal(t, "9", rec.Header().Get("X-RateLimit-Remaining"))
}

func TestReportRateLimiter_NoUserContext(t *testing.T) {
	// Setup miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      zerolog.Nop(),
	}

	limiter := middleware.ReportRateLimiter(cfg)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter(nextHandler)

	// Request without user context
	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	// Don't set user context
	ctx := middleware.SetRequestID(req.Context(), "req-id-no-user")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 500 Internal Server Error as documented in code
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
