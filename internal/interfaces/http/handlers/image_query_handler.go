package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryParams := r.URL.Query()

	ownerID := queryParams.Get("owner_id")
	visibility := queryParams.Get("visibility")
	tag := queryParams.Get("tag")
	sortBy := queryParams.Get("sort_by")
	sortOrder := queryParams.Get("sort_order")

	offset, err := parseIntParam(queryParams.Get("offset"), 0)
	if err != nil {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid offset parameter",
		)
		return
	}

	limit, err := parseIntParam(queryParams.Get("limit"), defaultPerPage)
	if err != nil {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid limit parameter",
		)
		return
	}

	if limit > maxPerPage {
		limit = maxPerPage
	}

	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	query := queries.ListImagesQuery{
		OwnerID:          ownerID,
		Visibility:       visibility,
		Tag:              tag,
		RequestingUserID: requestingUserID,
		Offset:           offset,
		Limit:            limit,
		SortBy:           sortBy,
		SortOrder:        sortOrder,
	}

	result, err := h.listImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list images")
		return
	}

	h.logger.Debug().
		Str("owner_id", ownerID).
		Str("tag", tag).
		Int("offset", offset).
		Int("limit", limit).
		Int("results", len(result.Images)).
		Int64("total_count", result.TotalCount).
		Msg("images listed successfully")

	response := PaginatedImagesResponse{
		Images:     result.Images,
		TotalCount: result.TotalCount,
		Offset:     result.Offset,
		Limit:      result.Limit,
		HasMore:    result.HasMore,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list images response")
	}
}

//nolint:funlen,cyclop // HTTP handler with multiple query parameters and validation.
func (h *ImageHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryParams := r.URL.Query()

	searchQuery := queryParams.Get("q")
	if searchQuery == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Search query (q) is required",
		)
		return
	}

	ownerID := queryParams.Get("owner_id")
	visibility := queryParams.Get("visibility")
	sortBy := queryParams.Get("sort_by")
	if sortBy == "" {
		sortBy = "relevance"
	}

	var tags []string
	if tagsParam := queryParams.Get("tags"); tagsParam != "" {
		rawTags := strings.Split(tagsParam, ",")
		for _, tag := range rawTags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				tags = append(tags, trimmedTag)
			}
		}
	}

	page, err := parseIntParam(queryParams.Get("page"), 1)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := parseIntParam(queryParams.Get("per_page"), defaultPerPage)
	if err != nil || perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	query := queries.SearchImagesQuery{
		Query:      searchQuery,
		Tags:       tags,
		OwnerID:    ownerID,
		Visibility: visibility,
		SortBy:     sortBy,
		Page:       page,
		PerPage:    perPage,
	}

	result, err := h.searchImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "search images")
		return
	}

	h.logger.Debug().
		Str("query", searchQuery).
		Strs("tags", tags).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Images)).
		Int64("total_count", result.TotalCount).
		Msg("image search completed")

	if err := EncodeJSON(w, http.StatusOK, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode search images response")
	}
}

func parseIntParam(param string, defaultValue int) (int, error) {
	if param == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(param)
	if err != nil {
		return defaultValue, fmt.Errorf("parse int param: %w", err)
	}
	return value, nil
}
