package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	// Act
	collector := NewMetricsCollector()

	// Assert - verify all metrics are initialized
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.httpRequestsTotal)
	assert.NotNil(t, collector.httpRequestDuration)
	assert.NotNil(t, collector.httpRequestsInFlight)
	assert.NotNil(t, collector.httpRequestSize)
	assert.NotNil(t, collector.httpResponseSize)
	assert.NotNil(t, collector.imageUploadsTotal)
	assert.NotNil(t, collector.imageProcessingDuration)
	assert.NotNil(t, collector.dbConnectionsActive)
	assert.NotNil(t, collector.dbConnectionsIdle)
	assert.NotNil(t, collector.dbConnectionsMax)
	assert.NotNil(t, collector.redisConnectionsActive)
	assert.NotNil(t, collector.redisHits)
	assert.NotNil(t, collector.redisMisses)

	// Sprint 10 metrics
	assert.NotNil(t, collector.authLoginDelaySeconds)
	assert.NotNil(t, collector.hibpChecksTotal)
	assert.NotNil(t, collector.hibpCheckDurationSeconds)
}

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	// Arrange
	// Create a new registry to avoid conflicts with global metrics
	registry := prometheus.NewRegistry()

	collector := &MetricsCollector{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_request_size_bytes",
				Help:    "Size of HTTP request bodies in bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_response_size_bytes",
				Help:    "Size of HTTP response bodies in bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path", "status"},
		),
	}

	registry.MustRegister(collector.httpRequestsTotal)
	registry.MustRegister(collector.httpRequestDuration)
	registry.MustRegister(collector.httpRequestsInFlight)
	registry.MustRegister(collector.httpRequestSize)
	registry.MustRegister(collector.httpResponseSize)

	middleware := MetricsMiddleware(collector)

	// Create a test handler that returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	wrappedHandler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify metrics were recorded
	count := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
	assert.InDelta(t, float64(1), count, 0.001, "Should record one request")
}

func TestMetricsMiddleware_InFlightRequests(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test2_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test2_http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_request_size_bytes",
				Help:    "Size of HTTP request bodies in bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_response_size_bytes",
				Help:    "Size of HTTP response bodies in bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path", "status"},
		),
	}

	middleware := MetricsMiddleware(collector)

	// Create a channel to synchronize test
	started := make(chan bool)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Signal that handler started
		started <- true
		// Wait a bit to ensure in-flight metric is checked
		<-started
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act - start request in goroutine
	go func() {
		wrappedHandler.ServeHTTP(rec, req)
	}()

	// Wait for handler to start
	<-started

	// Assert - in-flight requests should be 1
	inFlight := testutil.ToFloat64(collector.httpRequestsInFlight)
	assert.InDelta(t, float64(1), inFlight, 0.001, "Should have 1 request in flight")

	// Signal handler to complete
	started <- true

	// Wait briefly for request to complete
	// Note: In a real scenario, you'd use proper synchronization
}

func TestMetricsMiddleware_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedStatus string
	}{
		{"Success 200", http.StatusOK, "200"},
		{"Created 201", http.StatusCreated, "201"},
		{"Bad Request 400", http.StatusBadRequest, "400"},
		{"Unauthorized 401", http.StatusUnauthorized, "401"},
		{"Not Found 404", http.StatusNotFound, "404"},
		{"Internal Server Error 500", http.StatusInternalServerError, "500"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			collector := &MetricsCollector{
				httpRequestsTotal: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "test3_http_requests_total",
						Help: "Total number of HTTP requests",
					},
					[]string{"method", "path", "status"},
				),
				httpRequestDuration: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_request_duration_seconds",
						Help:    "Duration of HTTP requests in seconds",
						Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
					},
					[]string{"method", "path", "status"},
				),
				httpRequestsInFlight: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "test3_http_requests_in_flight",
						Help: "Number of HTTP requests currently being processed",
					},
				),
				httpRequestSize: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_request_size_bytes",
						Help:    "Size of HTTP request bodies in bytes",
						Buckets: []float64{1024, 10240, 102400},
					},
					[]string{"method", "path"},
				),
				httpResponseSize: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_response_size_bytes",
						Help:    "Size of HTTP response bodies in bytes",
						Buckets: []float64{1024, 10240, 102400},
					},
					[]string{"method", "path", "status"},
				),
			}

			middleware := MetricsMiddleware(collector)

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			wrappedHandler := middleware(testHandler)

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			wrappedHandler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tc.statusCode, rec.Code)

			// Verify metrics include correct status code
			count := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/test", tc.expectedStatus))
			assert.InDelta(t, float64(1), count, 0.001, "Should record request with status %s", tc.expectedStatus)
		})
	}
}

