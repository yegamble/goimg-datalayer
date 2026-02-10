package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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

var (
	testMetricsCollector *middleware.MetricsCollector
	metricsOnce          sync.Once
)

func getTestMetricsCollector() *middleware.MetricsCollector {
	metricsOnce.Do(func() {
		testMetricsCollector = middleware.NewMetricsCollector()
	})
	return testMetricsCollector
}

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
		LoginLimit:  5, // Strict limit for login
		WindowSize:  time.Minute,
	}

	// 3. Setup Mocks
	mockRepo := new(testhelpers.MockUserRepository)
	mockJWT := new(testhelpers.MockJWTService)
	mockRefresh := new(testhelpers.MockRefreshTokenService)
	mockSession := new(testhelpers.MockSessionStore)
	mockMetrics := new(testhelpers.MockAuthMetricsRecorder)

	// 4. Create Handler
	// We mock the login handler to always fail with Unauthorized.
	// This simulates failed attempts which should be rate limited.
	loginHandler := commands.NewLoginHandler(
		mockRepo,
		mockJWT,
		mockRefresh,
		mockSession,
		mockMetrics,
		&logger,
	)

	authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, logger)

	// Other minimal handlers required by router
	healthHandler := handlers.NewHealthHandler(nil, nil, nil, nil, logger)

	// 5. Create Router
	// Use singleton metrics collector to avoid panic if run alongside other tests
	metricsCollector := getTestMetricsCollector()

	mwConfig := handlers.MiddlewareConfig{
		JWTService:        nil, // Not needed for public auth routes
		TokenBlacklist:    nil, // Not needed for public auth routes
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	router := handlers.NewRouter(
		authHandler, // Inject auth handler
		nil, nil, nil, nil, nil,
		healthHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		metricsCollector,
		mwConfig,
		false, // isProd
	)

	// Setup expectations for the mock handler
	// The handler should be called for allowed requests
	mockMetrics.On("RecordLoginDelay", mock.Anything).Return()
	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, identity.ErrUserNotFound)

	// 6. Test
	// 5 allowed requests
	for i := 0; i < 5; i++ {
		reqBody := handlers.LoginRequest{
			Email:    "test@example.com",
			Password: "WrongPassword",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Using default RemoteAddr (usually 127.0.0.1 or similar) which works for rate limiting

		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Expect 401 Unauthorized (handler called)
		// If rate limiting was incorrectly applied earlier, this might be 429
		assert.Equal(t, http.StatusUnauthorized, w.Code, "request %d should be allowed but fail auth", i+1)

		// If rate limiting headers are present, we can check them too
		// But let's focus on the 429 blocking behavior
	}

	// 7. 6th request - blocked (Should be 429)
	reqBody := handlers.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// This assertion should FAIL initially (will be 401 because rate limiter is missing)
	// After fix, it should be 429
	if w.Code != http.StatusTooManyRequests {
		t.Logf("Expected 429 Too Many Requests, got %d", w.Code)
		t.Fail() // Mark test as failed
	}
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "6th request should be blocked")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}
