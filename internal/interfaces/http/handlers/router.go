package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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
//  2. Logger - structured request/response logging
//  3. Recovery - panic recovery
//  4. SecurityHeaders - defense headers (CSP, X-Frame-Options, etc.)
//  5. CORS - cross-origin resource sharing
//
// Route groups:
//   - Public routes: /api/v1/auth/* (no authentication)
//   - Protected routes: /api/v1/users/* (JWT authentication required)
func NewRouter(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	middlewareConfig MiddlewareConfig,
	isProd bool,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware (applies to all routes)
	r.Use(middleware.RequestID)
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

	// Health check endpoint (no authentication required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"goimg-api"}`))
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no authentication required)
		r.Mount("/auth", authHandler.Routes())

		// Protected user routes (JWT authentication required)
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
		})
	})

	return r
}
