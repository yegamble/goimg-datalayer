package handlers_test

import (
	"bytes"
	"encoding/json"
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
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestLoginRateLimitIntegration(t *testing.T) {
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
		LoginLimit:  5,
		WindowSize:  time.Minute,
	}

	mockRepo := new(testhelpers.MockUserRepository)
	mockJWT := new(testhelpers.MockJWTService)
	mockRefresh := new(testhelpers.MockRefreshTokenService)
	mockSession := new(testhelpers.MockSessionStore)
	mockMetrics := new(testhelpers.MockAuthMetricsRecorder)

	loginHandler := commands.NewLoginHandler(
		mockRepo,
		mockJWT,
		mockRefresh,
		mockSession,
		mockMetrics,
		&logger,
	)

	authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, nil, nil, nil, nil, logger)

	mwConfig := handlers.MiddlewareConfig{
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	router := handlers.NewRouter(
		authHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		getTestMetricsCollector(),
		mwConfig,
		false,
	)

	reqBody := handlers.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(reqBody)

	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)
	mockMetrics.On("RecordLoginDelay", mock.Anything).Return()

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345"

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code, "Request %d should be allowed", i+1)

		if rr.Header().Get("X-RateLimit-Limit") != "" {
			assert.Equal(t, "5", rr.Header().Get("X-RateLimit-Limit"))
			assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "6th request should be rate limited")

	if rr.Code == http.StatusTooManyRequests {
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
		assert.Contains(t, rr.Body.String(), "exceeded the login attempt limit")
	}
}
