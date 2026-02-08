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
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	jwtpkg "github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockJWTMiddleware implements middleware.JWTServiceInterface
type MockJWTMiddleware struct {
	mock.Mock
}

func (m *MockJWTMiddleware) ValidateToken(tokenString string) (*jwtpkg.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.Claims), args.Error(1)
}

func (m *MockJWTMiddleware) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

// Global metrics collector for tests to avoid duplicate registration panic
var (
	testCollector     *middleware.MetricsCollector
	testCollectorOnce sync.Once
)

func getTestMetricsCollector() *middleware.MetricsCollector {
	testCollectorOnce.Do(func() {
		testCollector = middleware.NewMetricsCollector()
	})
	return testCollector
}

// TestLoginRateLimitIntegration verifies that login endpoint is rate limited.
// Security Control: Brute force protection prevents password guessing.
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
	loginLimit := 5
	rlConfig := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      logger,
		GlobalLimit: 100,
		AuthLimit:   300,
		LoginLimit:  loginLimit,
		WindowSize:  time.Minute,
	}

	// 3. Setup Mocks
	mockRepo := new(testhelpers.MockUserRepository)
	mockJWT := new(testhelpers.MockJWTService)
	mockRefresh := new(testhelpers.MockRefreshTokenService)
	mockSession := new(testhelpers.MockSessionStore)
	mockMetrics := new(testhelpers.MockAuthMetricsRecorder)
	mockBlacklist := new(testhelpers.MockTokenBlacklist)

	mockJWTMiddleware := new(MockJWTMiddleware)

	// 4. Create Handler
	loginHandler := commands.NewLoginHandler(
		mockRepo,
		mockJWT,
		mockRefresh,
		mockSession,
		mockMetrics,
		&logger,
	)

	// We create a AuthHandler with just loginHandler, others can be nil as we test login
	authHandler := handlers.NewAuthHandler(nil, loginHandler, nil, nil, nil, logger)

	// Other minimal handlers required by router
	healthHandler := handlers.NewHealthHandler(nil, nil, nil, nil, logger)

	// 5. Create Router
	mwConfig := handlers.MiddlewareConfig{
		JWTService:        mockJWTMiddleware,
		TokenBlacklist:    mockBlacklist,
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	router := handlers.NewRouter(
		authHandler, // Inject our auth handler
		nil, nil, nil, nil, nil, healthHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		getTestMetricsCollector(), // Use singleton collector
		mwConfig,
		false, // isProd
	)

	// 6. Test Login Rate Limit

	// Setup user for successful login
	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("StrongPassword123!")
	user, _ := identity.NewUser(email, username, passwordHash)
	_ = user.Activate()

	// Mock behavior for allowed requests
	mockRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(user, nil).Maybe()
	mockMetrics.On("RecordLoginDelay", mock.Anything).Return().Maybe()
	mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access_token", nil).Maybe()

	refreshMeta := &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}
	mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("refresh_token", refreshMeta, nil).Maybe()

	mockSession.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockJWT.On("GetTokenExpiration", "access_token").Return(time.Now().Add(15*time.Minute), nil).Maybe()

	reqBody := handlers.LoginRequest{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Execute requests up to the limit
	for i := 0; i < loginLimit; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should be successful (200 OK)
		if w.Code != http.StatusOK {
			t.Logf("Request %d failed with status %d: %s", i+1, w.Code, w.Body.String())
		}
		require.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		require.NotEqual(t, http.StatusTooManyRequests, w.Code, "Request %d should not be rate limited", i+1)
	}

	// Execute one more request - should be blocked
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// This assertion should PASS now that we applied the fix
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request over limit should be blocked")

	if w.Code == http.StatusTooManyRequests {
		assert.NotEmpty(t, w.Header().Get("Retry-After"))

		// Verify error response format
		var errorResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "Too Many Login Attempts", errorResp["title"])
	}
}
