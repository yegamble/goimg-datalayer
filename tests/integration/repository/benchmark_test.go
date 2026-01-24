package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func BenchmarkImageRepository_Save_WithTags(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping integration benchmark")
	}

	ctx := context.Background()
	// Use a shared container for the benchmark to avoid startup overhead per iteration
	// Ideally we would reuse the container but for simplicity I will spin it up once per benchmark run.
	pgContainer, err := containers.NewPostgresContainer(ctx, b)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	repo := postgres.NewImageRepository(pgContainer.DB)
	ownerID := createTestUser()

	// Create an image with many tags to exacerbate the N+1 problem
	numTags := 50
	image := createTestImage(ownerID, "Benchmark Image")
	for i := 0; i < numTags; i++ {
		tag := gallery.MustNewTag(fmt.Sprintf("tag-%d", i))
		_ = image.AddTag(tag)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We save the same image repeatedly.
		// This exercises the update path + saveTagsInTx (which does delete + insert N tags).
		err = repo.Save(ctx, image)
		if err != nil {
			b.Fatal(err)
		}
	}
}
