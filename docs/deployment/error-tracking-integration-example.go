// Example Integration Code for Error Tracking
//
// This file demonstrates how to integrate Sentry/GlitchTip error tracking
// with the existing goimg-datalayer middleware and handlers.
//
// DO NOT IMPORT THIS FILE - It's documentation/pseudocode only.
// Copy relevant patterns to actual implementation files.

package example

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// ============================================================================
// 1. INITIALIZATION IN main.go
// ============================================================================

// InitializeErrorTracking sets up Sentry/GlitchTip error tracking
func InitializeErrorTracking(logger *zerolog.Logger) error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		logger.Warn().Msg("SENTRY_DSN not set, error tracking disabled")
		return nil
	}

	environment := os.Getenv("SENTRY_ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	release := os.Getenv("SENTRY_RELEASE")
	if release == "" {
		release = "goimg-api@dev"
	}

	err := sentry.Init(sentry.ClientOptions{
		// Connection
		Dsn: dsn,

		// Environment and release tracking
		Environment: environment,
		Release:     release,

		// Sampling configuration
		SampleRate:       1.0,  // 100% of errors
		TracesSampleRate: 0.1,  // 10% of transactions

		// Error filtering and scrubbing
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Skip non-production environments if desired
			if environment == "development" || environment == "test" {
				// Optionally return nil to skip sending in dev/test
				// return nil
			}

			// Apply PII scrubbing
			event = scrubPII(event)

			return event
		},

		// Performance monitoring
		EnableTracing: true,

		// Debugging (only in dev)
		Debug: environment == "development",

		// Server identification
		ServerName: getHostname(),

		// Transport configuration
		Transport: sentry.NewHTTPSyncTransport(),

		// Timeouts
		HTTPTransport: &sentry.HTTPTransport{
			Timeout: 5 * time.Second,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	logger.Info().
		Str("environment", environment).
		Str("release", release).
		Msg("error tracking initialized")

	return nil
}

// Example main.go integration
func ExampleMain() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize error tracking EARLY in startup
	if err := InitializeErrorTracking(&logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize error tracking")
	}

	// Ensure all events are flushed before shutdown
	defer func() {
		logger.Info().Msg("flushing error tracking events...")
		if !sentry.Flush(5 * time.Second) {
			logger.Warn().Msg("error tracking flush timed out")
		}
	}()

	// ... rest of application startup ...
}

// ============================================================================
// 2. RECOVERY MIDDLEWARE INTEGRATION
// ============================================================================

// RecoveryMiddlewareWithSentry enhances panic recovery with error tracking
func RecoveryMiddlewareWithSentry(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					// 1. Capture stack trace
					stackTrace := debug.Stack()

					// 2. Extract request context
					requestID := getRequestIDFromContext(r.Context())
					userID := getUserIDFromContext(r.Context())
					clientIP := getClientIP(r)

					// 3. Build error message
					var err error
					if e, ok := rvr.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("panic: %v", rvr)
					}

					// 4. Log to zerolog (existing behavior)
					logger.Error().
						Str("request_id", requestID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_addr", clientIP).
						Str("panic", err.Error()).
						Bytes("stack_trace", stackTrace).
						Str("user_id", userID).
						Msg("panic recovered in http handler")

					// 5. Capture to Sentry/GlitchTip
					hub := sentry.GetHubFromContext(r.Context())
					if hub == nil {
						hub = sentry.CurrentHub().Clone()
					}

					// Configure scope with context
					hub.ConfigureScope(func(scope *sentry.Scope) {
						// Tags for filtering
						scope.SetTag("request_id", requestID)
						scope.SetTag("http.method", r.Method)
						scope.SetTag("http.path", r.URL.Path)
						scope.SetTag("handler", "panic_recovery")

						// User identification (user ID only, no email)
						if userID != "" {
							scope.SetUser(sentry.User{
								ID:        userID,
								IPAddress: maskIP(clientIP), // Masked IP
							})
						}

						// Request context
						scope.SetRequest(r)

						// Severity level
						scope.SetLevel(sentry.LevelFatal)

						// Additional context
						scope.SetContext("panic", map[string]interface{}{
							"value":      fmt.Sprintf("%v", rvr),
							"stacktrace": string(stackTrace),
						})
					})

					// Capture the panic
					eventID := hub.RecoverWithContext(r.Context(), err)

					logger.Info().
						Str("sentry_event_id", string(*eventID)).
						Msg("panic reported to error tracking")

					// 6. Return generic error response (don't leak stack trace)
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{
						"type": "https://api.goimg.dev/problems/internal-error",
						"title": "Internal Server Error",
						"status": 500,
						"detail": "An unexpected error occurred. Please try again later.",
						"traceId": "%s"
					}`, requestID)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================================
// 3. HANDLER ERROR CAPTURE
// ============================================================================

// ExampleImageUploadHandler demonstrates error capture in handlers
func ExampleImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start transaction for performance monitoring
	span := sentry.StartSpan(ctx, "http.server",
		sentry.TransactionName("POST /api/v1/images"),
	)
	defer span.Finish()

	// Use span context for downstream operations
	ctx = span.Context()

	// Parse request
	var req ImageUploadRequest
	if err := parseRequest(r, &req); err != nil {
		// User error - don't report to Sentry
		respondError(w, r, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Execute upload command
	uploadSpan := sentry.StartSpan(ctx, "image.upload")
	image, err := uploadImageCommand(ctx, req)
	uploadSpan.Finish()

	if err != nil {
		// Classify error: user error vs system error
		if isUserError(err) {
			// User errors: validation, not found, conflict
			// Log but don't report to Sentry
			respondError(w, r, http.StatusBadRequest, "Upload failed", err.Error())
			return
		}

		// System errors: database failures, storage errors, etc.
		// Report to Sentry for investigation
		captureError(ctx, err, map[string]interface{}{
			"handler":      "image.upload",
			"filename":     req.Filename,
			"content_type": req.ContentType,
			"size_bytes":   req.Size,
		})

		// Return generic error (don't leak internal details)
		respondError(w, r, http.StatusInternalServerError,
			"Upload failed",
			"An unexpected error occurred. Please try again later.")
		return
	}

	// Success response
	respondJSON(w, http.StatusCreated, image)
}

// captureError reports an error to Sentry with context
func captureError(ctx context.Context, err error, extra map[string]interface{}) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		// Extract context values
		requestID := getRequestIDFromContext(ctx)
		userID := getUserIDFromContext(ctx)

		// Set tags
		scope.SetTag("request_id", requestID)
		if handler, ok := extra["handler"].(string); ok {
			scope.SetTag("handler", handler)
		}

		// Set user
		if userID != "" {
			scope.SetUser(sentry.User{
				ID: userID,
			})
		}

		// Set error level based on error type
		scope.SetLevel(classifyErrorSeverity(err))

		// Add extra context
		if extra != nil {
			scope.SetContext("error_context", extra)
		}

		// Add breadcrumbs for debugging
		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "error",
			Message:  "Application error occurred",
			Level:    sentry.LevelError,
			Data:     extra,
		}, 100)
	})

	// Capture the error
	hub.CaptureException(err)
}

// classifyErrorSeverity determines Sentry severity level
func classifyErrorSeverity(err error) sentry.Level {
	// Critical: data loss, security issues
	if isDataLossError(err) || isSecurityError(err) {
		return sentry.LevelFatal
	}

	// Error: operational failures (database, storage, external APIs)
	if isInfrastructureError(err) {
		return sentry.LevelError
	}

	// Warning: recoverable issues
	return sentry.LevelWarning
}

// isUserError checks if error is caused by user input (don't report)
func isUserError(err error) bool {
	// Domain-specific error types
	return errors.Is(err, ErrValidation) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrForbidden)
}

// ============================================================================
// 4. PII SCRUBBING IMPLEMENTATION
// ============================================================================

var (
	emailRegex       = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	passwordRegex    = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api[_-]?key)["\s:=]+[^\s"&]+`)
	bearerTokenRegex = regexp.MustCompile(`Bearer\s+[A-Za-z0-9\-\._~\+\/]+=*`)
	creditCardRegex  = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)

	sensitiveHeaders = map[string]bool{
		"authorization":  true,
		"cookie":         true,
		"set-cookie":     true,
		"x-api-key":      true,
		"x-auth-token":   true,
		"x-csrf-token":   true,
		"x-access-token": true,
	}

	sensitiveFields = map[string]bool{
		"password":        true,
		"passwd":          true,
		"pwd":             true,
		"secret":          true,
		"api_key":         true,
		"token":           true,
		"access_token":    true,
		"refresh_token":   true,
		"credit_card":     true,
		"cvv":             true,
		"ssn":             true,
	}
)

