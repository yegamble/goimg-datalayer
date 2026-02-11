package handlers_test

import (
	"sync"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// getTestMetricsCollector returns a singleton MetricsCollector for testing
// to avoid "duplicate metrics collector registration" panic.
// This is shared across all tests in the handlers_test package.
var (
	testMetricsCollector *middleware.MetricsCollector
	metricsOnce          sync.Once
)

func getTestMetricsCollector() *middleware.MetricsCollector {
	metricsOnce.Do(func() {
		testMetricsCollector = middleware.NewMetricsCollector()
	})
	return testMetricsCollector
}
