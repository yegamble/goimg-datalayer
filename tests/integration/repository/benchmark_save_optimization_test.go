package repository_test

import (
	"context"
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func BenchmarkSave_Insert_Optimization(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, b)
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create new image for each iteration to test INSERT
		img := createTestImage(ownerID, "Benchmark Image")
		b.StartTimer()

		err := repo.Save(ctx, img)
		if err != nil {
			b.Fatalf("failed to save image: %v", err)
		}
	}
}

func BenchmarkSave_Update_Optimization(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	pgContainer, err := containers.NewPostgresContainer(ctx, b)
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()
	img := createTestImage(ownerID, "Benchmark Image")

	// Save once to ensure it exists
	if err := repo.Save(ctx, img); err != nil {
		b.Fatalf("failed to save initial image: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := repo.Save(ctx, img)
		if err != nil {
			b.Fatalf("failed to save image: %v", err)
		}
	}
}
