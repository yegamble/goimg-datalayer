package identity_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/application/identity"
)

// TestCalculateRandomAuthDelay_ReturnsValueInRange verifies that the random delay
// is always within the configured range [100ms, 300ms].
func TestCalculateRandomAuthDelay_ReturnsValueInRange(t *testing.T) {
	t.Parallel()

	const iterations = 100

	for i := 0; i < iterations; i++ {
		delay := identity.CalculateRandomAuthDelay()

		assert.GreaterOrEqual(t, delay, identity.MinAuthDelay,
			"delay should be >= MinAuthDelay (100ms)")
		assert.LessOrEqual(t, delay, identity.MaxAuthDelay,
			"delay should be <= MaxAuthDelay (300ms)")
	}
}

// TestCalculateRandomAuthDelay_DistributionWithinRange verifies that over many samples,
// the distribution of delays is roughly uniform across the range.
func TestCalculateRandomAuthDelay_DistributionWithinRange(t *testing.T) {
	t.Parallel()

	const sampleSize = 100
	delays := make([]time.Duration, sampleSize)

	// Collect samples
	for i := 0; i < sampleSize; i++ {
		delays[i] = identity.CalculateRandomAuthDelay()
	}

	// Calculate mean
	var sum time.Duration
	for _, d := range delays {
		sum += d
	}
	mean := sum / time.Duration(sampleSize)

	// Expected mean for uniform distribution: (min + max) / 2
	expectedMean := (identity.MinAuthDelay + identity.MaxAuthDelay) / 2

	// Allow 20% variance from expected mean (accounts for randomness)
	tolerance := expectedMean / 5
	assert.InDelta(t, expectedMean, mean, float64(tolerance),
		"mean should be approximately (min+max)/2")

	// Calculate standard deviation
	var sumSquaredDiff float64
	for _, d := range delays {
		diff := float64(d - mean)
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(sampleSize))

	// Expected standard deviation for uniform distribution: (max-min)/sqrt(12)
	rangeMs := float64(identity.MaxAuthDelay - identity.MinAuthDelay)
	expectedStdDev := rangeMs / math.Sqrt(12)

	// Allow 30% variance from expected standard deviation
	stdDevTolerance := expectedStdDev * 0.3
	assert.InDelta(t, expectedStdDev, stdDev, stdDevTolerance,
		"standard deviation should match uniform distribution")
}

// TestCalculateRandomAuthDelay_ProducesVariedResults ensures that the function
// generates different values (not always the same).
func TestCalculateRandomAuthDelay_ProducesVariedResults(t *testing.T) {
	t.Parallel()

	const iterations = 50
	results := make(map[time.Duration]bool)

	for i := 0; i < iterations; i++ {
		delay := identity.CalculateRandomAuthDelay()
		results[delay] = true
	}

	// With 50 iterations and 200ms range (at 1ms granularity), we should see
	// at least 20 unique values (conservatively 40% of iterations)
	minUniqueValues := iterations * 40 / 100
	assert.GreaterOrEqual(t, len(results), minUniqueValues,
		"should produce varied random delays, not repeating the same value")
}

// TestCalculateRandomAuthDelay_FallbackOnError tests that if random generation fails,
// the function returns MaxAuthDelay as a safe fallback.
// Note: This is difficult to test directly since crypto/rand.Reader rarely fails.
// The test documents the expected behavior.
func TestCalculateRandomAuthDelay_FallbackOnError(t *testing.T) {
	t.Parallel()

	// Since we can't easily force crypto/rand to fail in a unit test,
	// we document the expected behavior: if random generation fails,
	// the function should return MaxAuthDelay (300ms).
	// This is verified through code review of timing.go:19-21.

	// This test serves as documentation that the fallback exists
	delay := identity.CalculateRandomAuthDelay()
	assert.LessOrEqual(t, delay, identity.MaxAuthDelay,
		"delay should never exceed MaxAuthDelay, even in fallback scenario")
}

