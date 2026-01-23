package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// regexUUID matches standard UUID format (8-4-4-4-12 hex digits)
	regexUUID = regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)

	// regexID matches numeric IDs (using word boundaries to avoid matching versions like v1)
	regexID = regexp.MustCompile(`\b\d+\b`)
)

// MetricsCollector holds all Prometheus metrics for the application.
// It provides centralized metric registration and collection.
type MetricsCollector struct {
	// HTTP request metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge
	httpRequestSize      *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec

	// Image upload metrics
	imageUploadsTotal       *prometheus.CounterVec
	imageProcessingDuration *prometheus.HistogramVec

	// Storage metrics
	storageOperationsTotal *prometheus.CounterVec

	// Business metrics
	usersTotal          prometheus.Gauge
	imagesTotal         prometheus.Gauge
	activeSessionsTotal prometheus.Gauge
	storageBytesUsed    prometheus.Gauge

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbConnectionsMax    prometheus.Gauge

	// Redis metrics
	redisConnectionsActive prometheus.Gauge
	redisHits              *prometheus.CounterVec
	redisMisses            *prometheus.CounterVec

	// Security metrics
	authFailuresTotal        *prometheus.CounterVec
	rateLimitExceededTotal   *prometheus.CounterVec
	authorizationDeniedTotal *prometheus.CounterVec
	malwareDetectedTotal     *prometheus.CounterVec

	// Sprint 10: Timing attack mitigation metrics
	authLoginDelaySeconds *prometheus.HistogramVec

	// Sprint 10: HIBP password check metrics
	hibpChecksTotal          *prometheus.CounterVec
	hibpCheckDurationSeconds *prometheus.HistogramVec
}

