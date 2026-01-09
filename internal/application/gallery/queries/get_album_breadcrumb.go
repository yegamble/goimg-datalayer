package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetAlbumBreadcrumbQuery represents a query to retrieve an album's breadcrumb path.
type GetAlbumBreadcrumbQuery struct {
	AlbumID          string
	RequestingUserID string // Optional: empty string if not authenticated
}

// BreadcrumbItemDTO represents a single item in the breadcrumb path.
type BreadcrumbItemDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GetAlbumBreadcrumbHandler processes breadcrumb queries.
// It retrieves the ancestor path from root to the specified album.
type GetAlbumBreadcrumbHandler struct {
	albums gallery.AlbumRepository
}

// NewGetAlbumBreadcrumbHandler creates a new GetAlbumBreadcrumbHandler.
func NewGetAlbumBreadcrumbHandler(albums gallery.AlbumRepository) *GetAlbumBreadcrumbHandler {
	return &GetAlbumBreadcrumbHandler{
		albums: albums,
	}
}

// Handle executes the get album breadcrumb query.
//
// Process flow:
//  1. Parse album ID
//  2. Retrieve album to verify it exists and check visibility
//  3. Retrieve ancestors from repository
//  4. Convert to BreadcrumbItemDTOs and return
//
// Returns:
//   - []BreadcrumbItemDTO on success (from root to target album)
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if requesting user cannot view the album
func (h *GetAlbumBreadcrumbHandler) Handle(ctx context.Context, q GetAlbumBreadcrumbQuery) ([]BreadcrumbItemDTO, error) {
	// 1. Parse album ID
	albumID, err := gallery.ParseAlbumID(q.AlbumID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}

	// 2. Parse requesting user ID (may be empty for anonymous users)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// 3. Retrieve ancestors (includes the target album itself)
	ancestors, err := h.albums.FindAncestors(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("find ancestors: %w", err)
	}

	// 4. Check visibility permissions on the target album (last in ancestors)
	if len(ancestors) > 0 {
		targetAlbum := ancestors[len(ancestors)-1]
		if !canViewAlbum(targetAlbum, requestingUserID) {
			return nil, fmt.Errorf("unauthorized: cannot view this album")
		}
	}

	// 5. Convert to DTOs
	breadcrumb := make([]BreadcrumbItemDTO, len(ancestors))
	for i, album := range ancestors {
		breadcrumb[i] = BreadcrumbItemDTO{
			ID:    album.ID().String(),
			Title: album.Title(),
		}
	}

	return breadcrumb, nil
}

// GetAlbumChildrenQuery represents a query to retrieve an album's direct children.
type GetAlbumChildrenQuery struct {
	AlbumID          string
	RequestingUserID string // Optional: empty string if not authenticated
}

// GetAlbumChildrenHandler processes child album queries.
type GetAlbumChildrenHandler struct {
	albums gallery.AlbumRepository
}

// NewGetAlbumChildrenHandler creates a new GetAlbumChildrenHandler.
func NewGetAlbumChildrenHandler(albums gallery.AlbumRepository) *GetAlbumChildrenHandler {
	return &GetAlbumChildrenHandler{
		albums: albums,
	}
}

// Handle executes the get album children query.
//
// Returns:
//   - []AlbumDTO on success (direct child albums)
//   - ErrAlbumNotFound if parent album doesn't exist
//   - Authorization error if requesting user cannot view the parent album
func (h *GetAlbumChildrenHandler) Handle(ctx context.Context, q GetAlbumChildrenQuery) ([]AlbumDTO, error) {
	// 1. Parse album ID
	albumID, err := gallery.ParseAlbumID(q.AlbumID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}

	// 2. Parse requesting user ID (may be empty for anonymous users)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// 3. Verify parent album exists and user can view it
	parentAlbum, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("find parent album: %w", err)
	}

	if !canViewAlbum(parentAlbum, requestingUserID) {
		return nil, fmt.Errorf("unauthorized: cannot view this album")
	}

	// 4. Retrieve children
	children, err := h.albums.FindChildren(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("find children: %w", err)
	}

	// 5. Filter children based on visibility for requesting user
	result := make([]AlbumDTO, 0, len(children))
	for _, child := range children {
		if canViewAlbum(child, requestingUserID) {
			result = append(result, albumToDTO(child))
		}
	}

	return result, nil
}
