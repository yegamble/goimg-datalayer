package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// StubLikeRepository for benchmarking.
type StubLikeRepository struct {
	count int
}

func (r *StubLikeRepository) GetLikedImageIDs(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]uuid.UUID, error) {
	// Simulate DB latency
	time.Sleep(1 * time.Millisecond)

	limit := pagination.Limit()
	if limit > r.count {
		limit = r.count
	}

	ids := make([]uuid.UUID, 0, limit)
	for i := 0; i < limit; i++ {
		ids = append(ids, uuid.New())
	}
	return ids, nil
}

func (r *StubLikeRepository) CountLikedImagesByUser(ctx context.Context, userID identity.UserID) (int64, error) {
	return int64(r.count), nil
}

// Other methods are no-ops
func (r *StubLikeRepository) Like(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error { return nil }
func (r *StubLikeRepository) Unlike(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) error { return nil }
func (r *StubLikeRepository) HasLiked(ctx context.Context, userID identity.UserID, imageID gallery.ImageID) (bool, error) { return false, nil }
func (r *StubLikeRepository) GetLikeCount(ctx context.Context, imageID gallery.ImageID) (int64, error) { return 0, nil }

func BenchmarkGetUserLikedImagesHandler(b *testing.B) {
	// Simulate 20 liked images per page
	likesRepo := &StubLikeRepository{count: 20}
	imagesRepo := &StubImageRepository{}

	handler := queries.NewGetUserLikedImagesHandler(likesRepo, imagesRepo)
	ctx := context.Background()
	query := queries.GetUserLikedImagesQuery{
		UserID:  identity.NewUserID().String(),
		Page:    1,
		PerPage: 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(ctx, query)
		if err != nil {
			b.Fatal(err)
		}
	}
}
