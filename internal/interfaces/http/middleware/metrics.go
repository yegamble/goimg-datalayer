package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all Prometheus metrics for the application.
// It provides centralized metric registration and collection.
type MetricsCollector struct {
	// HTTP request metrics
	httpRequestsTotal       *prometheus.CounterVec
	httpRequestDuration     *prometheus.HistogramVec
	httpRequestsInFlight    prometheus.Gauge
	httpRequestSize         *prometheus.HistogramVec
	httpResponseSize        *prometheus.HistogramVec

	// Image upload metrics
	imageUploadsTotal       *prometheus.CounterVec
	imageProcessingDuration *prometheus.HistogramVec

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbConnectionsMax    prometheus.Gauge

	// Redis metrics
	redisConnectionsActive prometheus.Gauge
	redisHits              *prometheus.CounterVec
	redisMisses            *prometheus.CounterVec
}

// NewMetricsCollector creates and registers all application metrics with Prometheus.
// Uses promauto to automatically register metrics with the default registry.
//
// Metrics are organized by subsystem:
//   - http: HTTP server metrics (requests, latency, in-flight)
//   - image: Image processing metrics (uploads, processing time)
//   - database: PostgreSQL connection pool metrics
//   - redis: Redis connection and cache metrics
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
				Help:      "Total number of image uploads, labeled by status (success/failure)",
			},
			[]string{"status"},
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

			// Record request size
			if r.ContentLength > 0 {
				collector.httpRequestSize.WithLabelValues(
					r.Method,
					normalizePathForMetrics(r.URL.Path),
				).Observe(float64(r.ContentLength))
			}

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
			path := normalizePathForMetrics(r.URL.Path)
			method := r.Method
			status := strconv.Itoa(wrapped.statusCode)

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
	return n, err
}

// normalizePathForMetrics converts dynamic paths to static labels for Prometheus.
// This prevents cardinality explosion from path parameters like UUIDs.
//
// Examples:
//   /api/v1/users/123e4567-e89b-12d3-a456-426614174000 → /api/v1/users/:id
//   /api/v1/images/abc123/comments → /api/v1/images/:id/comments
//   /health → /health (no change)
//
// Path normalization rules:
//   - UUID patterns → :id
//   - Numeric IDs → :id
//   - Preserve static paths (health, metrics, etc.)
func normalizePathForMetrics(path string) string {
	// Common static paths that should not be normalized
	switch path {
	case "/health", "/health/ready", "/metrics":
		return path
	}

	// For now, return the full path
	// TODO: Implement intelligent path parameter detection
	// Options:
	//   1. Use chi's route patterns if available from context
	//   2. Regex-based UUID/number detection
	//   3. Path template matching
	//
	// For MVP, we return the full path. In production, this should be
	// normalized to prevent cardinality explosion.
	return path
}

// RecordImageUpload records an image upload metric.
// Call this from the image upload handler after successful upload.
//
// Parameters:
//   - success: true if upload succeeded, false if failed
func (mc *MetricsCollector) RecordImageUpload(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	mc.imageUploadsTotal.WithLabelValues(status).Inc()
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
//   - max: Maximum allowed connections (configured)
func (mc *MetricsCollector) UpdateDatabaseStats(active, idle, max int) {
	mc.dbConnectionsActive.Set(float64(active))
	mc.dbConnectionsIdle.Set(float64(idle))
	mc.dbConnectionsMax.Set(float64(max))
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
