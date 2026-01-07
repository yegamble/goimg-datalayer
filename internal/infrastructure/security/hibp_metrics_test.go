package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHIBPMetricsRecorder is a mock implementation of HIBPMetricsRecorder for testing.
type MockHIBPMetricsRecorder struct {
	mock.Mock
}

func (m *MockHIBPMetricsRecorder) RecordHIBPCheck(result string) {
	m.Called(result)
}

func (m *MockHIBPMetricsRecorder) RecordHIBPCheckDuration(durationSeconds float64, cacheHit bool) {
	m.Called(durationSeconds, cacheHit)
}

// MockPasswordCache is a mock implementation of PasswordCache for testing.
type MockPasswordCache struct {
	mock.Mock
}

func (m *MockPasswordCache) Get(ctx context.Context, prefix, suffix string) (bool, bool) {
	args := m.Called(ctx, prefix, suffix)
	return args.Bool(0), args.Bool(1)
}

func (m *MockPasswordCache) Set(ctx context.Context, prefix, suffix string, pwned bool, ttl time.Duration) error {
	args := m.Called(ctx, prefix, suffix, pwned, ttl)
	return args.Error(0)
}

func (m *MockPasswordCache) Delete(ctx context.Context, prefix, suffix string) error {
	args := m.Called(ctx, prefix, suffix)
	return args.Error(0)
}

// TestNoOpHIBPMetricsRecorder_RecordHIBPCheck verifies that NoOp implementation
// doesn't panic or error when called.
func TestNoOpHIBPMetricsRecorder_RecordHIBPCheck(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := &NoOpHIBPMetricsRecorder{}

	// Act - should not panic
	recorder.RecordHIBPCheck("clean")
	recorder.RecordHIBPCheck("compromised")
	recorder.RecordHIBPCheck("error")
	recorder.RecordHIBPCheck("cache_hit")
	recorder.RecordHIBPCheck("skipped")

	// Assert - just verify it implements the interface
	var _ HIBPMetricsRecorder = recorder
}

// TestNoOpHIBPMetricsRecorder_RecordHIBPCheckDuration verifies that NoOp implementation
// doesn't panic or error when called.
func TestNoOpHIBPMetricsRecorder_RecordHIBPCheckDuration(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := &NoOpHIBPMetricsRecorder{}

	// Act - should not panic
	recorder.RecordHIBPCheckDuration(0.5, true)
	recorder.RecordHIBPCheckDuration(1.2, false)
	recorder.RecordHIBPCheckDuration(0.0, false)

	// Assert - just verify it implements the interface
	var _ HIBPMetricsRecorder = recorder
}

// TestNoOpHIBPMetricsRecorder_ThreadSafety verifies that NoOp implementation
// is safe for concurrent use.
func TestNoOpHIBPMetricsRecorder_ThreadSafety(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := &NoOpHIBPMetricsRecorder{}

	// Act - call from multiple goroutines
	done := make(chan bool, 200)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			recorder.RecordHIBPCheck("clean")
			recorder.RecordHIBPCheckDuration(float64(idx)*0.01, idx%2 == 0)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Assert - no panics occurred
	assert.True(t, true)
}

// TestHIBPClient_RecordsMetrics_PasswordCompromised verifies that correct
// metrics are recorded when a password is found in breach database.
func TestHIBPClient_RecordsMetrics_PasswordCompromised(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return suffix that matches "password123"
		_, _ = w.Write([]byte("C6008F9CAB4083784CBD1874F76618D2A97:12345\n"))
	}))
	defer server.Close()

	logger := zerolog.Nop()
	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second},
		nil, // no cache
		mockMetrics,
		&logger,
	)
	client.apiURL = server.URL + "/range/"

	// Expect metrics to be recorded
	mockMetrics.On("RecordHIBPCheck", "compromised").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0 // Should be positive
	}), false).Once() // cacheHit = false

	// Act
	compromised, err := client.IsCompromised(context.Background(), "password123")

	// Assert
	require.NoError(t, err)
	assert.True(t, compromised)
	mockMetrics.AssertExpectations(t)
}

// TestHIBPClient_RecordsMetrics_PasswordClean verifies that correct
// metrics are recorded when a password is not found in breach database.
func TestHIBPClient_RecordsMetrics_PasswordClean(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return different suffixes
		_, _ = w.Write([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:123\n"))
	}))
	defer server.Close()

	logger := zerolog.Nop()
	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second},
		nil, // no cache
		mockMetrics,
		&logger,
	)
	client.apiURL = server.URL + "/range/"

	// Expect metrics to be recorded
	mockMetrics.On("RecordHIBPCheck", "clean").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0
	}), false).Once()

	// Act
	compromised, err := client.IsCompromised(context.Background(), "MyStrongP@ssw0rd!")

	// Assert
	require.NoError(t, err)
	assert.False(t, compromised)
	mockMetrics.AssertExpectations(t)
}

