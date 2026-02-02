package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// StubImageRepository for benchmarking.
type StubImageRepository struct{}

func (r *StubImageRepository) FindByID(ctx context.Context, id gallery.ImageID) (*gallery.Image, error) {
	// Simulate DB latency
	time.Sleep(1 * time.Millisecond)

	// Return a dummy image
	return gallery.ReconstructImage(
		id,
		identity.NewUserID(),
		createMetadata(),
		gallery.VisibilityPublic,
		gallery.StatusActive,
		gallery.ScanStatusClean,
		nil, nil, nil, 0, 0, 0, time.Now(), time.Now(),
	), nil
}

func (r *StubImageRepository) FindByIDs(ctx context.Context, ids []gallery.ImageID) ([]*gallery.Image, error) {
	// Simulate DB latency (single round trip)
	time.Sleep(1 * time.Millisecond)

	images := make([]*gallery.Image, len(ids))
	for i, id := range ids {
		images[i] = gallery.ReconstructImage(
			id,
			identity.NewUserID(),
			createMetadata(),
			gallery.VisibilityPublic,
			gallery.StatusActive,
			gallery.ScanStatusClean,
			nil, nil, nil, 0, 0, 0, time.Now(), time.Now(),
		)
	}
	return images, nil
}

// Other methods are no-ops
func (r *StubImageRepository) NextID() gallery.ImageID { return gallery.NewImageID() }

func (r *StubImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}

func (r *StubImageRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}

func (r *StubImageRepository) FindByTag(ctx context.Context, tag gallery.Tag, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}

func (r *StubImageRepository) FindByStatus(ctx context.Context, status gallery.ImageStatus, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}

func (r *StubImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (r *StubImageRepository) Save(ctx context.Context, image *gallery.Image) error { return nil }
func (r *StubImageRepository) Delete(ctx context.Context, id gallery.ImageID) error { return nil }
func (r *StubImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	return true, nil
}

// StubFeaturedPickRepository for benchmarking.
type StubFeaturedPickRepository struct {
	count int
}

func (r *StubFeaturedPickRepository) ListActive(ctx context.Context, limit int) ([]*gallery.FeaturedPick, error) {
	picks := make([]*gallery.FeaturedPick, r.count)
	for i := 0; i < r.count; i++ {
		pick, _ := gallery.NewFeaturedPick(
			gallery.NewImageID(),
			identity.NewUserID(),
			"Reason",
			i,
		)
		picks[i] = pick
	}
	return picks, nil
}

// Other methods are no-ops
func (r *StubFeaturedPickRepository) FindByID(ctx context.Context, id gallery.FeaturedPickID) (*gallery.FeaturedPick, error) {
	return nil, nil
}

func (r *StubFeaturedPickRepository) FindByImageID(ctx context.Context, imageID gallery.ImageID) (*gallery.FeaturedPick, error) {
	return nil, nil
}

func (r *StubFeaturedPickRepository) ListAll(ctx context.Context, includeExpired bool, offset, limit int) ([]*gallery.FeaturedPick, int, error) {
	return nil, 0, nil
}

func (r *StubFeaturedPickRepository) Save(ctx context.Context, pick *gallery.FeaturedPick) error {
	return nil
}

func (r *StubFeaturedPickRepository) Delete(ctx context.Context, id gallery.FeaturedPickID) error {
	return nil
}

func (r *StubFeaturedPickRepository) ExistsByImageID(ctx context.Context, imageID gallery.ImageID) (bool, error) {
	return true, nil
}

func createMetadata() gallery.ImageMetadata {
	m, _ := gallery.NewImageMetadata("Title", "Desc", "orig.jpg", "image/jpeg", 100, 100, 1024, "key", "local")
	return m
}

func BenchmarkListFeaturedImagesHandler(b *testing.B) {
	// Simulate 10 featured images
	picksRepo := &StubFeaturedPickRepository{count: 10}
	imagesRepo := &StubImageRepository{}
	logger := zerolog.Nop()

	handler := queries.NewListFeaturedImagesHandler(picksRepo, imagesRepo, logger)
	ctx := context.Background()
	query := queries.ListFeaturedImagesQuery{Limit: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(ctx, query)
		if err != nil {
			b.Fatal(err)
		}
	}
}