// NewMetricsCollector creates and registers all application metrics with Prometheus.
// Uses promauto to automatically register metrics with the default registry.
//
// Metrics are organized by subsystem:
//   - http: HTTP server metrics (requests, latency, in-flight)
//   - image: Image processing metrics (uploads, processing time)
//   - database: PostgreSQL connection pool metrics
//   - redis: Redis connection and cache metrics
//
//nolint:funlen // Metrics collector initialization with Prometheus.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// HTTP Metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests, labeled by method, path, and status code",
			},
			[]string{"method", "path", "status"},
		),

		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency in seconds",
				// Buckets: 1ms, 5ms, 10ms, 50ms, 100ms, 500ms, 1s, 5s, 10s
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			},
			[]string{"method", "path", "status"},
		),

		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being served",
			},
		),

		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "http",
				Name:      "request_size_bytes",
				Help:      "HTTP request size in bytes",
				// Buckets: 1KB, 10KB, 100KB, 1MB, 10MB, 100MB
				Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600},
			},
			[]string{"method", "path"},
		),

		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes",
				// Buckets: 1KB, 10KB, 100KB, 1MB, 10MB, 100MB
				Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600},
			},
			[]string{"method", "path", "status"},
		),

		// Image Processing Metrics
		imageUploadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "image",
				Name:      "uploads_total",
				Help:      "Total number of image uploads, labeled by status and format",
			},
			[]string{"status", "format"},
		),

		imageProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "image",
				Name:      "processing_duration_seconds",
				Help:      "Image processing duration in seconds (resize, thumbnail generation, etc.)",
				// Buckets: 10ms, 50ms, 100ms, 500ms, 1s, 5s, 10s, 30s
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30},
			},
			[]string{"operation"},
		),

		// Storage Metrics
		storageOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "storage",
				Name:      "operations_total",
				Help:      "Total number of storage operations, labeled by provider, operation, and status",
			},
			[]string{"provider", "operation", "status"},
		),

		// Business Metrics
		usersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Name:      "users",
				Help:      "Total number of registered users",
			},
		),

		imagesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Name:      "images",
				Help:      "Total number of uploaded images",
			},
		),

		activeSessionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Name:      "active_sessions",
				Help:      "Number of active user sessions",
			},
		),

		storageBytesUsed: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Name:      "storage_bytes_used",
				Help:      "Total storage space used in bytes",
			},
		),

		// Database Metrics
		dbConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Subsystem: "database",
				Name:      "connections_active",
				Help:      "Number of active database connections currently in use",
			},
		),

		dbConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Subsystem: "database",
				Name:      "connections_idle",
				Help:      "Number of idle database connections in the pool",
			},
		),

		dbConnectionsMax: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Subsystem: "database",
				Name:      "connections_max",
				Help:      "Maximum number of open database connections allowed",
			},
		),

		// Redis Metrics
		redisConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "goimg",
				Subsystem: "redis",
				Name:      "connections_active",
				Help:      "Number of active Redis connections from the pool",
			},
		),

		redisHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "redis",
				Name:      "cache_hits_total",
				Help:      "Total number of Redis cache hits",
			},
			[]string{"operation"},
		),

		redisMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "redis",
				Name:      "cache_misses_total",
				Help:      "Total number of Redis cache misses",
			},
			[]string{"operation"},
		),

		// Security Metrics
		authFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "security",
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures, labeled by reason",
			},
			[]string{"reason"},
		),

		rateLimitExceededTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "security",
				Name:      "rate_limit_violations_total",
				Help:      "Total number of rate limit violations, labeled by scope",
			},
			[]string{"scope"},
		),

		authorizationDeniedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "security",
				Name:      "authorization_denied_total",
				Help:      "Total number of authorization failures (privilege escalation attempts)",
			},
			[]string{"role", "required_permission"},
		),

		malwareDetectedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Name:      "malware_detections_total",
				Help:      "Total number of malware detections",
			},
			[]string{},
		),

		// Sprint 10: Timing attack mitigation
		authLoginDelaySeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "auth",
				Name:      "login_delay_seconds",
				Help:      "Random delay applied to login attempts to prevent timing attacks",
				// Buckets: 100ms, 150ms, 200ms, 250ms, 300ms
				Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
			},
			[]string{},
		),

		// Sprint 10: HIBP password checks
		hibpChecksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "goimg",
				Subsystem: "security",
				Name:      "hibp_checks_total",
				Help:      "Total number of HIBP password checks, labeled by result",
			},
			[]string{"result"},
		),

		hibpCheckDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "goimg",
				Subsystem: "security",
				Name:      "hibp_check_duration_seconds",
				Help:      "Duration of HIBP password checks in seconds",
				// Buckets: 1ms, 5ms, 10ms, 50ms, 100ms, 500ms, 1s, 2s
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
			},
			[]string{"cache_hit"},
		),
	}
}

// MetricsMiddleware wraps HTTP handlers to automatically collect request metrics.
// It records:
//   - Request count (by method, path, status)
//   - Request duration (histogram)
//   - In-flight requests (gauge)
//   - Request and response sizes
//
// This middleware should be placed early in the middleware chain (after RequestID
// but before authentication) to capture all requests including auth failures.
//
// Usage:
//
//	collector := middleware.NewMetricsCollector()
//	r.Use(middleware.MetricsMiddleware(collector))
func MetricsMiddleware(collector *MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment in-flight requests
			collector.httpRequestsInFlight.Inc()
			defer collector.httpRequestsInFlight.Dec()

			// Wrap response writer to capture status and size
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status
			}

			// Record start time for duration calculation
			start := time.Now()

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Normalize path for metrics (remove dynamic path parameters)
			path := normalizePathForMetrics(r)
			method := r.Method
			status := strconv.Itoa(wrapped.statusCode)

			// Record request size
			if r.ContentLength > 0 {
				collector.httpRequestSize.WithLabelValues(
					method,
					path,
				).Observe(float64(r.ContentLength))
			}

			// Record metrics
			collector.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
			collector.httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
			collector.httpResponseSize.WithLabelValues(method, path, status).Observe(float64(wrapped.bytesWritten))
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and bytes written.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (mrw *metricsResponseWriter) WriteHeader(statusCode int) {
	if !mrw.wroteHeader {
		mrw.statusCode = statusCode
		mrw.wroteHeader = true
		mrw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !mrw.wroteHeader {
		mrw.WriteHeader(http.StatusOK)
	}
	n, err := mrw.ResponseWriter.Write(b)
	mrw.bytesWritten += int64(n)
	if err != nil {
		return n, fmt.Errorf("response write: %w", err)
	}
	return n, nil
}

// normalizePathForMetrics converts dynamic paths to static labels for Prometheus.
// This prevents cardinality explosion from path parameters like UUIDs.
//
// Examples:
//
//	/api/v1/users/123e4567-e89b-12d3-a456-426614174000 → /api/v1/users/:id
//	/api/v1/images/abc123/comments → /api/v1/images/:id/comments
//	/health → /health (no change)
//
// Path normalization rules:
//   - UUID patterns → :id
//   - Numeric IDs → :id
//   - Preserve static paths (health, metrics, etc.)
func normalizePathForMetrics(r *http.Request) string {
	path := r.URL.Path

	// Common static paths that should not be normalized
	switch path {
	case "/health", "/health/ready", "/metrics":
		return path
	}

	// Try to get chi route pattern from context
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}

	// Fallback: Regex-based normalization
	// Replace UUIDs first
	normalized := regexUUID.ReplaceAllString(path, ":id")
	// Replace numeric IDs
	normalized = regexID.ReplaceAllString(normalized, ":id")

	return normalized
}

