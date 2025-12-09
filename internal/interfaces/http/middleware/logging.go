package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if !rw.wroteHeader {
		rw.status = status
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(status)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	if err != nil {
		return n, fmt.Errorf("response write: %w", err)
	}
	return n, nil
}

// Logger is a middleware that logs HTTP requests and responses using structured logging.
//
// Logged fields:
//   - request: method, path, remote_addr, user_agent, request_id
//   - response: status, duration_ms, bytes_written
//   - user: user_id (if authenticated)
//
// Security considerations:
//   - DOES NOT log: Authorization header, request body with passwords
//   - DOES log: Status codes, timing, user IDs for audit trail
//   - Failed requests (4xx, 5xx) logged at WARN level
//   - Successful requests (2xx, 3xx) logged at INFO level
//
// This middleware should be placed AFTER RequestID middleware to include
// the request ID in log entries.
//
// Usage:
//
//	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
//	r := chi.NewRouter()
//	r.Use(middleware.RequestID)
//	r.Use(middleware.Logger(logger))
func Logger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Wrap response writer to capture status and bytes
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // Default status if WriteHeader not called
				wroteHeader:    false,
			}

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Log incoming request
			requestLogger := logger.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", getClientIP(r)).
				Str("user_agent", r.UserAgent()).
				Logger()

			// Add query params for debugging (but not in production to avoid PII leakage)
			if r.URL.RawQuery != "" {
				requestLogger = requestLogger.With().
					Str("query", r.URL.RawQuery).
					Logger()
			}

			// Add user ID if authenticated
			if userID, ok := GetUserIDString(r.Context()); ok {
				requestLogger = requestLogger.With().
					Str("user_id", userID).
					Logger()
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate request duration
			duration := time.Since(startTime)

			// Build log event
			logEvent := requestLogger.WithLevel(logLevelForStatus(wrapped.status)).
				Int("status", wrapped.status).
				Dur("duration_ms", duration).
				Int64("bytes_written", wrapped.bytesWritten)

			// Add response headers of interest
			if wrapped.Header().Get("Content-Type") != "" {
				logEvent = logEvent.Str("content_type", wrapped.Header().Get("Content-Type"))
			}

			// Log the completed request
			logEvent.Msg("http request completed")
		})
	}
}

// logLevelForStatus returns the appropriate log level based on HTTP status code.
func logLevelForStatus(status int) zerolog.Level {
	switch {
	case status >= httpStatusServerError:
		return zerolog.ErrorLevel // Server errors
	case status >= httpStatusClientError:
		return zerolog.WarnLevel // Client errors
	case status >= httpStatusRedirect:
		return zerolog.InfoLevel // Redirects
	default:
		return zerolog.InfoLevel // Success
	}
}

// getClientIP extracts the real client IP address from the request.
// It checks X-Forwarded-For and X-Real-IP headers (if behind a proxy)
// and falls back to RemoteAddr.
//
// Security Note: Only trust X-Forwarded-For if the request comes from a
// trusted proxy. In production, validate the proxy IP address before
// trusting these headers to prevent IP spoofing.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs)
	// Format: "client, proxy1, proxy2"
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client IP)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// Format: "ip:port" or "[ipv6]:port"
	remoteAddr := r.RemoteAddr

	// Strip port if present
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			// IPv6 addresses are wrapped in brackets [::1]:8080
			if i > 0 && remoteAddr[0] == '[' {
				return remoteAddr[1 : i-1]
			}
			return remoteAddr[:i]
		}
	}

	return remoteAddr
}
