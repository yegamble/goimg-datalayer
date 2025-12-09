package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// Maximum page size for image listings.
	maxPageSize = 100
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
	params, err := h.parseQueryParams(q)
	if err != nil {
		return nil, err
	}

	// 2. Load images based on filters
	images, totalCount, err := h.loadImages(ctx, params)
	if err != nil {
		return nil, err
	}

	// 3. Convert to DTOs
	imageDTOs := h.buildImageDTOs(images)

	// 4. Build result with pagination metadata
	result := &ListImagesResult{
		Images:     imageDTOs,
		TotalCount: totalCount,
		Offset:     params.offset,
		Limit:      params.limit,
		HasMore:    int64(params.offset+params.limit) < totalCount,
	}

	h.logger.Debug().
		Str("owner_id", q.OwnerID).
		Str("requesting_user_id", q.RequestingUserID).
		Str("tag", q.Tag).
		Int("offset", params.offset).
		Int("limit", params.limit).
		Int("results", len(imageDTOs)).
		Int64("total_count", totalCount).
		Msg("images listed successfully")

	return result, nil
}

// queryParams holds parsed query parameters for image listing.
type queryParams struct {
	ownerID          identity.UserID
	requestingUserID identity.UserID
	visibilityFilter *gallery.Visibility
	tagFilter        *gallery.Tag
	pagination       shared.Pagination
	offset           int
	limit            int
}

// parseQueryParams validates and parses all query parameters.
func (h *ListImagesHandler) parseQueryParams(q ListImagesQuery) (*queryParams, error) {
	params := &queryParams{}

	// Parse owner ID if provided
	if q.OwnerID != "" {
		ownerID, err := identity.ParseUserID(q.OwnerID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("owner_id", q.OwnerID).
				Msg("invalid owner id in list query")
			return nil, fmt.Errorf("invalid owner id: %w", err)
		}
		params.ownerID = ownerID
	}

	// Parse requesting user ID if provided
	if q.RequestingUserID != "" {
		requestingUserID, err := identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("requesting_user_id", q.RequestingUserID).
				Msg("invalid requesting user id in list query")
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
		params.requestingUserID = requestingUserID
	}

	// Parse visibility filter if provided
	if q.Visibility != "" {
		visibility, err := gallery.ParseVisibility(q.Visibility)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("visibility", q.Visibility).
				Msg("invalid visibility filter in list query")
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}
		params.visibilityFilter = &visibility
	}

	// Parse tag filter if provided
	if q.Tag != "" {
		tag, err := gallery.NewTag(q.Tag)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("tag", q.Tag).
				Msg("invalid tag filter in list query")
			return nil, fmt.Errorf("invalid tag: %w", err)
		}
		params.tagFilter = &tag
	}

	// Validate and set pagination
	params.offset, params.limit, params.pagination = h.normalizePagination(q.Offset, q.Limit)

	return params, nil
}

// normalizePagination validates and normalizes pagination parameters.
func (h *ListImagesHandler) normalizePagination(offset, limit int) (int, int, shared.Pagination) {
	if offset < 0 {
		offset = 0
	}

	if limit <= 0 {
		limit = 20 // Default page size
	}
	if limit > maxPageSize {
		limit = maxPageSize
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

	return offset, limit, pagination
}

// loadImages loads images from repository based on query parameters.
func (h *ListImagesHandler) loadImages(ctx context.Context, params *queryParams) ([]*gallery.Image, int64, error) {
	switch {
	case params.tagFilter != nil:
		return h.loadImagesByTag(ctx, params)
	case !params.ownerID.IsZero():
		return h.loadImagesByOwner(ctx, params)
	default:
		return h.loadPublicImages(ctx, params)
	}
}

// loadImagesByTag loads images filtered by tag.
func (h *ListImagesHandler) loadImagesByTag(ctx context.Context, params *queryParams) ([]*gallery.Image, int64, error) {
	images, totalCount, err := h.images.FindByTag(ctx, *params.tagFilter, params.pagination)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("tag", params.tagFilter.String()).
			Msg("failed to list images by tag")
		return nil, 0, fmt.Errorf("list images by tag: %w", err)
	}
	return images, totalCount, nil
}

// loadImagesByOwner loads images filtered by owner with visibility filtering.
func (h *ListImagesHandler) loadImagesByOwner(ctx context.Context, params *queryParams) ([]*gallery.Image, int64, error) {
	images, totalCount, err := h.images.FindByOwner(ctx, params.ownerID, params.pagination)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("owner_id", params.ownerID.String()).
			Msg("failed to list images by owner")
		return nil, 0, fmt.Errorf("list images by owner: %w", err)
	}

	// Apply visibility filtering
	images = h.applyVisibilityFilter(images, params)
	return images, totalCount, nil
}

// loadPublicImages loads all public images.
func (h *ListImagesHandler) loadPublicImages(ctx context.Context, params *queryParams) ([]*gallery.Image, int64, error) {
	images, totalCount, err := h.images.FindPublic(ctx, params.pagination)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to list public images")
		return nil, 0, fmt.Errorf("list public images: %w", err)
	}
	return images, totalCount, nil
}

// applyVisibilityFilter applies visibility filtering based on requester.
func (h *ListImagesHandler) applyVisibilityFilter(images []*gallery.Image, params *queryParams) []*gallery.Image {
	// Filter by visibility if requester is not the owner
	if !params.requestingUserID.IsZero() && !params.ownerID.Equals(params.requestingUserID) {
		return filterByVisibility(images, gallery.VisibilityPublic)
	}
	if params.visibilityFilter != nil {
		return filterByVisibility(images, *params.visibilityFilter)
	}
	return images
}

// buildImageDTOs converts images to DTOs, filtering out non-viewable images.
func (h *ListImagesHandler) buildImageDTOs(images []*gallery.Image) []ImageDTO {
	imageDTOs := make([]ImageDTO, 0, len(images))
	for _, image := range images {
		if !image.IsViewable() {
			continue
		}
		dto := ImageToDTO(image)
		imageDTOs = append(imageDTOs, *dto)
	}
	return imageDTOs
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
