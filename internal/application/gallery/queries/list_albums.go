package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListAlbumsQuery represents a query to list albums with filters and pagination.
type ListAlbumsQuery struct {
	OwnerUserID string // Optional: filter by owner (empty = all public)
	Visibility  string // Optional: filter by visibility (empty = all)
	Page        int    // Page number (1-indexed)
	PerPage     int    // Items per page
}

// ListAlbumsResult represents the paginated album list result.
type ListAlbumsResult struct {
	Albums     []AlbumDTO `json:"albums"`
	TotalCount int64      `json:"total_count"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
	TotalPages int        `json:"total_pages"`
}

// ListAlbumsHandler processes list albums queries.
// It retrieves albums based on filters and pagination.
type ListAlbumsHandler struct {
	albums gallery.AlbumRepository
}

// NewListAlbumsHandler creates a new ListAlbumsHandler.
func NewListAlbumsHandler(albums gallery.AlbumRepository) *ListAlbumsHandler {
	return &ListAlbumsHandler{
		albums: albums,
	}
}

// Handle executes the list albums query.
//
// Process flow:
//  1. Validate pagination parameters
//  2. If OwnerUserID is provided, list albums by owner
//  3. Otherwise, list public albums
//  4. Convert to DTOs and return with pagination metadata
//
// Returns:
//   - ListAlbumsResult with albums and pagination info
//   - Validation errors if pagination is invalid
func (h *ListAlbumsHandler) Handle(ctx context.Context, q ListAlbumsQuery) (*ListAlbumsResult, error) {
	// 1. Validate and create pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		// Use defaults if invalid
		pagination = shared.DefaultPagination()
	}

	var albums []*gallery.Album
	var total int64

	// 2. Determine query type: by owner or public
	if q.OwnerUserID != "" {
		// List albums by owner
		ownerID, err := identity.ParseUserID(q.OwnerUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid owner user id: %w", err)
		}

		albums, err = h.albums.FindByOwner(ctx, ownerID)
		if err != nil {
			return nil, fmt.Errorf("find albums by owner: %w", err)
		}
		total = int64(len(albums))

		// Apply pagination manually for FindByOwner (doesn't have built-in pagination)
		albums = paginateAlbums(albums, pagination)
	} else {
		// List public albums with pagination
		albums, total, err = h.albums.FindPublic(ctx, pagination)
		if err != nil {
			return nil, fmt.Errorf("find public albums: %w", err)
		}
	}

	// 3. Filter by visibility if specified
	if q.Visibility != "" {
		visibility, err := gallery.ParseVisibility(q.Visibility)
		if err != nil {
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}
		albums = filterAlbumsByVisibility(albums, visibility)
		total = int64(len(albums))
	}

	// 4. Convert to DTOs
	albumDTOs := make([]AlbumDTO, 0, len(albums))
	for _, album := range albums {
		albumDTOs = append(albumDTOs, albumToDTO(album))
	}

	// 5. Calculate pagination metadata
	pagination = pagination.WithTotal(total)

	return &ListAlbumsResult{
		Albums:     albumDTOs,
		TotalCount: total,
		Page:       pagination.Page(),
		PerPage:    pagination.PerPage(),
		TotalPages: pagination.TotalPages(),
	}, nil
}

// paginateAlbums applies pagination to an in-memory album slice.
func paginateAlbums(albums []*gallery.Album, pagination shared.Pagination) []*gallery.Album {
	offset := pagination.Offset()
	limit := pagination.Limit()

	if offset >= len(albums) {
		return []*gallery.Album{}
	}

	end := offset + limit
	if end > len(albums) {
		end = len(albums)
	}

	return albums[offset:end]
}

// filterAlbumsByVisibility filters albums by visibility.
func filterAlbumsByVisibility(albums []*gallery.Album, visibility gallery.Visibility) []*gallery.Album {
	filtered := make([]*gallery.Album, 0, len(albums))
	for _, album := range albums {
		if album.Visibility() == visibility {
			filtered = append(filtered, album)
		}
	}
	return filtered
}
