package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/activity/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// ActivityHandler handles activity feed HTTP endpoints.
// It delegates to application layer query handlers for retrieving activity feeds.
type ActivityHandler struct {
	getFeedHandler *queries.GetActivityFeedHandler
	logger         zerolog.Logger
}

// NewActivityHandler creates a new ActivityHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewActivityHandler(
	getFeedHandler *queries.GetActivityFeedHandler,
	logger zerolog.Logger,
) *ActivityHandler {
	return &ActivityHandler{
		getFeedHandler: getFeedHandler,
		logger:         logger,
	}
}

// GetFeed handles GET /api/v1/feed
// Returns activities from users that the authenticated user follows.
//
// Query Parameters:
//   - limit: Maximum number of activities to return (default: 20, max: 100)
//   - offset: Pagination offset (default: 0)
//
// Response: 200 OK with activity feed
// Errors:
//   - 400: Invalid query parameters
//   - 401: Not authenticated
//   - 404: User not found
//   - 500: Internal server error
func (h *ActivityHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in activity feed handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Parse query parameters for pagination
	query := r.URL.Query()
	limit, err := parseLimit(query.Get("limit"), 20, 100)
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid limit parameter")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid limit parameter",
		)
		return
	}

	offset, err := parseOffset(query.Get("offset"))
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid offset parameter")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid offset parameter",
		)
		return
	}

	// 3. Delegate to query handler
	q := queries.GetActivityFeedQuery{
		UserID: userCtx.UserID.String(),
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.getFeedHandler.Handle(ctx, q)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get activity feed")
		return
	}

	// 4. Return feed
	h.logger.Debug().
		Str("user_id", userCtx.UserID.String()).
		Int("activity_count", len(result.Activities)).
		Int("total_count", result.TotalCount).
		Msg("activity feed retrieved successfully")

	EncodeJSON(w, http.StatusOK, result)
}

// mapErrorAndRespond maps domain errors to HTTP responses and writes them.
func (h *ActivityHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("error in activity handler")

	switch {
	case errors.Is(err, identity.ErrUserNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User not found",
		)

	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred",
		)
	}
}

// parseLimit parses and validates the limit query parameter.
// Returns defaultLimit if param is empty, or error if invalid.
func parseLimit(param string, defaultLimit, maxLimit int) (int, error) {
	if param == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(param)
	if err != nil {
		return 0, err
	}

	if limit < 1 {
		return 0, errors.New("limit must be positive")
	}

	if limit > maxLimit {
		return maxLimit, nil
	}

	return limit, nil
}

// parseOffset parses and validates the offset query parameter.
// Returns 0 if param is empty, or error if invalid.
func parseOffset(param string) (int, error) {
	if param == "" {
		return 0, nil
	}

	offset, err := strconv.Atoi(param)
	if err != nil {
		return 0, err
	}

	if offset < 0 {
		return 0, errors.New("offset must be non-negative")
	}

	return offset, nil
}
