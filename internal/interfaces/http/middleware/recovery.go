package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// Recovery is a middleware that recovers from panics in HTTP handlers.
//
// When a panic occurs:
// 1. Captures the panic value and stack trace
// 2. Logs the error with full stack trace for debugging
// 3. Returns a 500 Internal Server Error with RFC 7807 Problem Details
// 4. DOES NOT expose panic details to the client (security consideration)
//
// This middleware should be placed early in the middleware chain (after RequestID and Logger)
// to ensure it can catch panics from all downstream middleware and handlers.
//
// Security Properties:
// - Stack traces are logged server-side only (not sent to client)
// - Generic error message prevents information disclosure
// - Request ID included for correlation with logs
//
// Usage:
//
//	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
//	r := chi.NewRouter()
//	r.Use(middleware.RequestID)
//	r.Use(middleware.Logger(logger))
//	r.Use(middleware.Recovery(logger))
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			defer func() {
				//nolint:nestif // Standard panic recovery pattern requires nested if for proper error handling.
				if rvr := recover(); rvr != nil {
					// Capture stack trace
					stackTrace := debug.Stack()

					// Get request ID for correlation
					requestID := GetRequestID(ctx)

					// Get user ID if authenticated (for audit purposes)
					userID, _ := GetUserIDString(ctx)

					// Build error message from panic value
					var errMsg string
					if err, ok := rvr.(error); ok {
						errMsg = err.Error()
					} else {
						errMsg = fmt.Sprintf("%v", rvr)
					}

					// Log the panic with full context
					logEvent := logger.Error().
						Str("request_id", requestID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_addr", getClientIP(r)).
						Str("panic", errMsg).
						Bytes("stack_trace", stackTrace)

					if userID != "" {
						logEvent = logEvent.Str("user_id", userID)
					}

					logEvent.Msg("panic recovered in http handler")

					// Return generic error to client (DO NOT leak stack trace)
					// Check if response has already been written
					if !isResponseWritten(w) {
						WriteError(w, r,
							http.StatusInternalServerError,
							"Internal Server Error",
							"An unexpected error occurred. Please try again later.",
						)
					} else {
						// Response already partially written, can't send problem details
						// Log that we couldn't send proper error response
						logger.Warn().
							Str("request_id", requestID).
							Msg("response already written when panic occurred, cannot send error response")
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// isResponseWritten checks if the response headers have been written.
// This is a best-effort check using type assertion on http.ResponseWriter.
func isResponseWritten(w http.ResponseWriter) bool {
	// If we wrapped the response writer, check our wrapper
	if rw, ok := w.(*responseWriter); ok {
		return rw.wroteHeader
	}

	// For unwrapped writers, we can't reliably detect if headers were sent
	// Default to false (safer to attempt error response)
	return false
}
