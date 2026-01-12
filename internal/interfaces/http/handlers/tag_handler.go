package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// TagHandler handles tag-related HTTP endpoints.
// These endpoints are public and allow anonymous access.
type TagHandler struct {
	listPopular  *queries.ListPopularTagsHandler
	listTrending *queries.ListTrendingTagsHandler
	searchTags   *queries.SearchTagsHandler
	logger       zerolog.Logger
}

// NewTagHandler creates a new TagHandler with the given dependencies.
func NewTagHandler(
	listPopular *queries.ListPopularTagsHandler,
	listTrending *queries.ListTrendingTagsHandler,
	searchTags *queries.SearchTagsHandler,
	logger zerolog.Logger,
) *TagHandler {
	return &TagHandler{
		listPopular:  listPopular,
		listTrending: listTrending,
		searchTags:   searchTags,
		logger:       logger,
	}
}

// Routes registers tag routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/tags
//
// All routes are public (no authentication required).
func (h *TagHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/popular", h.ListPopular)
	r.Get("/trending", h.ListTrending)
	r.Get("/search", h.Search)

	return r
}

// ListPopular handles GET /api/v1/tags/popular
// Retrieves the most popular tags by usage count.
//
// Query parameters:
//   - limit (int): Maximum tags to return, default 20, max 100
//   - period (string): Time period (day, week, month, all), default "all"
//
// Response: List of popular tags sorted by usage count
// Errors:
//   - 400: Invalid query parameters
func (h *TagHandler) ListPopular(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse limit parameter
	limit, err := parseIntParam(r.URL.Query().Get("limit"), 20)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Parse period parameter
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}

	// Validate period
	validPeriods := map[string]bool{"day": true, "week": true, "month": true, "all": true}
	if !validPeriods[period] {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid period. Must be one of: day, week, month, all",
		)
		return
	}

	// Build and execute query
	query := queries.ListPopularTagsQuery{
		Limit:  limit,
		Period: period,
	}

	result, err := h.listPopular.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("period", period).
			Int("limit", limit).
			Msg("failed to list popular tags")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve popular tags",
		)
		return
	}

	// Build response
	response := map[string]interface{}{
		"tags":   result.Tags,
		"period": result.Period,
		"limit":  result.Limit,
	}

	h.logger.Debug().
		Str("period", period).
		Int("limit", limit).
		Int("results", len(result.Tags)).
		Msg("popular tags retrieved")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode popular tags response")
	}
}

// ListTrending handles GET /api/v1/tags/trending
// Retrieves trending tags with time-weighted popularity.
//
// Query parameters:
//   - limit (int): Maximum tags to return, default 20, max 100
//   - period (string): Time period (day, week, month), default "week"
//
// Response: List of trending tags with trend scores
// Errors:
//   - 400: Invalid query parameters
func (h *TagHandler) ListTrending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse limit parameter
	limit, err := parseIntParam(r.URL.Query().Get("limit"), 20)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Parse period parameter - trending makes most sense for recent time windows
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	// Validate period (all time doesn't make sense for trending)
	validPeriods := map[string]bool{"day": true, "week": true, "month": true}
	if !validPeriods[period] {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid period. Must be one of: day, week, month",
		)
		return
	}

	// Build and execute query
	query := queries.ListTrendingTagsQuery{
		Limit:  limit,
		Period: period,
	}

	result, err := h.listTrending.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("period", period).
			Int("limit", limit).
			Msg("failed to list trending tags")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve trending tags",
		)
		return
	}

	// Build response
	response := map[string]interface{}{
		"tags":   result.Tags,
		"period": result.Period,
		"limit":  result.Limit,
	}

	h.logger.Debug().
		Str("period", period).
		Int("limit", limit).
		Int("results", len(result.Tags)).
		Msg("trending tags retrieved")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode trending tags response")
	}
}

// Search handles GET /api/v1/tags/search
// Searches tags by prefix for autocomplete functionality.
//
// Query parameters:
//   - q (string, required): Search query (minimum 2 characters)
//   - limit (int): Maximum tags to return, default 10, max 50
//
// Response: List of matching tags sorted by usage count
// Errors:
//   - 400: Missing or too short query parameter
func (h *TagHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameter (required)
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Query parameter 'q' must be at least 2 characters",
		)
		return
	}

	// Parse limit parameter
	limit, err := parseIntParam(r.URL.Query().Get("limit"), 10)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Build and execute query
	query := queries.SearchTagsQuery{
		Query: q,
		Limit: limit,
	}

	result, err := h.searchTags.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("query", q).
			Int("limit", limit).
			Msg("failed to search tags")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to search tags",
		)
		return
	}

	// Build response
	response := map[string]interface{}{
		"tags":  result.Tags,
		"query": result.Query,
		"limit": result.Limit,
	}

	h.logger.Debug().
		Str("query", q).
		Int("limit", limit).
		Int("results", len(result.Tags)).
		Msg("tags search completed")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode tag search response")
	}
}
