package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MiddlewareConfig holds configuration for HTTP middleware.
type MiddlewareConfig struct {
	// JWTService for token validation
	JWTService middleware.JWTServiceInterface

	// TokenBlacklist for revoked token checking
	TokenBlacklist middleware.TokenBlacklistInterface

	// Logger for structured logging
	Logger zerolog.Logger
}

// NewRouter creates a new chi router with all routes and middleware configured.
// This is the main entry point for HTTP routing.
//
// Middleware order (CRITICAL for security):
//  1. RequestID - generates correlation ID
//  2. Metrics - Prometheus metrics collection
//  3. Logger - structured request/response logging
//  4. Recovery - panic recovery
//  5. SecurityHeaders - defense headers (CSP, X-Frame-Options, etc.)
//  6. CORS - cross-origin resource sharing
//
// Route groups:
//   - Health/Metrics routes: /health, /health/ready, /metrics (no authentication)
//   - Public routes: /api/v1/auth/* (no authentication)
//   - Protected routes: /api/v1/users/*, /api/v1/images/*, /api/v1/albums/* (JWT authentication required)
//   - Social routes: /api/v1/images/{id}/likes, /api/v1/images/{id}/comments (JWT authentication required)
func NewRouter(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	imageHandler *ImageHandler,
	albumHandler *AlbumHandler,
	socialHandler *SocialHandler,
	healthHandler *HealthHandler,
	metricsCollector *middleware.MetricsCollector,
	middlewareConfig MiddlewareConfig,
	isProd bool,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware (applies to all routes)
	r.Use(middleware.RequestID)
	r.Use(middleware.MetricsMiddleware(metricsCollector))
	r.Use(middleware.Logger(middlewareConfig.Logger))
	r.Use(middleware.Recovery(middlewareConfig.Logger))

	// Security headers with production config
	securityCfg := middleware.DefaultSecurityHeadersConfig(isProd)
	r.Use(middleware.SecurityHeaders(securityCfg))

	// CORS with appropriate config
	var corsCfg middleware.CORSConfig
	if isProd {
		corsCfg = middleware.DefaultCORSConfig()
	} else {
		corsCfg = middleware.DevelopmentCORSConfig()
	}
	r.Use(middleware.CORS(corsCfg))

	// Timeout middleware (prevent long-running requests)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Health check endpoints (no authentication required)
	// Liveness probe - checks if server is running
	r.Get("/health", healthHandler.Liveness)

	// Readiness probe - checks if all dependencies (DB, Redis) are healthy
	r.Get("/health/ready", healthHandler.Readiness)

	// Prometheus metrics endpoint (no authentication required)
	// In production, consider adding basic auth or IP restriction
	r.Handle("/metrics", promhttp.Handler())

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no authentication required)
		r.Mount("/auth", authHandler.Routes())

		// Protected routes (JWT authentication required)
		r.Group(func(r chi.Router) {
			// JWT authentication middleware
			authCfg := middleware.AuthConfig{
				JWTService:     middlewareConfig.JWTService,
				TokenBlacklist: middlewareConfig.TokenBlacklist,
				Logger:         middlewareConfig.Logger,
				Optional:       false, // Authentication required
			}
			r.Use(middleware.JWTAuth(authCfg))

			// Mount user routes
			r.Mount("/users", userHandler.Routes())

			// Mount image routes
			// Note: Upload endpoint should have special rate limiting applied at handler level
			r.Mount("/images", imageHandler.Routes())

			// Mount album routes
			r.Mount("/albums", albumHandler.Routes())

			// Social interaction routes (likes and comments)
			// These are mounted under images and users paths
			r.Route("/images/{imageID}", func(r chi.Router) {
				r.Post("/likes", socialHandler.LikeImage)
				r.Delete("/likes", socialHandler.UnlikeImage)
				r.Post("/comments", socialHandler.AddComment)
				r.Get("/comments", socialHandler.ListImageComments)
			})

			// Comment deletion endpoint (not under images path)
			r.Delete("/comments/{commentID}", socialHandler.DeleteComment)

			// User liked images endpoint
			r.Get("/users/{userID}/likes", socialHandler.GetUserLikedImages)
		})
	})

	return r
}
