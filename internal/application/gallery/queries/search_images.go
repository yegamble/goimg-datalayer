package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SearchImagesQuery represents a query to search images with full-text search and filters.
type SearchImagesQuery struct {
	Query      string   // Full-text search query (searches title and description)
	Tags       []string // Filter by tag names (AND logic for multiple tags)
	OwnerID    string   // Optional: filter by owner (empty = all)
	Visibility string   // Optional: filter by visibility (empty = public only)
	SortBy     string   // Sort order: relevance, created_at, view_count, like_count
	Page       int      // Page number (1-indexed)
	PerPage    int      // Items per page
}

// SearchImagesResult represents the search result with pagination and relevance scoring.
type SearchImagesResult struct {
	Images     []*ImageDTO `json:"images"`
	TotalCount int64       `json:"total_count"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	Query      string      `json:"query"`
}

// SearchImagesHandler processes image search queries.
// It uses PostgreSQL full-text search with ts_vector and ts_rank for relevance.
type SearchImagesHandler struct {
	images gallery.ImageRepository
}

// NewSearchImagesHandler creates a new SearchImagesHandler.
func NewSearchImagesHandler(images gallery.ImageRepository) *SearchImagesHandler {
	return &SearchImagesHandler{
		images: images,
	}
}

// Handle executes the image search query.
//
// Process flow:
//  1. Validate and parse search parameters
//  2. Convert tag names to domain Tag objects
//  3. Build SearchParams for repository
//  4. Execute search via repository (PostgreSQL full-text search)
//  5. Convert results to DTOs with pagination metadata
//
// Returns:
//   - SearchImagesResult with images, total count, and pagination info
//   - Validation errors if parameters are invalid
//
// Search features:
//   - Full-text search on title and description (PostgreSQL ts_vector)
//   - Tag filtering with AND logic (image must have all specified tags)
//   - Visibility filtering (defaults to public only)
//   - Owner filtering
//   - Multiple sort options: relevance, created_at, view_count, like_count
//   - Pagination with total count
func (h *SearchImagesHandler) Handle(ctx context.Context, q SearchImagesQuery) (*SearchImagesResult, error) {
	// 1. Validate and create pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		// Use defaults if invalid
		pagination = shared.DefaultPagination()
	}

	// 2. Parse and validate parameters
	var ownerID *identity.UserID
	if q.OwnerID != "" {
		parsedOwnerID, err := identity.ParseUserID(q.OwnerID)
		if err != nil {
			return nil, fmt.Errorf("invalid owner user id: %w", err)
		}
		ownerID = &parsedOwnerID
	}

	var visibility *gallery.Visibility
	if q.Visibility != "" {
		parsedVisibility, err := gallery.ParseVisibility(q.Visibility)
		if err != nil {
			return nil, fmt.Errorf("invalid visibility: %w", err)
		}
		visibility = &parsedVisibility
	} else {
		// Default to public if not specified
		publicVisibility := gallery.VisibilityPublic
		visibility = &publicVisibility
	}

	// 3. Convert tag names to domain Tag objects
	tags := make([]gallery.Tag, 0, len(q.Tags))
	for _, tagName := range q.Tags {
		tag, err := gallery.NewTag(tagName)
		if err != nil {
			return nil, fmt.Errorf("invalid tag '%s': %w", tagName, err)
		}
		tags = append(tags, tag)
	}

	// 4. Parse sort parameter
	sortBy := gallery.SearchSortByRelevance // Default to relevance
	if q.SortBy != "" {
		sortBy = gallery.SearchSortBy(q.SortBy)
		// Validate sort parameter
		validSorts := map[gallery.SearchSortBy]bool{
			gallery.SearchSortByRelevance: true,
			gallery.SearchSortByCreatedAt: true,
			gallery.SearchSortByViewCount: true,
			gallery.SearchSortByLikeCount: true,
		}
		if !validSorts[sortBy] {
			return nil, fmt.Errorf("invalid sort parameter: %s (must be one of: relevance, created_at, view_count, like_count)", q.SortBy)
		}
	}

	// 5. Build SearchParams
	searchParams := gallery.SearchParams{
		Query:      q.Query,
		Tags:       tags,
		OwnerID:    ownerID,
		Visibility: visibility,
		SortBy:     sortBy,
		Pagination: pagination,
	}

	// 6. Execute search
	images, total, err := h.images.Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("search images: %w", err)
	}

	// 7. Convert to DTOs
	imageDTOs := make([]*ImageDTO, 0, len(images))
	for _, image := range images {
		dto := ImageToDTO(image)
		imageDTOs = append(imageDTOs, dto)
	}

	// 8. Calculate pagination metadata
	pagination = pagination.WithTotal(total)

	return &SearchImagesResult{
		Images:     imageDTOs,
		TotalCount: total,
		Page:       pagination.Page(),
		PerPage:    pagination.PerPage(),
		TotalPages: pagination.TotalPages(),
		Query:      q.Query,
	}, nil
}
