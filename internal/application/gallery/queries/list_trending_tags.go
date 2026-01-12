package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// ListTrendingTagsQuery retrieves trending tags with time-weighted popularity.
type ListTrendingTagsQuery struct {
	// Limit is the maximum number of tags to return (1-100, default 20).
	Limit int
	// Period defines the time window for trending calculation: "day", "week", "month".
	Period string
}

// ListTrendingTagsResult represents the result of a trending tags query.
type ListTrendingTagsResult struct {
	Tags   []TagWithUsageDTO `json:"tags"`
	Period string   `json:"period"`
	Limit  int      `json:"limit"`
}

// ListTrendingTagsHandler processes ListTrendingTagsQuery requests.
type ListTrendingTagsHandler struct {
	tags   gallery.TagRepository
	logger *zerolog.Logger
}

// NewListTrendingTagsHandler creates a new ListTrendingTagsHandler.
func NewListTrendingTagsHandler(
	tags gallery.TagRepository,
	logger *zerolog.Logger,
) *ListTrendingTagsHandler {
	return &ListTrendingTagsHandler{
		tags:   tags,
		logger: logger,
	}
}

// Handle executes the ListTrendingTagsQuery and returns trending tags.
//
// The trending algorithm calculates: (recent_count / total_count) * 100
// This highlights tags that have high recent activity relative to their total usage.
func (h *ListTrendingTagsHandler) Handle(ctx context.Context, query ListTrendingTagsQuery) (*ListTrendingTagsResult, error) {
	// Validate and normalize limit
	limit := query.Limit
	if limit < 1 {
		limit = defaultTagLimit
	}
	if limit > maxTagLimit {
		limit = maxTagLimit
	}

	// Validate and normalize period
	// Trending makes most sense for recent time windows, default to week
	period := gallery.TagPeriod(query.Period)
	if !period.IsValid() {
		period = gallery.TagPeriodWeek
	}

	h.logger.Debug().
		Int("limit", limit).
		Str("period", string(period)).
		Msg("fetching trending tags")

	// Fetch trending tags from repository
	tagsWithUsage, err := h.tags.FindTrending(ctx, limit, period)
	if err != nil {
		return nil, fmt.Errorf("find trending tags: %w", err)
	}

	// Convert to DTOs
	tagDTOs := make([]TagWithUsageDTO, 0, len(tagsWithUsage))
	for _, t := range tagsWithUsage {
		tagDTOs = append(tagDTOs, TagWithUsageDTO{
			Name:       t.Tag.Name(),
			Slug:       t.Tag.Slug(),
			UsageCount: t.UsageCount,
			TrendScore: t.TrendScore,
		})
	}

	return &ListTrendingTagsResult{
		Tags:   tagDTOs,
		Period: string(period),
		Limit:  limit,
	}, nil
}
