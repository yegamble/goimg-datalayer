package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// HealthHandler handles health check endpoints for monitoring and orchestration.
// It provides liveness and readiness probes for Kubernetes/Docker health checks.
type HealthHandler struct {
	db      *sqlx.DB
	redis   *redis.Client
	storage storage.Storage
	clamav  clamav.Scanner
	logger  zerolog.Logger
}

// NewHealthHandler creates a new HealthHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
//
// Parameters:
//   - db: PostgreSQL database connection pool
//   - redis: Redis client for cache/session store
//   - storage: Storage provider for images (local, S3, etc.)
//   - clamav: ClamAV scanner for malware detection
//   - logger: Structured logger for health check events
func NewHealthHandler(
	db *sqlx.DB,
	redis *redis.Client,
	storage storage.Storage,
	clamav clamav.Scanner,
	logger zerolog.Logger,
) *HealthHandler {
	return &HealthHandler{
		db:      db,
		redis:   redis,
		storage: storage,
		clamav:  clamav,
		logger:  logger,
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
//
//	{
//	  "status": "ok",
//	  "timestamp": "2024-12-05T12:00:00Z"
//	}
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
// critical dependencies (database, Redis, storage, ClamAV) are healthy.
//
// This endpoint should be used for Kubernetes readinessProbe to determine
// if the container should receive traffic.
//
// Status determination:
//   - "ok": All dependencies healthy
//   - "degraded": Redis down (non-critical), other dependencies healthy
//   - "down": Any critical dependency (database, storage, ClamAV) down
//
// Response:
//   - 200 OK if status is "ok" or "degraded"
//   - 503 Service Unavailable if status is "down"
//
// Example healthy response:
//
//	{
//	  "status": "ok",
//	  "timestamp": "2024-12-05T12:00:00Z",
//	  "checks": {
//	    "database": {"status": "up", "latency_ms": 5.2},
//	    "redis": {"status": "up", "latency_ms": 1.8},
//	    "storage": {"status": "up", "latency_ms": 3.1},
//	    "clamav": {"status": "up", "latency_ms": 2.5}
//	  }
//	}
//
// Example degraded response (200):
//
//	{
//	  "status": "degraded",
//	  "timestamp": "2024-12-05T12:00:00Z",
//	  "checks": {
//	    "database": {"status": "up", "latency_ms": 5.2},
//	    "redis": {"status": "down", "error": "connection refused"},
//	    "storage": {"status": "up", "latency_ms": 3.1},
//	    "clamav": {"status": "up", "latency_ms": 2.5}
//	  }
//	}
//
// Example down response (503):
//
//	{
//	  "status": "down",
//	  "timestamp": "2024-12-05T12:00:00Z",
//	  "checks": {
//	    "database": {"status": "down", "error": "connection refused"},
//	    "redis": {"status": "up", "latency_ms": 1.8},
//	    "storage": {"status": "up", "latency_ms": 3.1},
//	    "clamav": {"status": "up", "latency_ms": 2.5}
//	  }
//	}
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	checks := make(map[string]CheckDetails)

	// Check all dependencies
	dbStatus, dbLatency := h.checkDatabase(ctx)
	checks["database"] = dbStatus

	redisStatus, redisLatency := h.checkRedis(ctx)
	checks["redis"] = redisStatus

	storageStatus, storageLatency := h.checkStorage(ctx)
	checks["storage"] = storageStatus

	clamavStatus, clamavLatency := h.checkClamAV(ctx)
	checks["clamav"] = clamavStatus

	// Determine overall status with graceful degradation
	// Critical dependencies: database, storage, ClamAV
	// Non-critical: Redis (caching/sessions)
	criticalDown := dbStatus.Status == "down" ||
		storageStatus.Status == "down" ||
		clamavStatus.Status == "down"

	redisDown := redisStatus.Status == "down"

	var status string
	var httpStatus int

	if criticalDown {
		status = "down"
		httpStatus = http.StatusServiceUnavailable
	} else if redisDown {
		status = "degraded"
		httpStatus = http.StatusOK // Still accepting traffic
	} else {
		status = "ok"
		httpStatus = http.StatusOK
	}

	response := ReadinessResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	// Log readiness check results
	logEvent := h.logger.With().
		Str("status", status).
		Float64("database_latency_ms", dbLatency).
		Float64("redis_latency_ms", redisLatency).
		Float64("storage_latency_ms", storageLatency).
		Float64("clamav_latency_ms", clamavLatency).
		Bool("database_healthy", dbStatus.Status == "up").
		Bool("redis_healthy", redisStatus.Status == "up").
		Bool("storage_healthy", storageStatus.Status == "up").
		Bool("clamav_healthy", clamavStatus.Status == "up").
		Logger()

	if status == "down" {
		logEvent.Warn().Msg("readiness check failed: service down")
	} else if status == "degraded" {
		logEvent.Warn().Msg("readiness check degraded: non-critical dependency down")
	} else {
		logEvent.Debug().Msg("readiness check succeeded")
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

	// Handle nil redis client
	if h.redis == nil {
		latency := time.Since(start).Seconds() * 1000
		h.logger.Warn().
			Float64("latency_ms", latency).
			Msg("redis client is nil")

		return CheckDetails{
			Status:    "down",
			LatencyMs: latency,
			Error:     "redis client not configured",
		}, latency
	}

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

// checkStorage verifies storage provider connectivity and measures latency.
// Returns CheckDetails with status and latency, plus latency as a separate value for logging.
func (h *HealthHandler) checkStorage(ctx context.Context) (CheckDetails, float64) {
	// Create a context with 5 second timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	// Use a known health check key that should not exist
	// We just check if the storage is accessible by testing Exists operation
	healthCheckKey := "health-check-probe"
	_, err := h.storage.Exists(checkCtx, healthCheckKey)
	latency := time.Since(start).Seconds() * 1000 // Convert to milliseconds

	if err != nil {
		h.logger.Warn().
			Err(err).
			Float64("latency_ms", latency).
			Str("provider", h.storage.Provider()).
			Msg("storage health check failed")

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

// checkClamAV verifies ClamAV daemon connectivity and measures latency.
// Returns CheckDetails with status and latency, plus latency as a separate value for logging.
func (h *HealthHandler) checkClamAV(ctx context.Context) (CheckDetails, float64) {
	// Create a context with 5 second timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := h.clamav.Ping(checkCtx)
	latency := time.Since(start).Seconds() * 1000 // Convert to milliseconds

	if err != nil {
		h.logger.Warn().
			Err(err).
			Float64("latency_ms", latency).
			Msg("clamav health check failed")

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
