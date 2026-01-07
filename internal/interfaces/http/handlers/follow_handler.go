package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// FollowHandler handles user follow/unfollow HTTP endpoints.
// It delegates to application layer command/query handlers for business logic.
type FollowHandler struct {
	followUserHandler   *commands.FollowUserHandler
	unfollowUserHandler *commands.UnfollowUserHandler
	getFollowersHandler *queries.GetFollowersHandler
	getFollowingHandler *queries.GetFollowingHandler
	logger              zerolog.Logger
}

// NewFollowHandler creates a new FollowHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewFollowHandler(
	followUserHandler *commands.FollowUserHandler,
	unfollowUserHandler *commands.UnfollowUserHandler,
	getFollowersHandler *queries.GetFollowersHandler,
	getFollowingHandler *queries.GetFollowingHandler,
	logger zerolog.Logger,
) *FollowHandler {
	return &FollowHandler{
		followUserHandler:   followUserHandler,
		unfollowUserHandler: unfollowUserHandler,
		getFollowersHandler: getFollowersHandler,
		getFollowingHandler: getFollowingHandler,
		logger:              logger,
	}
}

// FollowUser handles POST /api/v1/users/{id}/follow
// Authenticated user follows the specified user.
//
// Path Parameters:
// - id: User ID to follow
//
// Response: 204 No Content on success
// Errors:
//   - 400: Invalid user ID format or cannot follow self
//   - 401: Not authenticated
//   - 404: Target user not found
//   - 409: Already following this user
//   - 500: Internal server error
func (h *FollowHandler) FollowUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context (follower)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in follow handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract target user ID from path parameter (followed)
	targetUserID := GetPathParam(r, "id")
	if targetUserID == "" {
		h.logger.Debug().Msg("missing user ID in follow request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 3. Delegate to command handler
	cmd := commands.FollowUserCommand{
		FollowerID: userCtx.UserID.String(),
		FollowedID: targetUserID,
	}

	err = h.followUserHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "follow user")
		return
	}

	// 4. Follow successful - return 204 No Content
	h.logger.Info().
		Str("follower_id", userCtx.UserID.String()).
		Str("followed_id", targetUserID).
		Msg("user followed successfully")

	w.WriteHeader(http.StatusNoContent)
}

// UnfollowUser handles DELETE /api/v1/users/{id}/follow
// Authenticated user unfollows the specified user.
//
// Path Parameters:
// - id: User ID to unfollow
//
// Response: 204 No Content on success
// Errors:
//   - 400: Invalid user ID format
//   - 401: Not authenticated
//   - 500: Internal server error
//
// Note: This operation is idempotent - unfollowing a user you don't follow succeeds.
func (h *FollowHandler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract authenticated user context (follower)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in unfollow handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract target user ID from path parameter (followed)
	targetUserID := GetPathParam(r, "id")
	if targetUserID == "" {
		h.logger.Debug().Msg("missing user ID in unfollow request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 3. Delegate to command handler
	cmd := commands.UnfollowUserCommand{
		FollowerID: userCtx.UserID.String(),
		FollowedID: targetUserID,
	}

	err = h.unfollowUserHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "unfollow user")
		return
	}

	// 4. Unfollow successful - return 204 No Content
	h.logger.Info().
		Str("follower_id", userCtx.UserID.String()).
		Str("followed_id", targetUserID).
		Msg("user unfollowed successfully")

	w.WriteHeader(http.StatusNoContent)
}

// GetFollowers handles GET /api/v1/users/{id}/followers
// Returns paginated list of users following the specified user.
//
// Path Parameters:
// - id: User ID whose followers to retrieve
//
// Query Parameters:
// - limit: Maximum number of items to return (default: 20, max: 100)
// - offset: Number of items to skip for pagination (default: 0)
//
// Response: 200 OK with FollowersListDTO
// Errors:
//   - 400: Invalid user ID format or invalid pagination parameters
//   - 404: User not found
//   - 500: Internal server error
func (h *FollowHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from path parameter
	userID := GetPathParam(r, "id")
	if userID == "" {
		h.logger.Debug().Msg("missing user ID in get followers request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 2. Parse pagination parameters
	limit, offset := parsePaginationParams(r)

	// 3. Delegate to query handler
	query := queries.GetFollowersQuery{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.getFollowersHandler.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get followers")
		return
	}

	// 4. Return followers list
	if err := EncodeJSON(w, http.StatusOK, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode followers list response")
	}
}

// GetFollowing handles GET /api/v1/users/{id}/following
// Returns paginated list of users that the specified user is following.
//
// Path Parameters:
// - id: User ID whose following list to retrieve
//
// Query Parameters:
// - limit: Maximum number of items to return (default: 20, max: 100)
// - offset: Number of items to skip for pagination (default: 0)
//
// Response: 200 OK with FollowingListDTO
// Errors:
//   - 400: Invalid user ID format or invalid pagination parameters
//   - 404: User not found
//   - 500: Internal server error
func (h *FollowHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from path parameter
	userID := GetPathParam(r, "id")
	if userID == "" {
		h.logger.Debug().Msg("missing user ID in get following request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing user ID",
		)
		return
	}

	// 2. Parse pagination parameters
	limit, offset := parsePaginationParams(r)

	// 3. Delegate to query handler
	query := queries.GetFollowingQuery{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.getFollowingHandler.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get following")
		return
	}

	// 4. Return following list
	if err := EncodeJSON(w, http.StatusOK, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode following list response")
	}
}

// parsePaginationParams extracts limit and offset from query parameters.
// Applies sensible defaults and constraints.
//
// Defaults:
// - limit: 20 (min: 1, max: 100)
// - offset: 0 (min: 0)
//
// Returns (limit, offset)
func parsePaginationParams(r *http.Request) (int, int) {
	const (
		defaultLimit = 20
		maxLimit     = 100
		minLimit     = 1
	)

	// Parse limit
	limit := defaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
			// Enforce constraints
			if limit < minLimit {
				limit = minLimit
			}
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	// Parse offset
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
// This centralizes error mapping logic for consistency across all follow endpoints.
func (h *FollowHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("follow operation failed")

	// Map specific domain errors to HTTP status codes
	switch {
	case errors.Is(err, identity.ErrUserNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User not found",
		)

	case errors.Is(err, identity.ErrCannotFollowSelf):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Cannot follow yourself",
		)

	case errors.Is(err, identity.ErrFollowAlreadyExists):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"You are already following this user",
		)

	case errors.Is(err, identity.ErrFollowNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Follow relationship not found",
		)

	// Check for error messages from command handlers
	case err != nil && err.Error() == "cannot follow inactive user":
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Cannot follow inactive user",
		)

	default:
		// Unknown error - return generic 500 without exposing internal details
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
	}
}