// scrubPII removes PII from Sentry events before sending
func scrubPII(event *sentry.Event) *sentry.Event {
	// 1. Scrub exception messages
	for i := range event.Exception {
		event.Exception[i].Value = scrubString(event.Exception[i].Value)
	}

	// 2. Scrub log message
	if event.Message != "" {
		event.Message = scrubString(event.Message)
	}

	// 3. Scrub breadcrumbs
	for i := range event.Breadcrumbs {
		event.Breadcrumbs[i].Message = scrubString(event.Breadcrumbs[i].Message)
		if event.Breadcrumbs[i].Data != nil {
			event.Breadcrumbs[i].Data = scrubMap(event.Breadcrumbs[i].Data)
		}
	}

	// 4. Scrub request data
	if event.Request != nil {
		event.Request = scrubRequest(event.Request)
	}

	// 5. Scrub user data
	if event.User.IPAddress != "" {
		event.User.IPAddress = maskIP(event.User.IPAddress)
	}
	event.User.Email = ""      // Never send email
	event.User.Username = ""   // Never send username

	// 6. Scrub extra context
	if event.Extra != nil {
		event.Extra = scrubMap(event.Extra)
	}

	// 7. Scrub contexts
	for key, context := range event.Contexts {
		if contextMap, ok := context.(map[string]interface{}); ok {
			event.Contexts[key] = scrubMap(contextMap)
		}
	}

	return event
}

