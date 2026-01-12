package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

const (
	// Default and maximum limits for tag queries.
	defaultTagLimit = 20
	maxTagLimit     = 100
)

// ListPopularTagsQuery retrieves the most popular tags by usage count.
type ListPopularTagsQuery struct {
	// Limit is the maximum number of tags to return (1-100, default 20).
	Limit int
	// Period filters popularity by time window: "day", "week", "month", "all".
	Period string
}

// TagWithUsageDTO represents a tag with usage statistics for API responses.
type TagWithUsageDTO struct {
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	UsageCount int64   `json:"usage_count"`
	TrendScore float64 `json:"trend_score,omitempty"`
}

// ListPopularTagsResult represents the result of a popular tags query.
type ListPopularTagsResult struct {
	Tags   []TagWithUsageDTO `json:"tags"`
	Period string   `json:"period"`
	Limit  int      `json:"limit"`
}

// ListPopularTagsHandler processes ListPopularTagsQuery requests.
type ListPopularTagsHandler struct {
	tags   gallery.TagRepository
	logger *zerolog.Logger
}

// NewListPopularTagsHandler creates a new ListPopularTagsHandler.
func NewListPopularTagsHandler(
	tags gallery.TagRepository,
	logger *zerolog.Logger,
) *ListPopularTagsHandler {
	return &ListPopularTagsHandler{
		tags:   tags,
		logger: logger,
	}
}

// Handle executes the ListPopularTagsQuery and returns popular tags.
func (h *ListPopularTagsHandler) Handle(ctx context.Context, query ListPopularTagsQuery) (*ListPopularTagsResult, error) {
	// Validate and normalize limit
	limit := query.Limit
	if limit < 1 {
		limit = defaultTagLimit
	}
	if limit > maxTagLimit {
		limit = maxTagLimit
	}

	// Validate and normalize period
	period := gallery.TagPeriod(query.Period)
	if !period.IsValid() {
		period = gallery.TagPeriodAll
	}

	h.logger.Debug().
		Int("limit", limit).
		Str("period", string(period)).
		Msg("fetching popular tags")

	// Fetch popular tags from repository
	tagsWithUsage, err := h.tags.FindPopular(ctx, limit, period)
	if err != nil {
		return nil, fmt.Errorf("find popular tags: %w", err)
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

	return &ListPopularTagsResult{
		Tags:   tagDTOs,
		Period: string(period),
		Limit:  limit,
	}, nil
}