// RecordImageUpload records an image upload metric.
// Call this from the image upload handler after successful upload.
//
// Parameters:
//   - success: true if upload succeeded, false if failed
//   - format: Image format (e.g., "jpeg", "png", "gif", "webp")
func (mc *MetricsCollector) RecordImageUpload(success bool, format string) {
	status := "success"
	if !success {
		status = "failure"
	}
	mc.imageUploadsTotal.WithLabelValues(status, format).Inc()
}

// RecordImageProcessing records image processing duration.
// Call this after image processing operations (resize, thumbnail, etc.)
//
// Parameters:
//   - operation: Type of processing ("resize", "thumbnail", "optimize", etc.)
//   - duration: Processing duration in seconds
func (mc *MetricsCollector) RecordImageProcessing(operation string, duration float64) {
	mc.imageProcessingDuration.WithLabelValues(operation).Observe(duration)
}

// UpdateDatabaseStats updates database connection pool metrics.
// Call this periodically (e.g., every 30 seconds) from a background goroutine.
//
// Parameters:
//   - active: Number of connections currently in use
//   - idle: Number of idle connections in the pool
//   - maxConn: Maximum allowed connections (configured)
func (mc *MetricsCollector) UpdateDatabaseStats(active, idle, maxConn int) {
	mc.dbConnectionsActive.Set(float64(active))
	mc.dbConnectionsIdle.Set(float64(idle))
	mc.dbConnectionsMax.Set(float64(maxConn))
}

// UpdateRedisStats updates Redis connection pool metrics.
// Call this periodically (e.g., every 30 seconds) from a background goroutine.
//
// Parameters:
//   - active: Number of active connections from the pool
func (mc *MetricsCollector) UpdateRedisStats(active int) {
	mc.redisConnectionsActive.Set(float64(active))
}

// RecordCacheHit records a Redis cache hit.
//
// Parameters:
//   - operation: Type of cache operation ("get", "set", "delete", etc.)
func (mc *MetricsCollector) RecordCacheHit(operation string) {
	mc.redisHits.WithLabelValues(operation).Inc()
}

// RecordCacheMiss records a Redis cache miss.
//
// Parameters:
//   - operation: Type of cache operation ("get", "set", "delete", etc.)
func (mc *MetricsCollector) RecordCacheMiss(operation string) {
	mc.redisMisses.WithLabelValues(operation).Inc()
}

// RecordStorageOperation records a storage operation metric.
// Call this after storage operations (upload, download, delete, etc.)
//
// Parameters:
//   - provider: Storage provider ("local", "s3", "do_spaces", "backblaze", "ipfs")
//   - operation: Operation type ("upload", "download", "delete", "exists", "stat")
//   - success: true if operation succeeded, false if failed
func (mc *MetricsCollector) RecordStorageOperation(provider, operation string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	mc.storageOperationsTotal.WithLabelValues(provider, operation, status).Inc()
}