// scrubString removes PII patterns from strings
func scrubString(s string) string {
	s = emailRegex.ReplaceAllString(s, "[EMAIL_REDACTED]")
	s = passwordRegex.ReplaceAllString(s, "$1=[REDACTED]")
	s = bearerTokenRegex.ReplaceAllString(s, "Bearer [TOKEN_REDACTED]")
	s = creditCardRegex.ReplaceAllString(s, "[CARD_REDACTED]")
	return s
}

// scrubMap removes sensitive keys and values from maps
func scrubMap(m map[string]interface{}) map[string]interface{} {
	scrubbed := make(map[string]interface{})

	for key, value := range m {
		lowerKey := strings.ToLower(key)

		// Check if key is sensitive
		if sensitiveFields[lowerKey] || sensitiveHeaders[lowerKey] {
			scrubbed[key] = "[REDACTED]"
			continue
		}

		// Recursively scrub nested maps
		switch v := value.(type) {
		case map[string]interface{}:
			scrubbed[key] = scrubMap(v)
		case string:
			scrubbed[key] = scrubString(v)
		default:
			scrubbed[key] = value
		}
	}

	return scrubbed
}

// scrubRequest removes PII from HTTP request data
func scrubRequest(req *sentry.Request) *sentry.Request {
	// Scrub headers
	if req.Headers != nil {
		scrubbedHeaders := make(map[string]string)
		for key, value := range req.Headers {
			if sensitiveHeaders[strings.ToLower(key)] {
				scrubbedHeaders[key] = "[REDACTED]"
			} else {
				scrubbedHeaders[key] = scrubString(value)
			}
		}
		req.Headers = scrubbedHeaders
	}

	// Scrub cookies
	req.Cookies = ""

	// Scrub query string
	if req.QueryString != "" {
		req.QueryString = scrubString(req.QueryString)
	}

	// Scrub POST data
	if req.Data != nil {
		if dataMap, ok := req.Data.(map[string]interface{}); ok {
			req.Data = scrubMap(dataMap)
		} else if dataStr, ok := req.Data.(string); ok {
			req.Data = scrubString(dataStr)
		}
	}

	return req
}

// maskIP masks the last octet of IPv4 or last 64 bits of IPv6
func maskIP(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "[INVALID_IP]"
	}

	// IPv4: mask last octet
	if ipv4 := parsedIP.To4(); ipv4 != nil {
		ipv4[3] = 0
		return ipv4.String() // e.g., 192.168.1.0
	}

	// IPv6: mask last 64 bits
	if ipv6 := parsedIP.To16(); ipv6 != nil {
		for i := 8; i < 16; i++ {
			ipv6[i] = 0
		}
		return parsedIP.String() // e.g., 2001:db8::/64
	}

	return "[INVALID_IP]"
}

// ============================================================================
// 5. BREADCRUMBS FOR DEBUGGING
// ============================================================================

// AddBreadcrumb adds a breadcrumb for debugging context
func AddBreadcrumb(ctx context.Context, category, message string, data map[string]interface{}) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		return
	}

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category:  category,
		Message:   message,
		Data:      data,
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}, 100)
}

