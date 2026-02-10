package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func BenchmarkSaveWithVariants(b *testing.B) {
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

	// Create 50 variants
	variants := make([]gallery.ImageVariant, 0, 50)
	for i := 0; i < 50; i++ {
		// We use VariantOriginal repeated.
		// NewImageVariant validates the type, but ReconstructImage doesn't check uniqueness.
		v, err := gallery.NewImageVariant(
			gallery.VariantOriginal,
			fmt.Sprintf("storage/variant_%d.jpg", i),
			100+i, 100+i,
			1024*int64(i+1),
			"jpeg",
		)
		if err != nil {
			b.Fatalf("failed to create variant: %v", err)
		}
		variants = append(variants, v)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// Create a new image using ReconstructImage to bypass AddVariant checks
		imgID := gallery.NewImageID()
		metadata := createTestMetadata(fmt.Sprintf("Benchmark Image %d", i))

		img := gallery.ReconstructImage(
			imgID,
			ownerID,
			metadata,
			gallery.VisibilityPrivate,
			gallery.StatusProcessing,
			gallery.ScanStatusPending,
			variants,
			[]gallery.Tag{},
			nil,
			0, 0, 0,
			time.Now().UTC(),
			time.Now().UTC(),
		)

		b.StartTimer()

		// Save calls saveVariantsInTx which does the N+1 insert
		err := repo.Save(ctx, img)
		if err != nil {
			b.Fatalf("failed to save image: %v", err)
		}
	}
}
