package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// ExploreHandler handles explore/discovery HTTP endpoints.
// These endpoints are public and allow anonymous users to discover content.
type ExploreHandler struct {
	listImages   *queries.ListImagesHandler
	listFeatured *queries.ListFeaturedImagesHandler
	logger       zerolog.Logger
}

// NewExploreHandler creates a new ExploreHandler with the given dependencies.
func NewExploreHandler(
	listImages *queries.ListImagesHandler,
	listFeatured *queries.ListFeaturedImagesHandler,
	logger zerolog.Logger,
) *ExploreHandler {
	return &ExploreHandler{
		listImages:   listImages,
		listFeatured: listFeatured,
		logger:       logger,
	}
}

// Routes registers explore routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/explore
//
// All routes are public (no authentication required) and only return public images.
func (h *ExploreHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/recent", h.ListRecent)
	r.Get("/popular", h.ListPopular)
	r.Get("/featured", h.ListFeatured)

	return r
}

// ListRecent handles GET /api/v1/explore/recent
// Retrieves recently uploaded public images.
//
// Query parameters:
//   - page (int): Page number, default 1
//   - per_page (int): Items per page, default 20, max 100
//   - include_nsfw (bool): Include NSFW content, default false
//
// Response: Paginated list of public images sorted by created_at DESC
// Errors:
//   - 400: Invalid query parameters
func (h *ExploreHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(r.URL.Query().Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// Parse NSFW filter - default excludes NSFW for public explore
	includeNSFW := r.URL.Query().Get("include_nsfw") == "true"

	// Calculate offset
	offset := (page - 1) * perPage

	// Build query for recent public images
	query := queries.ListImagesQuery{
		Visibility:  "public",
		ExcludeNSFW: !includeNSFW, // Default to exclude NSFW
		Offset:      offset,
		Limit:       perPage,
		SortBy:      "created_at",
		SortOrder:   "desc",
	}

	// Execute query
	result, err := h.listImages.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to list recent images")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve recent images",
		)
		return
	}

	// Build response with pagination
	response := map[string]interface{}{
		"items": result.Images,
		"pagination": map[string]interface{}{
			"total":       result.TotalCount,
			"page":        page,
			"per_page":    perPage,
			"total_pages": (result.TotalCount + int64(perPage) - 1) / int64(perPage),
		},
	}

	h.logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Images)).
		Int64("total_count", result.TotalCount).
		Msg("explore recent images retrieved")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode explore recent response")
	}
}

// ListPopular handles GET /api/v1/explore/popular
// Retrieves popular public images based on like count.
//
// Query parameters:
//   - period (string): Time period (day, week, month, all), default "week"
//   - page (int): Page number, default 1
//   - per_page (int): Items per page, default 20, max 100
//   - include_nsfw (bool): Include NSFW content, default false
//
// Response: Paginated list of public images sorted by like_count DESC
// Errors:
//   - 400: Invalid query parameters
//
//nolint:funlen // HTTP handler with validation and response.
func (h *ExploreHandler) ListPopular(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse period parameter
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
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

	// Parse pagination parameters
	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(r.URL.Query().Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	// Parse NSFW filter - default excludes NSFW for public explore
	includeNSFW := r.URL.Query().Get("include_nsfw") == "true"

	// Calculate offset
	offset := (page - 1) * perPage

	// Build query for popular public images
	// Note: Period filtering would ideally be handled in the repository layer
	// For MVP, we sort by like_count across all time (period is metadata for future enhancement)
	query := queries.ListImagesQuery{
		Visibility:  "public",
		ExcludeNSFW: !includeNSFW, // Default to exclude NSFW
		Offset:      offset,
		Limit:       perPage,
		SortBy:      "like_count",
		SortOrder:   "desc",
	}

	// Execute query
	result, err := h.listImages.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("period", period).
			Msg("failed to list popular images")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve popular images",
		)
		return
	}

	// Build response with pagination
	response := map[string]interface{}{
		"items":  result.Images,
		"period": period,
		"pagination": map[string]interface{}{
			"total":       result.TotalCount,
			"page":        page,
			"per_page":    perPage,
			"total_pages": (result.TotalCount + int64(perPage) - 1) / int64(perPage),
		},
	}

	h.logger.Debug().
		Str("period", period).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Images)).
		Int64("total_count", result.TotalCount).
		Msg("explore popular images retrieved")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode explore popular response")
	}
}

// ListFeatured handles GET /api/v1/explore/featured
// Retrieves admin-curated featured images.
//
// Query parameters:
//   - limit (int): Maximum number of featured images, default 10, max 50
//
// Response: List of featured images with display order
// Errors:
//   - 400: Invalid query parameters
func (h *ExploreHandler) ListFeatured(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if handler is configured
	if h.listFeatured == nil {
		h.logger.Warn().Msg("listFeatured handler not configured")
		middleware.WriteError(w, r,
			http.StatusNotImplemented,
			"Not Implemented",
			"Featured images endpoint is not configured",
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
	query := queries.ListFeaturedImagesQuery{
		Limit: limit,
	}

	result, err := h.listFeatured.Handle(ctx, query)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to list featured images")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve featured images",
		)
		return
	}

	h.logger.Debug().
		Int("limit", limit).
		Int("results", result.TotalCount).
		Msg("explore featured images retrieved")

	if err := EncodeJSON(w, http.StatusOK, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode explore featured response")
	}
}
