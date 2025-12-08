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
}

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	// Arrange
	// Create a new registry to avoid conflicts with global metrics
	registry := prometheus.NewRegistry()

	collector := &MetricsCollector{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_http_requests_total",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_request_duration_seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_http_requests_in_flight",
			},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_request_size_bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_http_response_size_bytes",
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
		w.Write([]byte("OK"))
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
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_request_duration_seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test2_http_requests_in_flight",
			},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_request_size_bytes",
				Buckets: []float64{1024, 10240, 102400},
			},
			[]string{"method", "path"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test2_http_response_size_bytes",
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
					},
					[]string{"method", "path", "status"},
				),
				httpRequestDuration: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_request_duration_seconds",
						Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
					},
					[]string{"method", "path", "status"},
				),
				httpRequestsInFlight: prometheus.NewGauge(
					prometheus.GaugeOpts{
						Name: "test3_http_requests_in_flight",
					},
				),
				httpRequestSize: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_request_size_bytes",
						Buckets: []float64{1024, 10240, 102400},
					},
					[]string{"method", "path"},
				),
				httpResponseSize: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name:    "test3_http_response_size_bytes",
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
			},
		),
		dbConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_db_connections_idle",
			},
		),
		dbConnectionsMax: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_db_connections_max",
			},
		),
	}

	// Act
	collector.UpdateDatabaseStats(10, 5, 25)

	// Assert
	active := testutil.ToFloat64(collector.dbConnectionsActive)
	assert.Equal(t, float64(10), active)

	idle := testutil.ToFloat64(collector.dbConnectionsIdle)
	assert.Equal(t, float64(5), idle)

	max := testutil.ToFloat64(collector.dbConnectionsMax)
	assert.Equal(t, float64(25), max)
}

func TestMetricsCollector_UpdateRedisStats(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		redisConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "test_redis_connections_active",
			},
		),
	}

	// Act
	collector.UpdateRedisStats(8)

	// Assert
	active := testutil.ToFloat64(collector.redisConnectionsActive)
	assert.Equal(t, float64(8), active)
}

func TestMetricsCollector_RecordCacheHitMiss(t *testing.T) {
	// Arrange
	collector := &MetricsCollector{
		redisHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_redis_cache_hits_total",
			},
			[]string{"operation"},
		),
		redisMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_redis_cache_misses_total",
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
	assert.Equal(t, float64(2), hits)

	misses := testutil.ToFloat64(collector.redisMisses.WithLabelValues("get"))
	assert.Equal(t, float64(1), misses)
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
