package security

// HIBPMetricsRecorder defines the interface for recording HIBP-related metrics.
// This interface allows the security infrastructure to record metrics without
// depending on the specific Prometheus implementation (following DDD layering).
//
// Implementations:
//   - middleware.MetricsCollector (production)
//   - NoOpHIBPMetricsRecorder (testing/disabled)
type HIBPMetricsRecorder interface {
	// RecordHIBPCheck records a Have I Been Pwned password check.
	// result should be one of: "clean", "compromised", "error", "cache_hit", "skipped"
	// Sprint 10: HIBP password check metric
	RecordHIBPCheck(result string)

	// RecordHIBPCheckDuration records the duration of a HIBP password check.
	// durationSeconds is the check duration in seconds.
	// cacheHit indicates if the result came from cache.
	// Sprint 10: HIBP password check metric
	RecordHIBPCheckDuration(durationSeconds float64, cacheHit bool)
}

// NoOpHIBPMetricsRecorder is a no-op implementation of HIBPMetricsRecorder.
// Used when metrics recording is disabled or in tests.
type NoOpHIBPMetricsRecorder struct{}

// RecordHIBPCheck is a no-op.
func (n *NoOpHIBPMetricsRecorder) RecordHIBPCheck(_ string) {}

// RecordHIBPCheckDuration is a no-op.
func (n *NoOpHIBPMetricsRecorder) RecordHIBPCheckDuration(_ float64, _ bool) {}

// Ensure NoOpHIBPMetricsRecorder implements HIBPMetricsRecorder.
var _ HIBPMetricsRecorder = (*NoOpHIBPMetricsRecorder)(nil)