// UpdateBusinessMetrics updates business-level metrics (users, images, sessions, storage).
// Call this periodically (e.g., every 60 seconds) from a background goroutine.
//
// Parameters:
//   - usersTotal: Total number of registered users
//   - imagesTotal: Total number of uploaded images
//   - activeSessions: Number of active user sessions
//   - storageBytesUsed: Total storage space used in bytes
func (mc *MetricsCollector) UpdateBusinessMetrics(
	usersTotal, imagesTotal, activeSessions int64,
	storageBytesUsed int64,
) {
	mc.usersTotal.Set(float64(usersTotal))
	mc.imagesTotal.Set(float64(imagesTotal))
	mc.activeSessionsTotal.Set(float64(activeSessions))
	mc.storageBytesUsed.Set(float64(storageBytesUsed))
}

// RecordAuthFailure records an authentication failure.
// Call this when authentication fails for any reason.
//
// Parameters:
//   - reason: Reason for failure ("invalid_credentials", "token_expired", "token_invalid", "account_locked", etc.)
func (mc *MetricsCollector) RecordAuthFailure(reason string) {
	mc.authFailuresTotal.WithLabelValues(reason).Inc()
}

// RecordRateLimitExceeded records a rate limit violation.
// Call this when a client exceeds rate limits.
//
// Parameters:
//   - scope: Rate limit scope ("global", "auth", "login", "upload")
func (mc *MetricsCollector) RecordRateLimitExceeded(scope string) {
	mc.rateLimitExceededTotal.WithLabelValues(scope).Inc()
}

// RecordAuthorizationDenied records an authorization failure (privilege escalation attempt).
// Call this when a user attempts to access a resource without proper permissions.
//
// Parameters:
//   - role: User's current role
//   - requiredPermission: Permission that was required (e.g., "admin", "moderator", "image:delete")
func (mc *MetricsCollector) RecordAuthorizationDenied(role, requiredPermission string) {
	mc.authorizationDeniedTotal.WithLabelValues(role, requiredPermission).Inc()
}

// RecordMalwareDetection records a malware detection event.
// Call this when ClamAV or another scanner detects malware.
func (mc *MetricsCollector) RecordMalwareDetection() {
	mc.malwareDetectedTotal.WithLabelValues().Inc()
}

// RecordLoginDelay records the random delay applied to a login attempt.
// Call this from the login handler after applying timing attack mitigation delay.
//
// Parameters:
//   - delaySeconds: The delay duration in seconds (e.g., 0.15 for 150ms)
//
// Sprint 10: Timing attack mitigation metric
func (mc *MetricsCollector) RecordLoginDelay(delaySeconds float64) {
	mc.authLoginDelaySeconds.WithLabelValues().Observe(delaySeconds)
}

// RecordHIBPCheck records a Have I Been Pwned password check.
// Call this from the password validation logic after checking HIBP.
//
// Parameters:
//   - result: Result of the check - one of:
//   - "clean": Password is not compromised
//   - "compromised": Password found in HIBP database
//   - "error": Check failed due to error
//   - "skipped": Check was skipped (e.g., feature disabled)
//
// Sprint 10: HIBP password check metric
func (mc *MetricsCollector) RecordHIBPCheck(result string) {
	mc.hibpChecksTotal.WithLabelValues(result).Inc()
}

// RecordHIBPCheckDuration records the duration of a HIBP password check.
// Call this after performing the HIBP check to track latency.
//
// Parameters:
//   - durationSeconds: The check duration in seconds
//   - cacheHit: true if result came from cache, false if API call was made
//
// Sprint 10: HIBP password check metric
func (mc *MetricsCollector) RecordHIBPCheckDuration(durationSeconds float64, cacheHit bool) {
	cacheHitLabel := "false"
	if cacheHit {
		cacheHitLabel = "true"
	}
	mc.hibpCheckDurationSeconds.WithLabelValues(cacheHitLabel).Observe(durationSeconds)
}
