package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
)

func TestHealthHandler_Liveness(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()
	handler := NewHealthHandler(nil, nil, logger)

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
	// This is an integration test that requires real DB and Redis
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup test database
	cfg := postgres.DefaultConfig()
	cfg.Host = "localhost"
	cfg.Port = 5432
	cfg.Database = "goimg_test"
	cfg.User = "postgres"
	cfg.Password = "postgres"

	db, err := postgres.NewDB(cfg)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}
	defer postgres.Close(db)

	// Setup test Redis
	redisCfg := redis.DefaultConfig()
	redisCfg.Host = "localhost"
	redisCfg.Port = 6379

	redisClient, err := redis.NewClient(redisCfg)
	if err != nil {
		t.Skipf("Cannot connect to test Redis: %v", err)
	}
	defer redisClient.Close()

	handler := NewHealthHandler(db, redisClient, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var response ReadinessResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response.Status)
	assert.NotEmpty(t, response.Timestamp)

	// Check database health
	assert.Contains(t, response.Checks, "database")
	assert.Equal(t, "up", response.Checks["database"].Status)
	assert.Greater(t, response.Checks["database"].LatencyMs, float64(0))

	// Check Redis health
	assert.Contains(t, response.Checks, "redis")
	assert.Equal(t, "up", response.Checks["redis"].Status)
	assert.Greater(t, response.Checks["redis"].LatencyMs, float64(0))
}

func TestHealthHandler_Readiness_DatabaseDown(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()

	// Create a closed database connection to simulate failure
	cfg := postgres.DefaultConfig()
	cfg.Host = "invalid-host"
	cfg.Port = 9999 // Non-existent port

	db := &sqlx.DB{} // Empty/invalid DB

	// Setup valid Redis
	redisCfg := redis.DefaultConfig()
	redisClient, err := redis.NewClient(redisCfg)
	if err != nil {
		// If Redis is not available, create handler without it
		// This test focuses on database failure
		handler := NewHealthHandler(db, nil, logger)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		// Act
		handler.Readiness(rec, req)

		// Assert - should fail because DB is down
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var response ReadinessResponse
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "not_ready", response.Status)
		return
	}
	defer redisClient.Close()

	handler := NewHealthHandler(db, redisClient, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var response ReadinessResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "not_ready", response.Status)
	assert.NotEmpty(t, response.Timestamp)

	// Check database health - should be down
	assert.Contains(t, response.Checks, "database")
	assert.Equal(t, "down", response.Checks["database"].Status)
	assert.NotEmpty(t, response.Checks["database"].Error)
}

func TestHealthHandler_Readiness_ResponseStructure(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()
	handler := NewHealthHandler(nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.Readiness(rec, req)

	// Assert - response should have correct structure
	var response ReadinessResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Status field should be present
	assert.NotEmpty(t, response.Status)
	assert.Contains(t, []string{"ready", "not_ready"}, response.Status)

	// Timestamp should be present and in RFC3339 format
	assert.NotEmpty(t, response.Timestamp)

	// Checks should be a map
	assert.NotNil(t, response.Checks)
	assert.IsType(t, map[string]CheckDetails{}, response.Checks)

	// Should have database and redis checks
	assert.Contains(t, response.Checks, "database")
	assert.Contains(t, response.Checks, "redis")

	// Each check should have a status
	for name, check := range response.Checks {
		assert.NotEmpty(t, check.Status, "Check %s should have status", name)
		assert.Contains(t, []string{"up", "down"}, check.Status)
	}
}

func TestHealthHandler_Liveness_ResponseStructure(t *testing.T) {
	// Arrange
	logger := zerolog.Nop()
	handler := NewHealthHandler(nil, nil, logger)

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

	// Verify content type
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