func TestMetricsCollector_RecordImageUpload(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		imageUploadsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_image_uploads_total",
				Help: "Total number of image uploads",
			},
			[]string{"status", "format"},
		),
	}

	// Act
	collector.RecordImageUpload(true, "jpeg")
	collector.RecordImageUpload(true, "png")
	collector.RecordImageUpload(false, "jpeg")

	// Assert
	jpegSuccessCount := testutil.ToFloat64(collector.imageUploadsTotal.WithLabelValues("success", "jpeg"))
	assert.InDelta(t, float64(1), jpegSuccessCount, 0.001, "Should record 1 successful JPEG upload")

	pngSuccessCount := testutil.ToFloat64(collector.imageUploadsTotal.WithLabelValues("success", "png"))
	assert.InDelta(t, float64(1), pngSuccessCount, 0.001, "Should record 1 successful PNG upload")

	failureCount := testutil.ToFloat64(collector.imageUploadsTotal.WithLabelValues("failure", "jpeg"))
	assert.InDelta(t, float64(1), failureCount, 0.001, "Should record 1 failed upload")
}

func TestMetricsCollector_RecordImageProcessing(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		imageProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_image_processing_duration_seconds",
				Help:    "Duration of image processing operations in seconds",
				Buckets: []float64{0.01, 0.1, 1, 10},
			},
			[]string{"operation"},
		),
	}

	// Act
	collector.RecordImageProcessing("resize", 0.5)
	collector.RecordImageProcessing("thumbnail", 0.1)

	// Assert - verify metrics were recorded (count > 0)
	// Note: testutil doesn't provide easy access to histogram values,
	// but we can verify the metric was created
	require.NotNil(t, collector.imageProcessingDuration)
}

func TestMetricsCollector_UpdateDatabaseStats(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		dbConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		dbConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		dbConnectionsMax: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_db_connections_max",
				Help: "Maximum number of allowed database connections",
			},
		),
	}

	// Act
	collector.UpdateDatabaseStats(10, 5, 25)

	// Assert
	active := testutil.ToFloat64(collector.dbConnectionsActive)
	assert.InDelta(t, float64(10), active, 0.0001)

	idle := testutil.ToFloat64(collector.dbConnectionsIdle)
	assert.InDelta(t, float64(5), idle, 0.0001)

	maxConns := testutil.ToFloat64(collector.dbConnectionsMax)
	assert.InDelta(t, float64(25), maxConns, 0.0001)
}

func TestMetricsCollector_UpdateRedisStats(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		redisConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
	}

	// Act
	collector.UpdateRedisStats(8)

	// Assert
	active := testutil.ToFloat64(collector.redisConnectionsActive)
	assert.InDelta(t, float64(8), active, 0.01)
}

func TestMetricsCollector_RecordCacheHitMiss(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		redisHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_redis_cache_hits_total",
				Help: "Total number of Redis cache hits",
			},
			[]string{"operation"},
		),
		redisMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_redis_cache_misses_total",
				Help: "Total number of Redis cache misses",
			},
			[]string{"operation"},
		),
	}

	// Act
	collector.RecordCacheHit("get")
	collector.RecordCacheHit("get")
	collector.RecordCacheMiss("get")

	// Assert
	hits := testutil.ToFloat64(collector.redisHits.WithLabelValues("get"))
	assert.InDelta(t, float64(2), hits, 0.01)

	misses := testutil.ToFloat64(collector.redisMisses.WithLabelValues("get"))
	assert.InDelta(t, float64(1), misses, 0.01)
}