// Example usage in handler
func ExampleHandlerWithBreadcrumbs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add breadcrumb for request start
	AddBreadcrumb(ctx, "http", "Processing image upload", map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})

	// Parse request
	var req ImageUploadRequest
	if err := parseRequest(r, &req); err != nil {
		AddBreadcrumb(ctx, "validation", "Request parsing failed", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, r, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	AddBreadcrumb(ctx, "validation", "Request parsed successfully", map[string]interface{}{
		"filename": req.Filename,
		"size":     req.Size,
	})

	// Validate file
	if err := validateFile(req); err != nil {
		AddBreadcrumb(ctx, "validation", "File validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		respondError(w, r, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	AddBreadcrumb(ctx, "validation", "File validated", nil)

	// Upload to storage
	AddBreadcrumb(ctx, "storage", "Starting upload", map[string]interface{}{
		"provider": "s3",
	})

	url, err := uploadToStorage(ctx, req)
	if err != nil {
		AddBreadcrumb(ctx, "storage", "Upload failed", map[string]interface{}{
			"error": err.Error(),
		})
		captureError(ctx, err, map[string]interface{}{
			"handler":  "image.upload",
			"filename": req.Filename,
		})
		respondError(w, r, http.StatusInternalServerError, "Upload failed", "Storage error")
		return
	}

	AddBreadcrumb(ctx, "storage", "Upload completed", map[string]interface{}{
		"url": url,
	})

	// If error occurs, Sentry will show all breadcrumbs leading up to the error
	respondJSON(w, http.StatusCreated, map[string]string{"url": url})
}

// ============================================================================
// 6. PERFORMANCE MONITORING
// ============================================================================

// MonitorDatabaseQuery wraps database queries with performance tracking
func MonitorDatabaseQuery(ctx context.Context, queryName string, fn func() error) error {
	span := sentry.StartSpan(ctx, "db.query",
		sentry.TransactionName(queryName),
	)
	defer span.Finish()

	// Execute query
	err := fn()

	// Set status based on result
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetTag("error", "true")
	} else {
		span.Status = sentry.SpanStatusOK
	}

	return err
}

// Example usage
func ExampleDatabaseQuery(ctx context.Context, userID string) (*User, error) {
	var user *User

	err := MonitorDatabaseQuery(ctx, "users.find_by_id", func() error {
		var err error
		user, err = db.FindUserByID(ctx, userID)
		return err
	})

	return user, err
}

// ============================================================================
// 7. CUSTOM METRICS
// ============================================================================

// TrackCustomMetric tracks custom business metrics
func TrackCustomMetric(ctx context.Context, metricName string, value float64, tags map[string]string) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		return
	}

	// Add as breadcrumb (GlitchTip doesn't have native metrics)
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "metric",
		Message:  metricName,
		Data: map[string]interface{}{
			"value": value,
			"tags":  tags,
		},
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}, 100)
}

// Example: Track image upload metrics
func ExampleImageUploadWithMetrics(ctx context.Context, req ImageUploadRequest) error {
	start := time.Now()

	// ... upload logic ...

	duration := time.Since(start).Seconds()

	TrackCustomMetric(ctx, "image.upload.duration", duration, map[string]string{
		"provider":     "s3",
		"content_type": req.ContentType,
	})

	TrackCustomMetric(ctx, "image.upload.size", float64(req.Size), map[string]string{
		"provider": "s3",
	})

	return nil
}

// ============================================================================
// HELPER TYPES AND FUNCTIONS (for example purposes)
// ============================================================================

type ImageUploadRequest struct {
	Filename    string
	ContentType string
	Size        int64
}

type User struct {
	ID string
}

var (
	ErrValidation   = errors.New("validation error")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

func getRequestIDFromContext(ctx context.Context) string { return "req-123" }
func getUserIDFromContext(ctx context.Context) string    { return "user-456" }
func getClientIP(r *http.Request) string                 { return r.RemoteAddr }
func getHostname() string                                 { hostname, _ := os.Hostname(); return hostname }
func parseRequest(r *http.Request, req interface{}) error { return nil }
func respondError(w http.ResponseWriter, r *http.Request, status int, title, detail string) {}
func respondJSON(w http.ResponseWriter, status int, data interface{})                        {}
func uploadImageCommand(ctx context.Context, req ImageUploadRequest) (interface{}, error)   { return nil, nil }
func isDataLossError(err error) bool                                                         { return false }
func isSecurityError(err error) bool                                                         { return false }
func isInfrastructureError(err error) bool                                                   { return false }
func validateFile(req ImageUploadRequest) error                                              { return nil }
func uploadToStorage(ctx context.Context, req ImageUploadRequest) (string, error)           { return "", nil }

var db interface {
	FindUserByID(ctx context.Context, id string) (*User, error)
}