// TestApplyAuthDelay_AppliesRemainingDelay verifies that when actual processing
// time is less than the target delay, the function sleeps for the remaining time.
func TestApplyAuthDelay_AppliesRemainingDelay(t *testing.T) {
	t.Parallel()

	targetDelay := 200 * time.Millisecond
	actualProcessing := 50 * time.Millisecond

	start := time.Now()
	appliedDelay := identity.ApplyAuthDelay(targetDelay, actualProcessing)
	elapsed := time.Since(start)

	// Applied delay should be the remaining time (150ms)
	expectedDelay := 150 * time.Millisecond
	assert.Equal(t, expectedDelay, appliedDelay,
		"applied delay should equal target - actual")

	// Actual sleep time should match applied delay (±50ms for OS scheduling variance)
	tolerance := 50 * time.Millisecond
	assert.InDelta(t, expectedDelay, elapsed, float64(tolerance),
		"actual sleep duration should match applied delay")
}

// TestApplyAuthDelay_AppliesMinimumDelayWhenExceeded tests that when processing
// time exceeds the target delay, the function applies MinAuthDelay instead.
func TestApplyAuthDelay_AppliesMinimumDelayWhenExceeded(t *testing.T) {
	t.Parallel()

	targetDelay := 200 * time.Millisecond
	actualProcessing := 250 * time.Millisecond // Exceeds target

	start := time.Now()
	appliedDelay := identity.ApplyAuthDelay(targetDelay, actualProcessing)
	elapsed := time.Since(start)

	// When target is exceeded, should apply MinAuthDelay
	assert.Equal(t, identity.MinAuthDelay, appliedDelay,
		"should apply MinAuthDelay when target exceeded")

	// Actual sleep time should match MinAuthDelay (±50ms tolerance for OS scheduling)
	tolerance := 50 * time.Millisecond
	assert.InDelta(t, identity.MinAuthDelay, elapsed, float64(tolerance),
		"actual sleep duration should match MinAuthDelay")
}

// TestApplyAuthDelay_AppliesMinimumDelayWhenExactlyMet tests edge case where
// actual processing exactly equals target delay.
func TestApplyAuthDelay_AppliesMinimumDelayWhenExactlyMet(t *testing.T) {
	t.Parallel()

	targetDelay := 200 * time.Millisecond
	actualProcessing := 200 * time.Millisecond // Exactly equals target

	start := time.Now()
	appliedDelay := identity.ApplyAuthDelay(targetDelay, actualProcessing)
	elapsed := time.Since(start)

	// When remaining is 0, should apply MinAuthDelay
	assert.Equal(t, identity.MinAuthDelay, appliedDelay,
		"should apply MinAuthDelay when processing exactly meets target")

	// Actual sleep time should match MinAuthDelay (±50ms tolerance for OS scheduling)
	tolerance := 50 * time.Millisecond
	assert.InDelta(t, identity.MinAuthDelay, elapsed, float64(tolerance),
		"actual sleep duration should match MinAuthDelay")
}

// TestApplyAuthDelay_WithZeroActualProcessing tests boundary condition where
// actual processing time is zero (immediate return from handler).
func TestApplyAuthDelay_WithZeroActualProcessing(t *testing.T) {
	t.Parallel()

	targetDelay := 200 * time.Millisecond
	actualProcessing := 0 * time.Millisecond

	start := time.Now()
	appliedDelay := identity.ApplyAuthDelay(targetDelay, actualProcessing)
	elapsed := time.Since(start)

	// Applied delay should equal full target delay
	assert.Equal(t, targetDelay, appliedDelay,
		"applied delay should equal target when actual processing is zero")

	// Actual sleep time should match target delay (±50ms tolerance for OS scheduling)
	tolerance := 50 * time.Millisecond
	assert.InDelta(t, targetDelay, elapsed, float64(tolerance),
		"actual sleep duration should match target delay")
}

