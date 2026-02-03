package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

// BenchmarkConnectionPool measures the impact of MaxIdleConns on concurrent request performance.
// It compares a "Low Idle" configuration (simulating the old default) vs "High Idle" (simulating the optimization).
func BenchmarkConnectionPool(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping integration benchmark")
	}

	ctx := context.Background()

	// Spin up a shared Postgres container
	pgContainer, err := containers.NewPostgresContainer(ctx, b)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	// Define scenarios
	scenarios := []struct {
		name         string
		maxOpenConns int
		maxIdleConns int
	}{
		{name: "LowIdle", maxOpenConns: 25, maxIdleConns: 5},
		{name: "HighIdle", maxOpenConns: 25, maxIdleConns: 25},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			// Configure the pool
			pgContainer.DB.SetMaxOpenConns(sc.maxOpenConns)
			pgContainer.DB.SetMaxIdleConns(sc.maxIdleConns)
			pgContainer.DB.SetConnMaxLifetime(30 * time.Minute)
			pgContainer.DB.SetConnMaxIdleTime(10 * time.Minute)

			// We use RunParallel to simulate concurrent requests
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// Execute a simple query that hits the DB but does minimal work.
					// "SELECT 1" is standard for connectivity checks.
					var result int
					err := pgContainer.DB.GetContext(ctx, &result, "SELECT 1")
					if err != nil {
						b.Error(err)
					}
				}
			})
		})
	}
}
