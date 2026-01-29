package handlers_test

import (
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
		LoginLimit:  5, // Strict limit for login
		WindowSize:  time.Minute,
	}

	// 3. Create Handlers with minimal/nil dependencies
	// We deliberately use nil for command handlers.
	authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil, logger)
	healthHandler := handlers.NewHealthHandler(nil, nil, nil, nil, logger)
	// Create other dummy handlers to prevent panics in NewRouter
	imageHandler := handlers.NewImageHandler(nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// NewUserHandler: Get, Update, Delete, GetSessions, Logger
	userHandler := handlers.NewUserHandler(nil, nil, nil, nil, logger)

	albumHandler := handlers.NewAlbumHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	socialHandler := handlers.NewSocialHandler(nil, nil, nil, nil, nil, nil, logger)
	exploreHandler := handlers.NewExploreHandler(nil, nil, logger)

	// NewTwoFAHandler: Setup, Verify, Disable, Regenerate, VerifyLogin, GetStatus, Logger
	twoFAHandler := handlers.NewTwoFAHandler(nil, nil, nil, nil, nil, nil, logger)

	activityHandler := handlers.NewActivityHandler(nil, logger)

	// NewNotificationHandler: Get, GetCount, MarkRead, Logger
	notificationHandler := handlers.NewNotificationHandler(nil, nil, nil, logger)

	// 4. Create Router
	metrics := middleware.NewMetricsCollector()
	mwConfig := handlers.MiddlewareConfig{
		JWTService:        nil, // Not needed for login endpoint test
		TokenBlacklist:    nil,
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	router := handlers.NewRouter(
		authHandler,
		userHandler,
		imageHandler,
		albumHandler,
		socialHandler,
		exploreHandler,
		healthHandler,
		twoFAHandler,
		nil, // oauthHandler (optional)
		nil, // followHandler (optional)
		activityHandler,
		notificationHandler,
		nil, // ipfsHandler
		nil, // moderationHandler
		nil, // guestHandler
		nil, // oembedHandler
		nil, // previewHandler
		nil, // variantConfigHandler
		nil, // tagHandler
		nil, // featuredHandler
		nil, // groupHandler
		nil, // groupAlbumHandler
		metrics,
		mwConfig,
		false, // isProd
	)

	// 5. Test Execution
	// We simulate requests from the same IP
	remoteIP := "192.0.2.1:12345"

	// First 5 requests should be allowed (but fail with 500 due to nil handler)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = remoteIP
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert it's NOT 429
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code, "Request %d should not be rate limited", i+1)
	}

	// 6th request should be BLOCKED with 429
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = remoteIP
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// With the BUG, this will fail (it will be 500 instead of 429)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "6th request should be rate limited")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}
