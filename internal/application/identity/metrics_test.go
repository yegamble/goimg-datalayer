package identity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
)

// TestNoOpAuthMetricsRecorder_RecordLoginDelay verifies that NoOp implementation
// doesn't panic or error when called.
func TestNoOpAuthMetricsRecorder_RecordLoginDelay(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := &appidentity.NoOpAuthMetricsRecorder{}

	// Act - should not panic
	recorder.RecordLoginDelay(0.15)
	recorder.RecordLoginDelay(0.0)
	recorder.RecordLoginDelay(1.5)

	// Assert - just verify it implements the interface
	var _ appidentity.AuthMetricsRecorder = recorder
}

// TestNoOpAuthMetricsRecorder_ImplementsInterface verifies that NoOp implements
// the AuthMetricsRecorder interface at compile time.
func TestNoOpAuthMetricsRecorder_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Arrange & Act
	var recorder appidentity.AuthMetricsRecorder = &appidentity.NoOpAuthMetricsRecorder{}

	// Assert - verify interface implementation
	assert.NotNil(t, recorder)
	assert.Implements(t, (*appidentity.AuthMetricsRecorder)(nil), recorder)
}

// TestNoOpAuthMetricsRecorder_ThreadSafety verifies that NoOp implementation
// is safe for concurrent use.
func TestNoOpAuthMetricsRecorder_ThreadSafety(t *testing.T) {
	t.Parallel()

	// Arrange
	recorder := &appidentity.NoOpAuthMetricsRecorder{}

	// Act - call from multiple goroutines
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(delay float64) {
			recorder.RecordLoginDelay(delay)
			done <- true
		}(float64(i) * 0.01)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Assert - no panics occurred
	assert.True(t, true)
}
