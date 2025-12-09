package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// UserHandler handles user-related HTTP endpoints.
// It delegates to application layer command/query handlers for business logic.
type UserHandler struct {
	getUserHandler     *queries.GetUserHandler
	updateUserHandler  *commands.UpdateUserHandler
	deleteUserHandler  *commands.DeleteUserHandler
	getSessionsHandler *queries.GetUserSessionsHandler
	logger             zerolog.Logger
}

// NewUserHandler creates a new UserHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewUserHandler(
	getUserHandler *queries.GetUserHandler,
	updateUserHandler *commands.UpdateUserHandler,
	deleteUserHandler *commands.DeleteUserHandler,
	getSessionsHandler *queries.GetUserSessionsHandler,
	logger zerolog.Logger,
) *UserHandler {
	return &UserHandler{
		getUserHandler:     getUserHandler,
		updateUserHandler:  updateUserHandler,
		deleteUserHandler:  deleteUserHandler,
		getSessionsHandler: getSessionsHandler,
		logger:             logger,
	}
}

// Routes registers user routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/users
//
// All routes require JWT authentication (should be applied by parent router).
//
// Usage:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.JWTAuth)
//	    r.Mount("/api/v1/users", userHandler.Routes())
//	})
func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All routes require authentication (enforced by parent router)
	r.Get("/{id}", h.GetUser)
	r.Put("/{id}", h.UpdateUser)
	r.Delete("/{id}", h.DeleteUser)
	r.Get("/{id}/sessions", h.GetUserSessions)

	return r
}

// GetUser handles GET /api/v1/users/{id}
// Retrieves user profile information by ID.
//
// Path Parameters:
//   - id: User UUID
//
// Response: 200 OK with UserDTO
// Errors:
//   - 400: Invalid user ID format
//   - 404: User not found
//   - 500: Internal server error
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate user ID from path parameter
	userID, err := GetPathParamUUID(r, "id")
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid user id in path")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 2. Get authenticated user context for audit
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 3. Delegate to query handler
	query := queries.GetUserQuery{
		UserID:      userID,
		RequestorID: userCtx.UserID,
	}

	userDTO, err := h.getUserHandler.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get user")
		return
	}

	// 4. Return user data
	if err := EncodeJSON(w, http.StatusOK, userDTO); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get user response")
	}
}

// UpdateUser handles PUT /api/v1/users/{id}
// Updates user profile information (display name, bio).
//
// Path Parameters:
//   - id: User UUID
//
// Request: UpdateUserRequest JSON body
// Response: 200 OK with updated UserDTO
// Errors:
//   - 400: Invalid request body or user ID format
//   - 403: Not authorized to update this user
//   - 404: User not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate user ID from path parameter
	userID, err := GetPathParamUUID(r, "id")
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid user id in path")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 2. Get authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 3. Verify ownership (user can only update their own profile, unless admin)
	if !ValidateOwnership(userCtx, userID) {
		h.logger.Warn().
			Str("requestor_id", userCtx.UserID.String()).
			Str("target_user_id", userID.String()).
			Msg("unauthorized user update attempt")
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to update this user",
		)
		return
	}

	// 4. Decode and validate request
	var req UpdateUserRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update user request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid update data",
			validationErrors,
		)
		return
	}

	// 5. Delegate to command handler
	cmd := commands.UpdateUserCommand{
		UserID:      userID,
		RequestorID: userCtx.UserID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
	}

	userDTO, err := h.updateUserHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update user")
		return
	}

	// 6. Return updated user data
	h.logger.Info().
		Str("user_id", userID.String()).
		Str("requestor_id", userCtx.UserID.String()).
		Msg("user profile updated successfully")

	if err := EncodeJSON(w, http.StatusOK, userDTO); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update user response")
	}
}

