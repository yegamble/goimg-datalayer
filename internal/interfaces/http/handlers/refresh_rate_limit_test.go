package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestRefreshRateLimitIntegration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := zerolog.Nop()
	rlConfig := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      logger,
		GlobalLimit: 100,
		AuthLimit:   300,
		LoginLimit:  5, // Strict limit for login and refresh
		WindowSize:  time.Minute,
	}

	mockRepo := new(testhelpers.MockUserRepository)
	mockJWT := new(testhelpers.MockJWTService)
	mockRefresh := new(testhelpers.MockRefreshTokenService)
	mockSession := new(testhelpers.MockSessionStore)

	// Create RefreshHandler
	refreshHandler := commands.NewRefreshTokenHandler(
		mockRepo,
		mockJWT,
		mockRefresh,
		mockSession,
		&logger,
	)

	// Create AuthHandler with RefreshHandler
	authHandler := handlers.NewAuthHandler(nil, nil, refreshHandler, nil, nil, nil, nil, nil, nil, logger)

	mwConfig := handlers.MiddlewareConfig{
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	// Create Router
	router := handlers.NewRouter(
		authHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		getTestMetricsCollector(),
		mwConfig,
		false,
	)

	reqBody := handlers.RefreshRequest{
		RefreshToken: "some-refresh-token",
	}
	body, _ := json.Marshal(reqBody)

	// Mock ValidateToken to return error (so handler doesn't crash but returns 401/500)
	// We only care that it passed the rate limiter
	mockRefresh.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, errors.New("mock error")).Maybe()

	// Make 5 requests - all should succeed (or at least pass rate limiter)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345" // Same IP

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should NOT be 429
		assert.NotEqual(t, http.StatusTooManyRequests, rr.Code, "Request %d should be allowed (got %d)", i+1, rr.Code)
	}

	// 6th request should be blocked
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "6th request should be rate limited")

	if rr.Code == http.StatusTooManyRequests {
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	}
}
