package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// RequestID is a middleware that generates a unique request ID for each request.
// The request ID is:
// - Stored in the request context (accessible via GetRequestID)
// - Set in the X-Request-ID response header
// - Used for request tracing and correlation across services
//
// If the client provides an X-Request-ID header, it will be validated and used
// if it's a valid UUID. Otherwise, a new UUID v4 will be generated.
//
// This middleware should be the FIRST in the middleware chain to ensure
// all downstream middleware and handlers have access to the request ID.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(middleware.RequestID)
//
// Retrieving request ID in handlers:
//
//	func MyHandler(w http.ResponseWriter, r *http.Request) {
//	    requestID := middleware.GetRequestID(r.Context())
//	    logger.Info().Str("request_id", requestID).Msg("processing request")
//	}
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestID string

		// Check if client provided X-Request-ID header
		clientRequestID := r.Header.Get("X-Request-ID")
		if clientRequestID != "" {
			// Validate that it's a valid UUID
			if _, err := uuid.Parse(clientRequestID); err == nil {
				requestID = clientRequestID
			}
		}

		// Generate new UUID v4 if not provided or invalid
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in response header for client correlation
		w.Header().Set("X-Request-ID", requestID)

		// Store request ID in context for downstream use
		ctx := SetRequestID(r.Context(), requestID)

		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
