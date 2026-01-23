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
	RequestingUserID string // Optional: ID of the user making the request
	OwnerUserID      string // Optional: filter by owner (empty = all public)
	Visibility       string // Optional: filter by visibility (empty = all)
	Page             int    // Page number (1-indexed)
	PerPage          int    // Items per page
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
//  2. If OwnerUserID is provided, list albums by owner with authorization logic
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

		// Authorization logic:
		// - If requesting user is the owner, they can see all visibilities (unless filtered).
		// - If requesting user is NOT the owner, they can only see Public albums.
		var visibilityFilter *gallery.Visibility

		isOwner := false
		if q.RequestingUserID != "" {
			reqUserID, err := identity.ParseUserID(q.RequestingUserID)
			if err == nil && reqUserID.Equals(ownerID) {
				isOwner = true
			}
		}

		if isOwner {
			// Owner can see everything, filter only if requested
			if q.Visibility != "" {
				v, err := gallery.ParseVisibility(q.Visibility)
				if err != nil {
					return nil, fmt.Errorf("invalid visibility: %w", err)
				}
				visibilityFilter = &v
			}
		} else {
			// Non-owners can strictly only see Public albums

			// If they explicitly requested something other than Public, return empty
			if q.Visibility != "" {
				v, err := gallery.ParseVisibility(q.Visibility)
				if err != nil {
					return nil, fmt.Errorf("invalid visibility: %w", err)
				}
				if v != gallery.VisibilityPublic {
					return &ListAlbumsResult{
						Albums:     []AlbumDTO{},
						TotalCount: 0,
						Page:       pagination.Page(),
						PerPage:    pagination.PerPage(),
						TotalPages: 0,
					}, nil
				}
			}

			v := gallery.VisibilityPublic
			visibilityFilter = &v
		}

		albums, total, err = h.albums.FindByOwner(ctx, ownerID, pagination, visibilityFilter)
		if err != nil {
			return nil, fmt.Errorf("find albums by owner: %w", err)
		}
	} else {
		// List public albums with pagination
		// If visibility filter is provided and is NOT public, return empty
		if q.Visibility != "" {
			v, err := gallery.ParseVisibility(q.Visibility)
			if err != nil {
				return nil, fmt.Errorf("invalid visibility: %w", err)
			}
			if v != gallery.VisibilityPublic {
				return &ListAlbumsResult{
					Albums:     []AlbumDTO{},
					TotalCount: 0,
					Page:       pagination.Page(),
					PerPage:    pagination.PerPage(),
					TotalPages: 0,
				}, nil
			}
		}

		albums, total, err = h.albums.FindPublic(ctx, pagination)
		if err != nil {
			return nil, fmt.Errorf("find public albums: %w", err)
		}
	}

	// 3. Convert to DTOs
	albumDTOs := make([]AlbumDTO, 0, len(albums))
	for _, album := range albums {
		albumDTOs = append(albumDTOs, albumToDTO(album))
	}

	// 4. Calculate pagination metadata
	pagination = pagination.WithTotal(total)

	return &ListAlbumsResult{
		Albums:     albumDTOs,
		TotalCount: total,
		Page:       pagination.Page(),
		PerPage:    pagination.PerPage(),
		TotalPages: pagination.TotalPages(),
	}, nil
}
