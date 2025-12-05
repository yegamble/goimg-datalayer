package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
)

// HealthHandler handles health check endpoints for monitoring and orchestration.
// It provides liveness and readiness probes for Kubernetes/Docker health checks.
type HealthHandler struct {
	db     *sqlx.DB
	redis  *redis.Client
	logger zerolog.Logger
}

// NewHealthHandler creates a new HealthHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
//
// Parameters:
//   - db: PostgreSQL database connection pool
//   - redis: Redis client for cache/session store
//   - logger: Structured logger for health check events
func NewHealthHandler(
	db *sqlx.DB,
	redis *redis.Client,
	logger zerolog.Logger,
) *HealthHandler {
	return &HealthHandler{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

// LivenessResponse represents the response from the liveness endpoint.
type LivenessResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ReadinessResponse represents the response from the readiness endpoint.
type ReadinessResponse struct {
	Status    string                  `json:"status"`
	Timestamp string                  `json:"timestamp"`
	Checks    map[string]CheckDetails `json:"checks"`
}

// CheckDetails provides detailed information about a specific health check.
type CheckDetails struct {
	Status    string  `json:"status"` // "up" or "down"
	LatencyMs float64 `json:"latency_ms,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// Liveness handles GET /health
// Returns 200 OK if the server is running. This is a simple liveness probe
// that indicates the HTTP server is responsive.
//
// This endpoint should be used for Kubernetes livenessProbe or Docker HEALTHCHECK
// to determine if the container should be restarted.
//
// Response: 200 OK with LivenessResponse JSON
// {
//   "status": "ok",
//   "timestamp": "2024-12-05T12:00:00Z"
// }
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	response := LivenessResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to encode liveness response")
	}
}

// Readiness handles GET /health/ready
// Checks if the application is ready to accept traffic by verifying all
// critical dependencies (database, Redis) are healthy.
//
// This endpoint should be used for Kubernetes readinessProbe to determine
// if the container should receive traffic.
//
// Response:
//   - 200 OK if all dependencies are healthy
//   - 503 Service Unavailable if any dependency is unhealthy
//
// Example healthy response:
// {
//   "status": "ready",
//   "timestamp": "2024-12-05T12:00:00Z",
//   "checks": {
//     "database": {"status": "up", "latency_ms": 5.2},
//     "redis": {"status": "up", "latency_ms": 1.8}
//   }
// }
//
// Example unhealthy response (503):
// {
//   "status": "not_ready",
//   "timestamp": "2024-12-05T12:00:00Z",
//   "checks": {
//     "database": {"status": "down", "error": "connection refused"},
//     "redis": {"status": "up", "latency_ms": 2.1}
//   }
// }
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	checks := make(map[string]CheckDetails)
	allHealthy := true

	// Check database connectivity
	dbStatus, dbLatency := h.checkDatabase(ctx)
	checks["database"] = dbStatus
	if dbStatus.Status == "down" {
		allHealthy = false
	}

	// Check Redis connectivity
	redisStatus, redisLatency := h.checkRedis(ctx)
	checks["redis"] = redisStatus
	if redisStatus.Status == "down" {
		allHealthy = false
	}

	// Determine overall status
	status := "ready"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	// Log readiness check results
	if !allHealthy {
		h.logger.Warn().
			Str("status", status).
			Float64("database_latency_ms", dbLatency).
			Float64("redis_latency_ms", redisLatency).
			Bool("database_healthy", dbStatus.Status == "up").
			Bool("redis_healthy", redisStatus.Status == "up").
			Msg("readiness check failed")
	} else {
		h.logger.Debug().
			Float64("database_latency_ms", dbLatency).
			Float64("redis_latency_ms", redisLatency).
			Msg("readiness check succeeded")
	}

	if err := EncodeJSON(w, httpStatus, response); err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to encode readiness response")
	}
}

// checkDatabase verifies PostgreSQL database connectivity and measures latency.
// Returns CheckDetails with status and latency, plus latency as a separate value for logging.
func (h *HealthHandler) checkDatabase(ctx context.Context) (CheckDetails, float64) {
	// Create a context with 5 second timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := postgres.HealthCheck(checkCtx, h.db)
	latency := time.Since(start).Seconds() * 1000 // Convert to milliseconds

	if err != nil {
		h.logger.Warn().
			Err(err).
			Float64("latency_ms", latency).
			Msg("database health check failed")

		return CheckDetails{
			Status:    "down",
			LatencyMs: latency,
			Error:     err.Error(),
		}, latency
	}

	return CheckDetails{
		Status:    "up",
		LatencyMs: latency,
	}, latency
}

// checkRedis verifies Redis connectivity and measures latency.
// Returns CheckDetails with status and latency, plus latency as a separate value for logging.
func (h *HealthHandler) checkRedis(ctx context.Context) (CheckDetails, float64) {
	// Create a context with 5 second timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := h.redis.HealthCheck(checkCtx)
	latency := time.Since(start).Seconds() * 1000 // Convert to milliseconds

	if err != nil {
		h.logger.Warn().
			Err(err).
			Float64("latency_ms", latency).
			Msg("redis health check failed")

		return CheckDetails{
			Status:    "down",
			LatencyMs: latency,
			Error:     err.Error(),
		}, latency
	}

	return CheckDetails{
		Status:    "up",
		LatencyMs: latency,
	}, latency
}