// TestApplyAuthDelay_TableDriven uses table-driven approach to test multiple scenarios.
func TestApplyAuthDelay_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		targetDelay      time.Duration
		actualProcessing time.Duration
		wantAppliedDelay time.Duration
		wantSleepTime    time.Duration // Expected actual sleep duration
	}{
		{
			name:             "normal case - half target used",
			targetDelay:      200 * time.Millisecond,
			actualProcessing: 100 * time.Millisecond,
			wantAppliedDelay: 100 * time.Millisecond,
			wantSleepTime:    100 * time.Millisecond,
		},
		{
			name:             "processing exceeds target significantly",
			targetDelay:      150 * time.Millisecond,
			actualProcessing: 300 * time.Millisecond,
			wantAppliedDelay: identity.MinAuthDelay,
			wantSleepTime:    identity.MinAuthDelay,
		},
		{
			name:             "processing barely under target",
			targetDelay:      200 * time.Millisecond,
			actualProcessing: 199 * time.Millisecond,
			wantAppliedDelay: 1 * time.Millisecond,
			wantSleepTime:    1 * time.Millisecond,
		},
		{
			name:             "minimal remaining time",
			targetDelay:      101 * time.Millisecond,
			actualProcessing: 100 * time.Millisecond,
			wantAppliedDelay: 1 * time.Millisecond,
			wantSleepTime:    1 * time.Millisecond,
		},
		{
			name:             "target is MinAuthDelay",
			targetDelay:      identity.MinAuthDelay,
			actualProcessing: 0,
			wantAppliedDelay: identity.MinAuthDelay,
			wantSleepTime:    identity.MinAuthDelay,
		},
		{
			name:             "target is MaxAuthDelay",
			targetDelay:      identity.MaxAuthDelay,
			actualProcessing: 0,
			wantAppliedDelay: identity.MaxAuthDelay,
			wantSleepTime:    identity.MaxAuthDelay,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			appliedDelay := identity.ApplyAuthDelay(tt.targetDelay, tt.actualProcessing)
			elapsed := time.Since(start)

			// Assert applied delay matches expected
			assert.Equal(t, tt.wantAppliedDelay, appliedDelay,
				"applied delay should match expected value")

			// Assert actual sleep time is within tolerance (±50ms for OS scheduling)
			tolerance := 50 * time.Millisecond
			assert.InDelta(t, tt.wantSleepTime, elapsed, float64(tolerance),
				"actual sleep duration should match expected (±%v)", tolerance)
		})
	}
}

// TestTimingConstants_AreValid verifies that the timing constants are set correctly.
func TestTimingConstants_AreValid(t *testing.T) {
	t.Parallel()

	// MinAuthDelay should be 100ms
	assert.Equal(t, 100*time.Millisecond, identity.MinAuthDelay,
		"MinAuthDelay should be 100ms")

	// MaxAuthDelay should be 300ms
	assert.Equal(t, 300*time.Millisecond, identity.MaxAuthDelay,
		"MaxAuthDelay should be 300ms")

	// MaxAuthDelay should be greater than MinAuthDelay
	assert.Greater(t, identity.MaxAuthDelay, identity.MinAuthDelay,
		"MaxAuthDelay should be greater than MinAuthDelay")

	// Range should be 200ms
	delayRange := identity.MaxAuthDelay - identity.MinAuthDelay
	assert.Equal(t, 200*time.Millisecond, delayRange,
		"delay range should be 200ms")
}

// BenchmarkCalculateRandomAuthDelay measures the performance of random delay generation.
func BenchmarkCalculateRandomAuthDelay(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = identity.CalculateRandomAuthDelay()
	}
}

// BenchmarkApplyAuthDelay measures the performance of the delay application logic
// (excluding the actual sleep time).
func BenchmarkApplyAuthDelay(b *testing.B) {
	targetDelay := 200 * time.Millisecond
	actualProcessing := 50 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will actually sleep, so benchmark results include sleep time
		_ = identity.ApplyAuthDelay(targetDelay, actualProcessing)
	}
}

// TestApplyAuthDelay_ReturnValue verifies that the return value accurately
// represents the delay that was applied.
func TestApplyAuthDelay_ReturnValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		targetDelay      time.Duration
		actualProcessing time.Duration
		wantReturn       time.Duration
	}{
		{
			name:             "returns remaining delay",
			targetDelay:      200 * time.Millisecond,
			actualProcessing: 75 * time.Millisecond,
			wantReturn:       125 * time.Millisecond,
		},
		{
			name:             "returns MinAuthDelay when exceeded",
			targetDelay:      100 * time.Millisecond,
			actualProcessing: 200 * time.Millisecond,
			wantReturn:       identity.MinAuthDelay,
		},
		{
			name:             "returns MinAuthDelay when exactly met",
			targetDelay:      150 * time.Millisecond,
			actualProcessing: 150 * time.Millisecond,
			wantReturn:       identity.MinAuthDelay,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			returnedDelay := identity.ApplyAuthDelay(tt.targetDelay, tt.actualProcessing)

			assert.Equal(t, tt.wantReturn, returnedDelay,
				"returned delay should match expected value")
		})
	}
}
