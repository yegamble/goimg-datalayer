package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// Mock implementations for testing

type mockStorage struct {
	existsError error
}

func (m *mockStorage) Put(ctx context.Context, key string, data io.Reader, size int64, opts storage.PutOptions) error {
	return nil
}

func (m *mockStorage) PutBytes(ctx context.Context, key string, data []byte, opts storage.PutOptions) error {
	return nil
}

func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockStorage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsError != nil {
		return false, m.existsError
	}
	return false, nil
}

func (m *mockStorage) URL(key string) string {
	return "http://example.com/" + key
}

func (m *mockStorage) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	return "", storage.ErrNotSupported
}

func (m *mockStorage) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	return nil, nil
}

func (m *mockStorage) Provider() string {
	return "mock"
}

type mockClamAV struct {
	pingError error
}

func (m *mockClamAV) Scan(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
	return &clamav.ScanResult{Clean: true}, nil
}

func (m *mockClamAV) ScanReader(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error) {
	return &clamav.ScanResult{Clean: true}, nil
}

func (m *mockClamAV) Ping(ctx context.Context) error {
	return m.pingError
}

func (m *mockClamAV) Version(ctx context.Context) (string, error) {
	return "ClamAV 1.0.0", nil
}

func (m *mockClamAV) Stats(ctx context.Context) (string, error) {
	return "stats", nil
}

// Tests

func TestHealthHandler_Liveness(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()
	handler := NewHealthHandler(nil, nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Liveness(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var response LivenessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.NotEmpty(t, response.Timestamp)
}

func TestHealthHandler_Readiness_AllHealthy(t *testing.T) {
	// This is a unit test with mocked dependencies
	logger := zerolog.Nop()

	// Create mock dependencies (all healthy)
	mockDB := &sqlx.DB{} // Will fail health check but we'll handle it
	mockStore := &mockStorage{existsError: nil}
	mockClamav := &mockClamAV{pingError: nil}

	handler := NewHealthHandler(mockDB, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	// Note: This will return "down" because mockDB is not a real connection
	// For proper testing, we need integration tests with real DB
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Verify response structure
	assert.NotEmpty(t, response.Status)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotNil(t, response.Checks)

	// Should have all 4 checks
	assert.Contains(t, response.Checks, "database")
	assert.Contains(t, response.Checks, "redis")
	assert.Contains(t, response.Checks, "storage")
	assert.Contains(t, response.Checks, "clamav")
}

func TestHealthHandler_Readiness_RedisDegradation(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	mockDB := &sqlx.DB{}
	mockStore := &mockStorage{existsError: nil}
	mockClamav := &mockClamAV{pingError: nil}

	// Redis is nil, simulating connection failure
	handler := NewHealthHandler(mockDB, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Redis should be down
	assert.Equal(t, "down", response.Checks["redis"].Status)
	assert.NotEmpty(t, response.Checks["redis"].Error)
}

func TestHealthHandler_Readiness_StorageDown(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	mockDB := &sqlx.DB{}
	mockStore := &mockStorage{existsError: errors.New("storage unavailable")}
	mockClamav := &mockClamAV{pingError: nil}

	handler := NewHealthHandler(mockDB, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Storage should be down
	assert.Equal(t, "down", response.Checks["storage"].Status)
	assert.NotEmpty(t, response.Checks["storage"].Error)

	// Overall status should be "down" (critical dependency)
	assert.Equal(t, "down", response.Status)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealthHandler_Readiness_ClamAVDown(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	mockDB := &sqlx.DB{}
	mockStore := &mockStorage{existsError: nil}
	mockClamav := &mockClamAV{pingError: errors.New("clamav daemon not responding")}

	handler := NewHealthHandler(mockDB, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// ClamAV should be down
	assert.Equal(t, "down", response.Checks["clamav"].Status)
	assert.NotEmpty(t, response.Checks["clamav"].Error)

	// Overall status should be "down" (critical dependency)
	assert.Equal(t, "down", response.Status)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealthHandler_Readiness_ResponseStructure(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	mockStore := &mockStorage{existsError: nil}
	mockClamav := &mockClamAV{pingError: nil}

	handler := NewHealthHandler(nil, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert - response should have correct structure
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Status field should be present and valid
	assert.NotEmpty(t, response.Status)
	assert.Contains(t, []string{"ok", "degraded", "down"}, response.Status)

	// Timestamp should be present and in RFC3339 format
	assert.NotEmpty(t, response.Timestamp)
	_, err = time.Parse(time.RFC3339, response.Timestamp)
	assert.NoError(t, err, "Timestamp should be in RFC3339 format")

	// Checks should be a map
	assert.NotNil(t, response.Checks)
	assert.IsType(t, map[string]CheckDetails{}, response.Checks)

	// Should have all 4 checks
	assert.Contains(t, response.Checks, "database")
	assert.Contains(t, response.Checks, "redis")
	assert.Contains(t, response.Checks, "storage")
	assert.Contains(t, response.Checks, "clamav")

	// Each check should have a status
	for name, check := range response.Checks {
		assert.NotEmpty(t, check.Status, "Check %s should have status", name)
		assert.Contains(t, []string{"up", "down"}, check.Status)

		// If down, should have error message
		if check.Status == "down" {
			assert.NotEmpty(t, check.Error, "Check %s should have error message when down", name)
		}
	}
}

func TestHealthHandler_Liveness_ResponseStructure(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()
	handler := NewHealthHandler(nil, nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Liveness(rec, req)

	// Assert
	var response LivenessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Check response structure
	assert.Equal(t, "ok", response.Status)
	assert.NotEmpty(t, response.Timestamp)

	// Verify timestamp format
	_, err = time.Parse(time.RFC3339, response.Timestamp)
	assert.NoError(t, err, "Timestamp should be in RFC3339 format")

	// Verify content type
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestHealthHandler_Readiness_MultipleFailures(t *testing.T) {
	// Arrange - both Redis (nil) and Storage down
	logger := zerolog.Nop()

	mockDB := &sqlx.DB{}
	mockStore := &mockStorage{existsError: errors.New("storage down")}
	mockClamav := &mockClamAV{pingError: nil}

	handler := NewHealthHandler(mockDB, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Both should be down
	assert.Equal(t, "down", response.Checks["redis"].Status)
	assert.Equal(t, "down", response.Checks["storage"].Status)

	// Overall status should be "down" (critical dependency failed)
	assert.Equal(t, "down", response.Status)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealthHandler_Readiness_LatencyTracking(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	mockStore := &mockStorage{existsError: nil}
	mockClamav := &mockClamAV{pingError: nil}

	handler := NewHealthHandler(nil, nil, mockStore, mockClamav, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// All healthy checks should have latency > 0
	if response.Checks["redis"].Status == "up" {
		assert.Greater(t, response.Checks["redis"].LatencyMs, float64(0))
	}
	if response.Checks["storage"].Status == "up" {
		assert.Greater(t, response.Checks["storage"].LatencyMs, float64(0))
	}
	if response.Checks["clamav"].Status == "up" {
		assert.Greater(t, response.Checks["clamav"].LatencyMs, float64(0))
	}
}
