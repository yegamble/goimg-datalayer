package identity

// AuthMetricsRecorder defines the interface for recording authentication-related metrics.
// This interface allows the application layer to record metrics without depending on
// the specific Prometheus implementation in the HTTP layer (following DDD layering).
//
// Implementations:
//   - middleware.MetricsCollector (production)
//   - NoOpAuthMetricsRecorder (testing/disabled)
type AuthMetricsRecorder interface {
	// RecordLoginDelay records the random delay applied to a login attempt.
	// delaySeconds is the delay duration in seconds (e.g., 0.15 for 150ms).
	// Sprint 10: Timing attack mitigation metric
	RecordLoginDelay(delaySeconds float64)
}

// NoOpAuthMetricsRecorder is a no-op implementation of AuthMetricsRecorder.
// Used when metrics recording is disabled or in tests.
type NoOpAuthMetricsRecorder struct{}

// RecordLoginDelay is a no-op.
func (n *NoOpAuthMetricsRecorder) RecordLoginDelay(_ float64) {}

// Ensure NoOpAuthMetricsRecorder implements AuthMetricsRecorder.
var _ AuthMetricsRecorder = (*NoOpAuthMetricsRecorder)(nil)
