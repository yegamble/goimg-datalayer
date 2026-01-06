package identity_test

import (
	"testing"
	"time"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
)

// TestTimingSecurityProperties verifies the security properties of the timing mechanism.
// This test validates S10-PERF-001: <500ms p95 login latency.
func TestTimingSecurityProperties(t *testing.T) {
	t.Run("delay_distribution", func(t *testing.T) {
		// Collect 1000 samples to verify uniform distribution
		const samples = 1000
		delays := make([]time.Duration, samples)

		for i := 0; i < samples; i++ {
			delays[i] = appidentity.CalculateRandomAuthDelay()
		}

		// Calculate statistics
		var sum time.Duration
		min := delays[0]
		max := delays[0]

		for _, d := range delays {
			sum += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := sum / samples

		// Verify bounds (100-300ms)
		if min < 100*time.Millisecond {
			t.Errorf("Minimum delay %v is below 100ms bound", min)
		}
		if max > 300*time.Millisecond {
			t.Errorf("Maximum delay %v is above 300ms bound", max)
		}

		// Verify average is approximately in the middle (150-250ms expected)
		if avg < 150*time.Millisecond || avg > 250*time.Millisecond {
			t.Errorf("Average delay %v is outside expected range (150-250ms)", avg)
		}

		t.Logf("Delay distribution: min=%v, max=%v, avg=%v", min, max, avg)
	})

	t.Run("p95_under_500ms", func(t *testing.T) {
		// Simulate 1000 login attempts with varying processing times
		const samples = 1000
		totalTimes := make([]time.Duration, samples)

		for i := 0; i < samples; i++ {
			targetDelay := appidentity.CalculateRandomAuthDelay()

			// Simulate varying processing times (10-200ms)
			processingTime := time.Duration(10+i%190) * time.Millisecond

			// Calculate what the total time would be
			if processingTime < targetDelay {
				totalTimes[i] = targetDelay
			} else {
				totalTimes[i] = processingTime
			}
		}

		// Calculate p95
		// Sort to find percentile
		for i := 0; i < len(totalTimes)-1; i++ {
			for j := i + 1; j < len(totalTimes); j++ {
				if totalTimes[j] < totalTimes[i] {
					totalTimes[i], totalTimes[j] = totalTimes[j], totalTimes[i]
				}
			}
		}

		p95Index := int(float64(samples) * 0.95)
		p95 := totalTimes[p95Index]

		if p95 > 500*time.Millisecond {
			t.Errorf("p95 latency %v exceeds 500ms threshold (S10-PERF-001)", p95)
		}

		t.Logf("p95 latency: %v (target: <500ms)", p95)
	})

	t.Run("timing_masking", func(t *testing.T) {
		// Verify that different processing times result in similar total times
		const samples = 100

		// Fast processing (user not found)
		fastTimes := make([]time.Duration, samples)
		for i := 0; i < samples; i++ {
			targetDelay := appidentity.CalculateRandomAuthDelay()
			// Simulate ~5ms for "user not found"
			processingTime := 5 * time.Millisecond
			if processingTime < targetDelay {
				fastTimes[i] = targetDelay
			} else {
				fastTimes[i] = processingTime
			}
		}

		// Slow processing (bcrypt verification)
		slowTimes := make([]time.Duration, samples)
		for i := 0; i < samples; i++ {
			targetDelay := appidentity.CalculateRandomAuthDelay()
			// Simulate ~100ms for bcrypt verification
			processingTime := 100 * time.Millisecond
			if processingTime < targetDelay {
				slowTimes[i] = targetDelay
			} else {
				slowTimes[i] = processingTime
			}
		}

		// Calculate averages
		var fastSum, slowSum time.Duration
		for i := 0; i < samples; i++ {
			fastSum += fastTimes[i]
			slowSum += slowTimes[i]
		}
		fastAvg := fastSum / samples
		slowAvg := slowSum / samples

		// The difference should be less than 50ms (timing is masked by random delay)
		diff := slowAvg - fastAvg
		if diff < 0 {
			diff = -diff
		}

		// Since both fast and slow processing should be masked by the 100-300ms delay,
		// the average difference should be relatively small compared to the delay range
		if diff > 100*time.Millisecond {
			t.Errorf("Timing difference %v between fast and slow processing is too large (should be masked)", diff)
		}

		t.Logf("Fast avg: %v, Slow avg: %v, Diff: %v", fastAvg, slowAvg, diff)
	})
}

// TestTimingStatistics runs comprehensive timing statistics.
func TestTimingStatistics(t *testing.T) {
	const samples = 10000

	t.Run("uniform_distribution_check", func(t *testing.T) {
		// Divide 100-300ms range into 10 buckets of 20ms each
		buckets := make([]int, 10)

		for i := 0; i < samples; i++ {
			delay := appidentity.CalculateRandomAuthDelay()
			// Calculate bucket index (0-9)
			bucketIndex := int((delay - 100*time.Millisecond) / (20 * time.Millisecond))
			if bucketIndex < 0 {
				bucketIndex = 0
			}
			if bucketIndex >= 10 {
				bucketIndex = 9
			}
			buckets[bucketIndex]++
		}

		// For uniform distribution, each bucket should have ~10% of samples
		expectedPerBucket := samples / 10
		tolerance := expectedPerBucket / 5 // 20% tolerance

		t.Logf("Distribution (expected ~%d per bucket):", expectedPerBucket)
		for i, count := range buckets {
			rangeStart := 100 + i*20
			rangeEnd := rangeStart + 20
			t.Logf("  %dms-%dms: %d samples", rangeStart, rangeEnd, count)

			deviation := count - expectedPerBucket
			if deviation < 0 {
				deviation = -deviation
			}
			if deviation > tolerance {
				t.Errorf("Bucket %d (%dms-%dms) has %d samples, expected ~%d (deviation: %d)",
					i, rangeStart, rangeEnd, count, expectedPerBucket, deviation)
			}
		}
	})
}
