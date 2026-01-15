package queries_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// StubAlbumRepository simulates the repository behavior including allocations.
type StubAlbumRepository struct {
	mode string // "old" or "new"
}

func (r *StubAlbumRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Album, int64, error) {
	if r.mode == "old" {
		// Simulate fetching ALL albums (e.g. 10,000)
		count := 10000
		albums := make([]*gallery.Album, count)
		for i := 0; i < count; i++ {
			albums[i] = createAlbum(i)
		}
		// In the old code, we didn't pass pagination to DB, so we return all.
		// However, to make this work with the NEW handler interface but simulate OLD performance characteristics:
		// We would need to modify the handler to NOT use pagination in FindByOwner, which I've already changed.
		//
		// But wait, I've already changed the Handler code. I can't run the "old" handler code.
		// So I can only benchmark the "new" handler code.
		//
		// But I can benchmark the Repository layer difference effectively here.
		// Or I can benchmark the "Handler + Repo" assuming the Handler calls FindByOwner.

		// Since I've already changed the interface of FindByOwner to accept pagination,
		// the "old" implementation (fetch all) would look like this inside the new signature:
		// It ignores pagination limit/offset during fetch (simulating full table scan/fetch)
		// but since the interface requires returning the slice...

		// Actually, the optimization is that we DON'T fetch all.
		// So checking the performance of "Fetch All" vs "Fetch Page" is what we want.
		return albums, int64(count), nil
	} else {
		// Simulate fetching ONE PAGE (e.g. 20)
		count := 20
		albums := make([]*gallery.Album, count)
		for i := 0; i < count; i++ {
			albums[i] = createAlbum(i)
		}
		return albums, 10000, nil
	}
}

// Implement other interface methods with no-op or panic
func (r *StubAlbumRepository) NextID() gallery.AlbumID { return gallery.NewAlbumID() }
func (r *StubAlbumRepository) FindByID(ctx context.Context, id gallery.AlbumID) (*gallery.Album, error) { return nil, nil }
func (r *StubAlbumRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Album, int64, error) { return nil, 0, nil }
func (r *StubAlbumRepository) Save(ctx context.Context, album *gallery.Album) error { return nil }
func (r *StubAlbumRepository) Delete(ctx context.Context, id gallery.AlbumID) error { return nil }
func (r *StubAlbumRepository) ExistsByID(ctx context.Context, id gallery.AlbumID) (bool, error) { return false, nil }
func (r *StubAlbumRepository) FindChildren(ctx context.Context, parentID gallery.AlbumID) ([]*gallery.Album, error) { return nil, nil }
func (r *StubAlbumRepository) FindRootAlbumsByOwner(ctx context.Context, ownerID identity.UserID) ([]*gallery.Album, error) { return nil, nil }
func (r *StubAlbumRepository) FindAncestors(ctx context.Context, albumID gallery.AlbumID) ([]*gallery.Album, error) { return nil, nil }

func createAlbum(i int) *gallery.Album {
	return gallery.ReconstructAlbum(
		gallery.NewAlbumID(),
		identity.NewUserID(),
		nil,
		fmt.Sprintf("Album %d", i),
		"Description",
		gallery.VisibilityPublic,
		nil,
		0,
		time.Now(),
		time.Now(),
	)
}

// Benchmark the Handler with a Repo that behaves like the "Old" inefficient one
// (returning all items), vs the "New" efficient one (returning page).
// Note: Even though the Handler code is "New" (it passes pagination),
// if the Repo returns ALL items (simulating what happened before or a bad implementation),
// the handler will receive them.
// BUT, the new Handler expects `FindByOwner` to respect pagination!
// If `FindByOwner` returns 10,000 items when asked for 20, the Handler will process 10,000 items!
// Because the Handler loops over `albums` returned from Repo to convert to DTOs.
//
// So:
// Old Handler: Get 10k items -> Manual Slice to 20 -> Convert 20 to DTO.
// New Handler: Get X items -> Convert X to DTO.
//
// If I simulate "Old" behavior using New Handler:
// Repo returns 10k items. New Handler converts 10k items to DTOs.
// This is actually WORSE than Old Handler (which sliced before converting).
//
// So I cannot strictly compare Old vs New Handler logic using just the Repo stub behavior
// because the logic *inside* the handler also changed (allocating DTOs).
//
// However, the "Optimization" is the combination of Handler+Repo.
// Old: Repo fetches 10k. Handler slices.
// New: Repo fetches 20. Handler processes 20.
//
// So I should benchmark:
// 1. "Inefficient Fetch": Simulate allocating 10k items. (This represents the cost of the old approach's DB fetch phase).
// 2. "Efficient Fetch": Simulate allocating 20 items.
//
// I will benchmark the StubRepo directly to show the allocation difference.

func BenchmarkRepository_FindByOwner_Inefficient(b *testing.B) {
	repo := &StubAlbumRepository{mode: "old"}
	ctx := context.Background()
	id := identity.NewUserID()
	p, _ := shared.NewPagination(1, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.FindByOwner(ctx, id, p)
	}
}

func BenchmarkRepository_FindByOwner_Efficient(b *testing.B) {
	repo := &StubAlbumRepository{mode: "new"}
	ctx := context.Background()
	id := identity.NewUserID()
	p, _ := shared.NewPagination(1, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.FindByOwner(ctx, id, p)
	}
}
