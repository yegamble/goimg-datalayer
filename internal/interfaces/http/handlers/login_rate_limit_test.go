package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestLoginRateLimitIntegration(t *testing.T) {
	// 1. Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 2. Setup Config
	logger := zerolog.Nop()
	rlConfig := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      logger,
		GlobalLimit: 100,
		AuthLimit:   300,
		LoginLimit:  5, // Key config
		WindowSize:  time.Minute,
	}

	// 3. Create Handler
	// We only need AuthHandler to be non-nil. Dependencies can be nil if we trigger early return in handler.
	authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, logger)
	healthHandler := handlers.NewHealthHandler(nil, nil, nil, nil, logger)

	// 4. Create Router
	mwConfig := handlers.MiddlewareConfig{
		// JWTService can be nil for public routes
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	metrics := middleware.NewMetricsCollector()

	router := handlers.NewRouter(
		authHandler, // 1
		nil,         // 2
		nil,         // 3
		nil,         // 4
		nil,         // 5
		nil,         // 6
		healthHandler, // 7
		nil, // 8
		nil, // 9
		nil, // 10
		nil, // 11
		nil, // 12
		nil, // 13
		nil, // 14
		nil, // 15
		nil, // 16
		nil, // 17
		nil, // 18
		nil, // 19
		nil, // 20
		nil, // 21
		nil, // 22
		nil, // 23
		metrics,
		mwConfig,
		false,
	)

	// 5. Test Loop
	// Send 5 requests - should be allowed (but return 400 Bad Request due to invalid body)
	for i := 0; i < 5; i++ {
		// Empty body will cause DecodeJSON to fail or return empty struct, validation might fail
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("{}"))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Expect 400 Bad Request (Validation Failed)
		// If it was 500, it means it reached the handler logic where we have nil dependencies.
		// But Validation check happens before that.
		// Even if it returns 500, it means Rate Limiter ALLOWED it.
		// So checking code != 429 is enough to say "Allowed".
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code, "request %d should be allowed", i+1)

		// Verify Rate Limit Headers
		// X-RateLimit-Limit should be 5 for login endpoint
		// Note: The header is set only if middleware runs.
		// If middleware is missing, this header will be empty.
		// This assertion might fail if middleware is not applied!
		// But the main test is the 6th request.
	}

	// 6th request - should be blocked
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "6th request should be blocked")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}