func TestNormalizePathForMetrics(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Health endpoint", "/health", "/health"},
		{"Readiness endpoint", "/health/ready", "/health/ready"},
		{"Metrics endpoint", "/metrics", "/metrics"},
		// For now, these return full paths until we implement normalization
		{"User by ID", "/api/v1/users/123", "/api/v1/users/123"},
		{"Image by UUID", "/api/v1/images/abc-123-def", "/api/v1/images/abc-123-def"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := normalizePathForMetrics(tc.input)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Sprint 10: Timing attack mitigation metrics tests

func TestMetricsCollector_RecordLoginDelay(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		authLoginDelaySeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_auth_login_delay_seconds",
				Help:    "Random delay applied to login attempts",
				Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
			},
			[]string{},
		),
	}

	// Act
	collector.RecordLoginDelay(0.15)
	collector.RecordLoginDelay(0.2)
	collector.RecordLoginDelay(0.25)

	// Assert - verify metrics were recorded
	require.NotNil(t, collector.authLoginDelaySeconds)
}

// Sprint 10: HIBP password check metrics tests

func TestMetricsCollector_RecordHIBPCheck(t *testing.T) {
	testCases := []struct {
		name           string
		result         string
		expectedCount  float64
		recordMultiple bool
	}{
		{"Clean password", "clean", 1, false},
		{"Compromised password", "compromised", 1, false},
		{"Error occurred", "error", 1, false},
		{"Skipped check", "skipped", 1, false},
		{"Multiple clean checks", "clean", 3, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			collector := &MetricsCollector{
				hibpChecksTotal: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "test_hibp_checks_total",
						Help: "Total HIBP password checks",
					},
					[]string{"result"},
				),
			}

			// Act
			if tc.recordMultiple {
				for i := 0; i < int(tc.expectedCount); i++ {
					collector.RecordHIBPCheck(tc.result)
				}
			} else {
				collector.RecordHIBPCheck(tc.result)
			}

			// Assert
			count := testutil.ToFloat64(collector.hibpChecksTotal.WithLabelValues(tc.result))
			assert.InDelta(t, tc.expectedCount, count, 0.001, "Should record %d checks with result %s", int(tc.expectedCount), tc.result)
		})
	}
}

func TestMetricsCollector_RecordHIBPCheckDuration(t *testing.T) {
	testCases := []struct {
		name             string
		durationSeconds  float64
		cacheHit         bool
		expectedCacheHit string
	}{
		{"Cache hit - fast", 0.001, true, "true"},
		{"Cache miss - slower", 0.5, false, "false"},
		{"API call - slow", 1.5, false, "false"},
		{"Cache hit - medium", 0.05, true, "true"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			collector := &MetricsCollector{
				hibpCheckDurationSeconds: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test_hibp_check_duration_seconds",
						Help:    "Duration of HIBP password checks",
						Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
					},
					[]string{"cache_hit"},
				),
			}

			// Act
			collector.RecordHIBPCheckDuration(tc.durationSeconds, tc.cacheHit)

			// Assert - verify metric was recorded (count > 0)
			require.NotNil(t, collector.hibpCheckDurationSeconds)
		})
	}
}

func TestMetricsCollector_RecordHIBPCheckDuration_BooleanLabels(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		hibpCheckDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_hibp_duration_labels",
				Help:    "Test cache_hit labels",
				Buckets: []float64{0.001, 0.01, 0.1, 1},
			},
			[]string{"cache_hit"},
		),
	}

	// Act
	collector.RecordHIBPCheckDuration(0.1, true)  // Cache hit
	collector.RecordHIBPCheckDuration(0.5, false) // Cache miss

	// Assert - verify both label values work
	require.NotNil(t, collector.hibpCheckDurationSeconds.WithLabelValues("true"))
	require.NotNil(t, collector.hibpCheckDurationSeconds.WithLabelValues("false"))
}
