package middleware

import "time"

// HTTP middleware configuration constants.
const (
	// Auth constants.
	authHeaderParts = 2 // Expected parts in Authorization header ("Bearer" + token)

	// CORS constants.
	corsMaxAge = 3600 // CORS preflight cache duration in seconds

	// Logging constants.
	httpStatusServerError = 500 // HTTP 5xx status codes (server errors)
	httpStatusClientError = 400 // HTTP 4xx status codes (client errors)
	httpStatusRedirect    = 300 // HTTP 3xx status codes (redirects)

	// Rate limit constants.
	defaultRPM      = 100          // Default requests per minute
	defaultRPMBurst = 300          // Default burst capacity
	defaultRPMMin   = 5            // Minimum requests per minute
	defaultDuration = time.Minute  // Default rate limit window

	// Security headers constants.
	hstsMaxAge = 31536000 // HSTS max-age in seconds (1 year)
)