// DeleteUser handles DELETE /api/v1/users/{id}
// Soft-deletes a user account (requires password confirmation).
//
// Path Parameters:
//   - id: User UUID
//
// Request: DeleteUserRequest JSON body (password confirmation)
// Response: 204 No Content
// Errors:
//   - 400: Invalid request body or user ID format
//   - 401: Invalid password
//   - 403: Not authorized to delete this user
//   - 404: User not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate user ID from path parameter
	userID, err := GetPathParamUUID(r, "id")
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid user id in path")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 2. Get authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 3. Verify ownership (user can only delete their own account, unless admin)
	if !ValidateOwnership(userCtx, userID) {
		h.logger.Warn().
			Str("requestor_id", userCtx.UserID.String()).
			Str("target_user_id", userID.String()).
			Msg("unauthorized user delete attempt")
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to delete this user",
		)
		return
	}

	// 4. Decode and validate request (password confirmation required)
	var req DeleteUserRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid delete user request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Password confirmation is required",
			validationErrors,
		)
		return
	}

	// 5. Delegate to command handler
	cmd := commands.DeleteUserCommand{
		UserID:      userID,
		RequestorID: userCtx.UserID,
		Password:    req.Password,
	}

	if err := h.deleteUserHandler.Handle(ctx, cmd); err != nil {
		// Check for invalid password error specifically
		if errors.Is(err, identity.ErrInvalidCredentials) {
			h.logger.Warn().
				Str("user_id", userID.String()).
				Msg("delete user failed - invalid password")
			middleware.WriteError(w, r,
				http.StatusUnauthorized,
				"Unauthorized",
				"Invalid password",
			)
			return
		}

		h.mapErrorAndRespond(w, r, err, "delete user")
		return
	}

	// 6. Deletion successful - return 204 No Content
	h.logger.Info().
		Str("user_id", userID.String()).
		Str("requestor_id", userCtx.UserID.String()).
		Msg("user account deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// GetUserSessions handles GET /api/v1/users/{id}/sessions
// Retrieves all active sessions for a user.
//
// Path Parameters:
//   - id: User UUID
//
// Response: 200 OK with []SessionDTO
// Errors:
//   - 400: Invalid user ID format
//   - 403: Not authorized to view sessions for this user
//   - 404: User not found
//   - 500: Internal server error
func (h *UserHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate user ID from path parameter
	userID, err := GetPathParamUUID(r, "id")
	if err != nil {
		h.logger.Debug().Err(err).Msg("invalid user id in path")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid user ID format",
		)
		return
	}

	// 2. Get authenticated user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 3. Verify ownership (user can only view their own sessions, unless admin)
	if !ValidateOwnership(userCtx, userID) {
		h.logger.Warn().
			Str("requestor_id", userCtx.UserID.String()).
			Str("target_user_id", userID.String()).
			Msg("unauthorized sessions access attempt")
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to view sessions for this user",
		)
		return
	}

	// 4. Delegate to query handler
	query := queries.GetUserSessionsQuery{
		UserID:      userID,
		RequestorID: userCtx.UserID,
	}

	sessionDTOs, err := h.getSessionsHandler.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get user sessions")
		return
	}

	// 5. Mark the current session in the response
	for i := range sessionDTOs {
		if sessionDTOs[i].SessionID == userCtx.SessionID.String() {
			sessionDTOs[i].IsCurrent = true
		}
	}

	// 6. Return sessions list
	if err := EncodeJSON(w, http.StatusOK, sessionDTOs); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get sessions response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
// This centralizes error mapping logic for consistency across all user endpoints.
func (h *UserHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("user operation failed")

	// Map specific domain/application errors to HTTP status codes
	switch {
	case errors.Is(err, identity.ErrUserNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User not found",
		)

	case errors.Is(err, identity.ErrInvalidCredentials):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Invalid password",
		)

	case errors.Is(err, identity.ErrEmailInvalid),
		errors.Is(err, identity.ErrEmailEmpty),
		errors.Is(err, identity.ErrEmailTooLong),
		errors.Is(err, identity.ErrUsernameInvalid),
		errors.Is(err, identity.ErrUsernameEmpty),
		errors.Is(err, identity.ErrUsernameTooShort),
		errors.Is(err, identity.ErrUsernameTooLong):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			err.Error(),
		)

	default:
		// Check for authorization errors (string matching as fallback)
		if errors.Is(err, errors.New("unauthorized")) ||
			(err != nil && (err.Error() == "unauthorized: cannot update another user's profile" ||
				err.Error() == "unauthorized: cannot delete another user's account" ||
				err.Error() == "unauthorized: cannot view sessions for another user")) {
			middleware.WriteError(w, r,
				http.StatusForbidden,
				"Forbidden",
				"You do not have permission to perform this action",
			)
			return
		}

		// Unknown error - return generic 500 without exposing internal details
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
	}
}
