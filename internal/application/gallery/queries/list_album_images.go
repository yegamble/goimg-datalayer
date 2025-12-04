package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListAlbumImagesQuery represents a query to list images in an album.
type ListAlbumImagesQuery struct {
	AlbumID          string
	RequestingUserID string // Optional: empty string if not authenticated
	Page             int    // Page number (1-indexed)
	PerPage          int    // Items per page
}

// ListAlbumImagesResult represents the paginated image list result.
type ListAlbumImagesResult struct {
	Images     []*ImageDTO `json:"images"`
	TotalCount int64       `json:"total_count"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}

// ListAlbumImagesHandler processes list album images queries.
// It retrieves images in an album with pagination and respects visibility.
type ListAlbumImagesHandler struct {
	albums      gallery.AlbumRepository
	albumImages gallery.AlbumImageRepository
}

// NewListAlbumImagesHandler creates a new ListAlbumImagesHandler.
func NewListAlbumImagesHandler(
	albums gallery.AlbumRepository,
	albumImages gallery.AlbumImageRepository,
) *ListAlbumImagesHandler {
	return &ListAlbumImagesHandler{
		albums:      albums,
		albumImages: albumImages,
	}
}

// Handle executes the list album images query.
//
// Process flow:
//  1. Parse album ID and validate pagination
//  2. Retrieve album and check visibility permissions
//  3. Retrieve images in album with pagination
//  4. Convert to DTOs and return with pagination metadata
//
// Returns:
//   - ListAlbumImagesResult with images and pagination info
//   - ErrAlbumNotFound if album doesn't exist
//   - Authorization error if requesting user cannot view the album
func (h *ListAlbumImagesHandler) Handle(ctx context.Context, q ListAlbumImagesQuery) (*ListAlbumImagesResult, error) {
	// 1. Parse album ID
	albumID, err := gallery.ParseAlbumID(q.AlbumID)
	if err != nil {
		return nil, fmt.Errorf("invalid album id: %w", err)
	}

	// 2. Validate and create pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		// Use defaults if invalid
		pagination = shared.DefaultPagination()
	}

	// 3. Retrieve album to check visibility
	album, err := h.albums.FindByID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("find album: %w", err)
	}

	// 4. Parse requesting user ID (may be empty for anonymous users)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// 5. Check if user can view this album
	if !canViewAlbum(album, requestingUserID) {
		return nil, fmt.Errorf("unauthorized: cannot view this album")
	}

	// 6. Retrieve images in album with pagination
	images, total, err := h.albumImages.FindImagesInAlbum(ctx, albumID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find images in album: %w", err)
	}

	// 7. Convert to DTOs
	imageDTOs := make([]*ImageDTO, 0, len(images))
	for _, image := range images {
		dto := ImageToDTO(image)
		imageDTOs = append(imageDTOs, dto)
	}

	// 8. Calculate pagination metadata
	pagination = pagination.WithTotal(total)

	return &ListAlbumImagesResult{
		Images:     imageDTOs,
		TotalCount: total,
		Page:       pagination.Page(),
		PerPage:    pagination.PerPage(),
		TotalPages: pagination.TotalPages(),
	}, nil
}