// TestHIBPClient_RecordsMetrics_APIError verifies that error metrics are
// recorded when HIBP API fails.
func TestHIBPClient_RecordsMetrics_APIError(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second, FailOpen: true},
		nil, // no cache
		mockMetrics,
		&logger,
	)
	client.apiURL = server.URL + "/range/"

	// Expect error metrics to be recorded
	mockMetrics.On("RecordHIBPCheck", "error").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0
	}), false).Once()

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword")

	// Assert - fail-open configuration
	require.NoError(t, err)
	assert.False(t, compromised)
	mockMetrics.AssertExpectations(t)
}

// TestHIBPClient_RecordsMetrics_CacheHit verifies that cache hit metrics are
// recorded correctly.
func TestHIBPClient_RecordsMetrics_CacheHit(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	mockCache := new(MockPasswordCache)
	logger := zerolog.Nop()

	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second, CacheTTL: 24 * time.Hour},
		mockCache,
		mockMetrics,
		&logger,
	)

	// Mock cache returning a hit for compromised password
	mockCache.On("Get", mock.Anything, "CBFDA", "C6008F9CAB4083784CBD1874F76618D2A97").
		Return(true, true) // pwned=true, found=true

	// Expect cache hit metrics
	mockMetrics.On("RecordHIBPCheck", "compromised").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0
	}), true).Once() // cacheHit = true

	// Act
	compromised, err := client.IsCompromised(context.Background(), "password123")

	// Assert
	require.NoError(t, err)
	assert.True(t, compromised)
	mockMetrics.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestHIBPClient_RecordsMetrics_CacheHitClean verifies that cache hit metrics
// are recorded for clean passwords.
func TestHIBPClient_RecordsMetrics_CacheHitClean(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	mockCache := new(MockPasswordCache)
	logger := zerolog.Nop()

	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second, CacheTTL: 24 * time.Hour},
		mockCache,
		mockMetrics,
		&logger,
	)

	// Mock cache returning clean password
	mockCache.On("Get", mock.Anything, mock.Anything, mock.Anything).
		Return(false, true) // pwned=false, found=true

	// Expect cache hit metrics for clean password
	mockMetrics.On("RecordHIBPCheck", "clean").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0
	}), true).Once() // cacheHit = true

	// Act
	compromised, err := client.IsCompromised(context.Background(), "MyStrongP@ssw0rd!")

	// Assert
	require.NoError(t, err)
	assert.False(t, compromised)
	mockMetrics.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestHIBPClient_RecordsMetrics_Disabled verifies that "skipped" metrics are
// recorded when HIBP checks are disabled.
func TestHIBPClient_RecordsMetrics_Disabled(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	logger := zerolog.Nop()

	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: false}, // Disabled
		nil,
		mockMetrics,
		&logger,
	)

	// Expect "skipped" metrics
	mockMetrics.On("RecordHIBPCheck", "skipped").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0
	}), false).Once()

	// Act
	compromised, err := client.IsCompromised(context.Background(), "anypassword")

	// Assert
	require.NoError(t, err)
	assert.False(t, compromised)
	mockMetrics.AssertExpectations(t)
}

// TestHIBPClient_WithNilMetrics_UsesNoOp verifies that passing nil metrics
// uses the NoOp implementation without panicking.
func TestHIBPClient_WithNilMetrics_UsesNoOp(t *testing.T) {
	t.Parallel()

	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second},
		nil, // no cache
		nil, // nil metrics - should use NoOp
		nil, // no logger
	)
	client.apiURL = server.URL + "/range/"

	// Act - should not panic
	compromised, err := client.IsCompromised(context.Background(), "testpassword")

	// Assert
	require.NoError(t, err)
	assert.False(t, compromised)
}

// TestHIBPClient_MetricsDurationAccurate verifies that recorded duration
// is reasonably accurate.
func TestHIBPClient_MetricsDurationAccurate(t *testing.T) {
	t.Parallel()

	// Arrange
	mockMetrics := new(MockHIBPMetricsRecorder)
	serverDelay := 100 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverDelay) // Simulate API delay
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	logger := zerolog.Nop()
	client := NewHIBPClientWithConfig(
		HIBPConfig{Enabled: true, Timeout: 5 * time.Second},
		nil,
		mockMetrics,
		&logger,
	)
	client.apiURL = server.URL + "/range/"

	// Expect duration to be at least the server delay (100ms = 0.1s)
	mockMetrics.On("RecordHIBPCheck", "clean").Once()
	mockMetrics.On("RecordHIBPCheckDuration", mock.MatchedBy(func(duration float64) bool {
		return duration >= 0.1 // At least 100ms
	}), false).Once()

	// Act
	_, _ = client.IsCompromised(context.Background(), "testpassword")

	// Assert
	mockMetrics.AssertExpectations(t)
}
