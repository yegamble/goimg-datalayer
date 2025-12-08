package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListImagesQuery retrieves a paginated list of images with filtering options.
// This is a read-only operation that respects visibility rules.
type ListImagesQuery struct {
	// Filter options
	OwnerID    string // Optional: Filter by owner (empty = all public)
	Visibility string // Optional: Filter by visibility (empty = all visible to requester)
	Tag        string // Optional: Filter by tag slug

	// Requesting user (for authorization)
	RequestingUserID string // Optional: ID of the user requesting

	// Pagination
	Offset int
	Limit  int

	// Sorting
	SortBy    string // Options: "created_at", "updated_at", "view_count", "like_count"
	SortOrder string // Options: "asc", "desc" (default: desc)
}

// ListImagesResult represents the paginated result of a list query.
type ListImagesResult struct {
	Images     []ImageDTO `json:"images"`
	TotalCount int64      `json:"total_count"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
}

// ListImagesHandler processes ListImagesQuery requests.
// It retrieves a paginated list of images with filtering and sorting.
type ListImagesHandler struct {
	images gallery.ImageRepository
	logger *zerolog.Logger
}

// NewListImagesHandler creates a new ListImagesHandler with the given dependencies.
func NewListImagesHandler(
	images gallery.ImageRepository,
	logger *zerolog.Logger,
) *ListImagesHandler {
	return &ListImagesHandler{
		images: images,
		logger: logger,
	}
}

// Handle executes the ListImagesQuery and returns paginated results.
//
// Process flow:
//  1. Validate and parse query parameters
//  2. Determine which repository method to use based on filters
//  3. Load images from repository with pagination
//  4. Convert to DTOs and return with pagination metadata
//
// Returns:
//   - *ListImagesResult: Paginated list of images
//   - Validation errors for invalid parameters
func (h *ListImagesHandler) Handle(ctx context.Context, q ListImagesQuery) (*ListImagesResult, error) {
	// 1. Validate and parse query parameters

	// Parse owner ID if provided
	var ownerID identity.UserID
	if q.OwnerID != "" {
		var err error
		ownerID, err = identity.ParseUserID(q.OwnerID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("owner_id", q.OwnerID).
				Msg("invalid owner id in list query")
			return nil, fmt.Errorf("invalid owner id: %w", err)
		}
	}

	// Parse requesting user ID if provided
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		var err error
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("requesting_user_id", q.RequestingUserID).
				Msg("invalid requesting user id in list query")
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// Parse visibility filter if provided
	var visibilityFilter *gallery.Visibility
	if q.Visibility != "" {
		visibility, err := gallery.ParseVisibility(q.Visibility)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("visibility", q.Visibility).
				Msg("invalid visibility filter in list query")
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}
		visibilityFilter = &visibility
	}

	// Parse tag filter if provided
	var tagFilter *gallery.Tag
	if q.Tag != "" {
		tag, err := gallery.NewTag(q.Tag)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", q.Tag).
				Msg("invalid tag filter in list query")
			return nil, fmt.Errorf("invalid tag: %w", err)
		}
		tagFilter = &tag
	}

	// Validate and set pagination defaults
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20 // Default page size
	}
	if limit > 100 {
		limit = 100 // Max page size
	}

	// Convert offset/limit to page-based pagination
	page := (offset / limit) + 1
	if page < 1 {
		page = 1
	}

	pagination, err := shared.NewPagination(page, limit)
	if err != nil {
		// This should not happen given our validation above, but handle it
		pagination = shared.DefaultPagination()
	}

	// 2. Determine which repository method to use based on filters
	var images []*gallery.Image
	var totalCount int64

	if tagFilter != nil {
		// List by tag (always public only)
		images, totalCount, err = h.images.FindByTag(ctx, *tagFilter, pagination)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("tag", q.Tag).
				Msg("failed to list images by tag")
			return nil, fmt.Errorf("list images by tag: %w", err)
		}
	} else if !ownerID.IsZero() {
		// List by owner
		images, totalCount, err = h.images.FindByOwner(ctx, ownerID, pagination)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("owner_id", ownerID.String()).
				Msg("failed to list images by owner")
			return nil, fmt.Errorf("list images by owner: %w", err)
		}

		// Filter by visibility if requester is not the owner
		if !requestingUserID.IsZero() && !ownerID.Equals(requestingUserID) {
			images = filterByVisibility(images, gallery.VisibilityPublic)
		} else if visibilityFilter != nil {
			images = filterByVisibility(images, *visibilityFilter)
		}
	} else {
		// List public images (no owner filter)
		images, totalCount, err = h.images.FindPublic(ctx, pagination)
		if err != nil {
			h.logger.Error().
				Err(err).
				Msg("failed to list public images")
			return nil, fmt.Errorf("list public images: %w", err)
		}
	}

	// 3. Convert to DTOs
	imageDTOs := make([]ImageDTO, 0, len(images))
	for _, image := range images {
		// Skip deleted or non-viewable images
		if !image.IsViewable() {
			continue
		}

		dto := ImageToDTO(image)
		imageDTOs = append(imageDTOs, *dto)
	}

	// 4. Build result with pagination metadata
	result := &ListImagesResult{
		Images:     imageDTOs,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
		HasMore:    int64(offset+limit) < totalCount,
	}

	h.logger.Debug().
		Str("owner_id", q.OwnerID).
		Str("requesting_user_id", q.RequestingUserID).
		Str("tag", q.Tag).
		Int("offset", offset).
		Int("limit", limit).
		Int("results", len(imageDTOs)).
		Int64("total_count", totalCount).
		Msg("images listed successfully")

	return result, nil
}

// filterByVisibility filters images by visibility setting.
// This is used for in-memory filtering when needed.
func filterByVisibility(images []*gallery.Image, visibility gallery.Visibility) []*gallery.Image {
	filtered := make([]*gallery.Image, 0, len(images))
	for _, image := range images {
		if image.Visibility() == visibility {
			filtered = append(filtered, image)
		}
	}
	return filtered
}
