package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// ExploreHandler handles explore/discovery HTTP endpoints.
// These endpoints are public and allow anonymous users to discover content.
type ExploreHandler struct {
	listImages *queries.ListImagesHandler
	logger     zerolog.Logger
}

// NewExploreHandler creates a new ExploreHandler with the given dependencies.
func NewExploreHandler(
	listImages *queries.ListImagesHandler,
	logger zerolog.Logger,
) *ExploreHandler {
	return &ExploreHandler{
		listImages: listImages,
		logger:     logger,
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

	return r
}

// ListRecent handles GET /api/v1/explore/recent
// Retrieves recently uploaded public images.
//
// Query parameters:
//   - page (int): Page number, default 1
//   - per_page (int): Items per page, default 20, max 100
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

	perPage, err := parseIntParam(r.URL.Query().Get("per_page"), 20)
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Build query for recent public images
	query := queries.ListImagesQuery{
		Visibility: "public",
		Offset:     offset,
		Limit:      perPage,
		SortBy:     "created_at",
		SortOrder:  "desc",
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
//
// Response: Paginated list of public images sorted by like_count DESC
// Errors:
//   - 400: Invalid query parameters
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

	perPage, err := parseIntParam(r.URL.Query().Get("per_page"), 20)
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Build query for popular public images
	// Note: Period filtering would ideally be handled in the repository layer
	// For MVP, we sort by like_count across all time (period is metadata for future enhancement)
	query := queries.ListImagesQuery{
		Visibility: "public",
		Offset:     offset,
		Limit:      perPage,
		SortBy:     "like_count",
		SortOrder:  "desc",
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
