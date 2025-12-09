package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

const (
	// Default CORS preflight cache duration in seconds (1 hour).
	defaultCORSMaxAge = 3600
)

// CORSConfig holds configuration for Cross-Origin Resource Sharing (CORS).
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to make cross-origin requests.
	// Use ["*"] to allow all origins (NOT recommended for production).
	// Use specific origins like ["https://app.example.com", "https://admin.example.com"]
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods allowed for CORS requests.
	// Common methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
	AllowedMethods []string

	// AllowedHeaders is a list of headers allowed in CORS requests.
	// Must include headers sent by client (Authorization, Content-Type, etc.)
	AllowedHeaders []string

	// ExposedHeaders is a list of headers exposed to the client.
	// This allows JavaScript to access custom response headers.
	// Example: X-Request-ID, X-RateLimit-*, Content-Range
	ExposedHeaders []string

	// AllowCredentials indicates whether cookies and authorization headers
	// are allowed in CORS requests.
	// SECURITY: Cannot be true when AllowedOrigins contains "*"
	AllowCredentials bool

	// MaxAge is the time (in seconds) that browsers can cache CORS preflight responses.
	// Reduces OPTIONS requests from browsers.
	// Recommended: 3600 (1 hour) to 86400 (24 hours)
	MaxAge int
}

// DefaultCORSConfig returns a secure CORS configuration for production.
// Allows specific origins only and includes common security headers.
//
// You MUST customize AllowedOrigins for your deployment:
//
//	cfg := middleware.DefaultCORSConfig()
//	cfg.AllowedOrigins = []string{
//	    "https://app.goimg.com",
//	    "https://admin.goimg.com",
//	}
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		// IMPORTANT: Replace with your actual frontend origins
		// NEVER use "*" in production with AllowCredentials=true
		AllowedOrigins: []string{
			"http://localhost:3000", // Development frontend
			"http://localhost:5173", // Vite dev server
			"https://app.goimg.dev", // Production frontend
		},

		// Allow common HTTP methods
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions, // Required for preflight requests
		},

		// Allow common headers
		AllowedHeaders: []string{
			"Accept",
			"Authorization", // Required for Bearer tokens
			"Content-Type",  // Required for JSON requests
			"X-Request-ID",  // Allow client-provided request IDs
			"X-CSRF-Token",  // CSRF protection token
		},

		// Expose custom response headers to JavaScript
		ExposedHeaders: []string{
			"X-Request-ID",      // Request correlation ID
			"X-RateLimit-Limit", // Rate limit info
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Content-Range", // Pagination info
			"X-Total-Count", // Total items count
		},

		// Allow credentials (cookies, authorization headers)
		AllowCredentials: true,

		// Cache preflight responses for 1 hour
		MaxAge: defaultCORSMaxAge,
	}
}

// DevelopmentCORSConfig returns a permissive CORS configuration for local development.
// WARNING: DO NOT use in production!
//
// This configuration:
// - Allows all origins (*)
// - Allows all methods
// - Allows all headers
// - Does NOT allow credentials (security requirement with wildcard origins).
func DevelopmentCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"}, // Allow all origins in development

		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},

		AllowedHeaders: []string{"*"}, // Allow all headers

		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Content-Range",
			"X-Total-Count",
		},

		// CANNOT be true when AllowedOrigins = "*" (browser security policy)
		AllowCredentials: false,

		MaxAge: defaultCORSMaxAge,
	}
}

// CORS creates a CORS middleware with the given configuration.
//
// This middleware handles:
// 1. Preflight OPTIONS requests (CORS negotiation)
// 2. Setting CORS response headers on actual requests
// 3. Validating origin against allowed list
//
// Security considerations:
// - Never use AllowedOrigins=["*"] with AllowCredentials=true (browsers reject this)
// - Validate origins against a strict allowlist in production
// - Be careful with AllowedHeaders - only allow what's necessary
// - ExposedHeaders only exposes specific headers to JavaScript
//
// This middleware should be placed AFTER security headers middleware
// but BEFORE authentication middleware (to allow preflight requests).
//
// Usage (production):
//
//	cfg := middleware.DefaultCORSConfig()
//	cfg.AllowedOrigins = []string{"https://app.example.com"}
//	r.Use(middleware.CORS(cfg))
//
// Usage (development):
//
//	if isDevelopment {
//	    cfg := middleware.DevelopmentCORSConfig()
//	    r.Use(middleware.CORS(cfg))
//	} else {
//	    cfg := middleware.DefaultCORSConfig()
//	    cfg.AllowedOrigins = os.Getenv("ALLOWED_ORIGINS") // From config
//	    r.Use(middleware.CORS(cfg))
//	}
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	// Validate configuration
	if cfg.AllowCredentials && containsWildcard(cfg.AllowedOrigins) {
		panic("CORS configuration error: AllowCredentials cannot be true when AllowedOrigins contains '*'")
	}

	// Create go-chi/cors handler with our configuration
	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		ExposedHeaders:   cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,

		// Enable debug logging in development
		// Set to false in production to reduce log noise
		Debug: false,

		// OptionsPassthrough determines whether to pass OPTIONS requests to handlers
		// or to handle them entirely in the CORS middleware.
		// false = CORS middleware handles OPTIONS (recommended for most cases)
		OptionsPassthrough: false,
	})

	return corsHandler
}

// containsWildcard checks if the origins list contains a wildcard "*".
func containsWildcard(origins []string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

// IsOriginAllowed checks if an origin is allowed by the CORS configuration.
// Useful for manual origin validation in custom middleware.
func IsOriginAllowed(origin string, allowedOrigins []string) bool {
	// Check for wildcard
	if containsWildcard(allowedOrigins) {
		return true
	}

	// Normalize origin (lowercase, trim)
	normalizedOrigin := strings.ToLower(strings.TrimSpace(origin))

	// Check exact matches
	for _, allowed := range allowedOrigins {
		if strings.ToLower(strings.TrimSpace(allowed)) == normalizedOrigin {
			return true
		}
	}

	return false
}
