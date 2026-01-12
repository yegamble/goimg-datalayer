package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

const (
	// Default and maximum limits for tag search.
	defaultSearchTagLimit = 10
	maxSearchTagLimit     = 50
	minSearchQueryLength  = 2
)

// SearchTagsQuery searches for tags by prefix for autocomplete functionality.
type SearchTagsQuery struct {
	// Query is the search prefix (minimum 2 characters).
	Query string
	// Limit is the maximum number of tags to return (1-50, default 10).
	Limit int
}

// SearchTagsResult represents the result of a tag search query.
type SearchTagsResult struct {
	Tags  []TagWithUsageDTO `json:"tags"`
	Query string   `json:"query"`
	Limit int      `json:"limit"`
}

// SearchTagsHandler processes SearchTagsQuery requests for autocomplete.
type SearchTagsHandler struct {
	tags   gallery.TagRepository
	logger *zerolog.Logger
}

// NewSearchTagsHandler creates a new SearchTagsHandler.
func NewSearchTagsHandler(
	tags gallery.TagRepository,
	logger *zerolog.Logger,
) *SearchTagsHandler {
	return &SearchTagsHandler{
		tags:   tags,
		logger: logger,
	}
}

// Handle executes the SearchTagsQuery and returns matching tags.
//
// Tags are searched by prefix matching on both name and slug.
// Results are sorted by usage count (most popular first).
func (h *SearchTagsHandler) Handle(ctx context.Context, query SearchTagsQuery) (*SearchTagsResult, error) {
	// Validate query length
	if len(query.Query) < minSearchQueryLength {
		return &SearchTagsResult{
			Tags:  []TagWithUsageDTO{},
			Query: query.Query,
			Limit: query.Limit,
		}, nil
	}

	// Validate and normalize limit
	limit := query.Limit
	if limit < 1 {
		limit = defaultSearchTagLimit
	}
	if limit > maxSearchTagLimit {
		limit = maxSearchTagLimit
	}

	h.logger.Debug().
		Str("query", query.Query).
		Int("limit", limit).
		Msg("searching tags")

	// Search tags from repository
	tagsWithUsage, err := h.tags.SearchByPrefix(ctx, query.Query, limit)
	if err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}

	// Convert to DTOs
	tagDTOs := make([]TagWithUsageDTO, 0, len(tagsWithUsage))
	for _, t := range tagsWithUsage {
		tagDTOs = append(tagDTOs, TagWithUsageDTO{
			Name:       t.Tag.Name(),
			Slug:       t.Tag.Slug(),
			UsageCount: t.UsageCount,
		})
	}

	return &SearchTagsResult{
		Tags:  tagDTOs,
		Query: query.Query,
		Limit: limit,
	}, nil
}
