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
		LoginLimit:  5, // Limit to 5 attempts per minute
		WindowSize:  time.Minute,
	}

	// 3. Setup Mocks for LoginHandler
	mockRepo := new(testhelpers.MockUserRepository)
	mockJWT := new(testhelpers.MockJWTService)
	mockRefresh := new(testhelpers.MockRefreshTokenService)
	mockSession := new(testhelpers.MockSessionStore)
	mockMetrics := new(testhelpers.MockAuthMetricsRecorder)

	// 4. Create LoginHandler
	loginHandler := commands.NewLoginHandler(
		mockRepo,
		mockJWT,
		mockRefresh,
		mockSession,
		mockMetrics,
		&logger,
	)

	// 5. Create AuthHandler
	// We only need loginHandler for this test
	authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, logger)

	// 6. Create Router with Middleware
	mwConfig := handlers.MiddlewareConfig{
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
		// Other fields nil as we don't need them for this specific test
	}

	router := handlers.NewRouter(
		authHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, // other handlers
		getTestMetricsCollector(),
		mwConfig,
		false, // isProd
	)

	// 7. Setup request data
	reqBody := handlers.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(reqBody)

	// Configure mock to fail login (invalid credentials)
	// We don't care about success/failure of login logic, only that the request reaches the handler
	// and rate limiter counts it.
	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)
	mockMetrics.On("RecordLoginDelay", mock.Anything).Return()

	// 8. Execute 5 allowed requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Use same IP for all requests
		req.RemoteAddr = "192.168.1.100:12345"

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should reach handler and return 401 (Unauthorized) due to mock setup
		// If rate limited, it would return 429
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "Request %d should be allowed", i+1)

		// Verify Rate Limit Headers (if middleware is applied)
		if rr.Header().Get("X-RateLimit-Limit") != "" {
			assert.Equal(t, "5", rr.Header().Get("X-RateLimit-Limit"))
			// Just checking existence for now as implementation details might vary slightly
			assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
		}
	}

	// 9. Execute 6th request - Expect Rate Limit Exceeded
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// This assertion is expected to FAIL until we apply the fix
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "6th request should be rate limited")

	if rr.Code == http.StatusTooManyRequests {
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
		assert.Contains(t, rr.Body.String(), "exceeded the login attempt limit")
	}
}
