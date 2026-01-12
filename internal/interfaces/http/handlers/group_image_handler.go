package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/community/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// GroupImageHandler handles group image moderation HTTP endpoints.
// It manages the image sharing and moderation queue for groups.
type GroupImageHandler struct {
	shareImage         *commands.ShareImageToGroupHandler
	approveImage       *commands.ApproveGroupImageHandler
	rejectImage        *commands.RejectGroupImageHandler
	listPendingImages  *queries.ListPendingGroupImagesHandler
	listApprovedImages *queries.ListApprovedGroupImagesHandler
	logger             zerolog.Logger
}

// NewGroupImageHandler creates a new GroupImageHandler with the given dependencies.
func NewGroupImageHandler(
	shareImage *commands.ShareImageToGroupHandler,
	approveImage *commands.ApproveGroupImageHandler,
	rejectImage *commands.RejectGroupImageHandler,
	listPendingImages *queries.ListPendingGroupImagesHandler,
	listApprovedImages *queries.ListApprovedGroupImagesHandler,
	logger zerolog.Logger,
) *GroupImageHandler {
	return &GroupImageHandler{
		shareImage:         shareImage,
		approveImage:       approveImage,
		rejectImage:        rejectImage,
		listPendingImages:  listPendingImages,
		listApprovedImages: listApprovedImages,
		logger:             logger,
	}
}

// Routes returns group image routes that require authentication.
// These should be mounted under /api/v1/groups/{groupID}/images.
func (h *GroupImageHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Image pool routes
	r.Get("/", h.ListApprovedImages) // List approved images in group
	r.Post("/", h.ShareImage)        // Share image to group

	// Moderation queue routes (admin+ only)
	r.Get("/pending", h.ListPendingImages)            // List pending images
	r.Post("/{groupImageID}/approve", h.ApproveImage) // Approve pending image
	r.Post("/{groupImageID}/reject", h.RejectImage)   // Reject pending image

	return r
}

// ShareImage handles POST /api/v1/groups/{groupID}/images
// Shares an image to the group's image pool.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Request: ShareImageRequest JSON body
// Response: 201 Created with GroupImageResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Not a group member
//   - 404: Group not found
//   - 409: Image already shared
//   - 500: Internal server error
func (h *GroupImageHandler) ShareImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in share image handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Decode request body
	var req ShareImageRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid share image request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid share image data",
			validationErrors,
		)
		return
	}

	// 4. Parse IDs
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	imageID, err := gallery.ParseImageID(req.ImageID)
	if err != nil {
		h.logger.Debug().Err(err).Str("image_id", req.ImageID).Msg("invalid image id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
		return
	}

	// 5. Build command
	cmd := commands.ShareImageToGroupCommand{
		GroupID: groupID,
		ImageID: imageID,
		ActorID: actorID,
	}

	// 6. Execute command
	result, err := h.shareImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "share image")
		return
	}

	// 7. Return response
	resp := GroupImageResponse{
		ID:       result.GroupImageID.String(),
		GroupID:  groupIDStr,
		ImageID:  req.ImageID,
		SharedBy: userCtx.UserID.String(),
		Status:   result.Status.String(),
	}

	h.logger.Info().
		Str("group_id", groupIDStr).
		Str("image_id", req.ImageID).
		Str("actor_id", userCtx.UserID.String()).
		Str("status", result.Status.String()).
		Msg("image shared to group successfully")

	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode share image response")
	}
}

// ListPendingImages handles GET /api/v1/groups/{groupID}/images/pending
// Lists images pending moderation (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//
// Query parameters:
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedGroupImagesResponse
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Not admin/owner
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupImageHandler) ListPendingImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (for authorization check in query handler)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in list pending images handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 3. Parse query parameters
	queryParams := r.URL.Query()

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

	// 4. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 5. Build query
	query := queries.ListPendingGroupImagesQuery{
		GroupID: groupID,
		ActorID: actorID,
		Page:    page,
		PerPage: perPage,
	}

	// 6. Execute query
	result, err := h.listPendingImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list pending images")
		return
	}

	// 7. Return paginated results
	resp := PaginatedGroupImagesResponse{
		Images:     mapGroupImagesToResponse(result.Images),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Str("group_id", groupIDStr).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Images)).
		Int("total", result.Total).
		Msg("pending images listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list pending images response")
	}
}

// ListApprovedImages handles GET /api/v1/groups/{groupID}/images
// Lists approved images in the group's image pool.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Query parameters:
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedGroupImagesResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupImageHandler) ListApprovedImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract group ID from path
	groupIDStr := GetPathParam(r, "groupID")
	if groupIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group ID",
		)
		return
	}

	groupID, err := community.ParseGroupID(groupIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_id", groupIDStr).Msg("invalid group id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group ID format",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

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

	// 3. Build query
	query := queries.ListApprovedGroupImagesQuery{
		GroupID: groupID,
		Page:    page,
		PerPage: perPage,
	}

	// 4. Execute query
	result, err := h.listApprovedImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list approved images")
		return
	}

	// 5. Return paginated results
	resp := PaginatedGroupImagesResponse{
		Images:     mapGroupImagesToResponse(result.Images),
		TotalCount: int64(result.Total),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.Total + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Str("group_id", groupIDStr).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Images)).
		Int("total", result.Total).
		Msg("approved images listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list approved images response")
	}
}

// ApproveImage handles POST /api/v1/groups/{groupID}/images/{groupImageID}/approve
// Approves a pending image in the moderation queue (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//   - groupImageID: UUID of the group image
//
// Response: 200 OK
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Not admin/owner
//   - 404: Group image not found
//   - 409: Image not in pending status
//   - 500: Internal server error
func (h *GroupImageHandler) ApproveImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in approve image handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group image ID from path
	groupImageIDStr := GetPathParam(r, "groupImageID")
	if groupImageIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group image ID",
		)
		return
	}

	groupImageID, err := community.ParseGroupImageID(groupImageIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_image_id", groupImageIDStr).Msg("invalid group image id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group image ID format",
		)
		return
	}

	// 3. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 4. Build command
	cmd := commands.ApproveGroupImageCommand{
		GroupImageID: groupImageID,
		ActorID:      actorID,
	}

	// 5. Execute command
	if err := h.approveImage.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "approve image")
		return
	}

	// 6. Return success
	h.logger.Info().
		Str("group_image_id", groupImageIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("image approved successfully")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"message":"Image approved successfully"}`)); err != nil {
		h.logger.Error().Err(err).Msg("failed to write approve image response")
	}
}

// RejectImage handles POST /api/v1/groups/{groupID}/images/{groupImageID}/reject
// Rejects a pending image in the moderation queue (admin+ only).
//
// Path parameters:
//   - groupID: UUID of the group
//   - groupImageID: UUID of the group image
//
// Request: RejectImageRequest JSON body (optional reason)
// Response: 200 OK
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Not admin/owner
//   - 404: Group image not found
//   - 409: Image not in pending status
//   - 500: Internal server error
func (h *GroupImageHandler) RejectImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in reject image handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract group image ID from path
	groupImageIDStr := GetPathParam(r, "groupImageID")
	if groupImageIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing group image ID",
		)
		return
	}

	groupImageID, err := community.ParseGroupImageID(groupImageIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("group_image_id", groupImageIDStr).Msg("invalid group image id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid group image ID format",
		)
		return
	}

	// 3. Decode optional request body
	var req RejectImageRequest
	_ = DecodeJSON(r, &req) // Ignore error - reason is optional

	// 4. Parse actor ID
	actorID, err := identity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().Err(err).Msg("invalid user id in context")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user context",
		)
		return
	}

	// 5. Build command
	cmd := commands.RejectGroupImageCommand{
		GroupImageID: groupImageID,
		ActorID:      actorID,
		Reason:       req.Reason,
	}

	// 6. Execute command
	if err := h.rejectImage.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "reject image")
		return
	}

	// 7. Return success
	h.logger.Info().
		Str("group_image_id", groupImageIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Str("reason", req.Reason).
		Msg("image rejected successfully")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"message":"Image rejected successfully"}`)); err != nil {
		h.logger.Error().Err(err).Msg("failed to write reject image response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses.
func (h *GroupImageHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("group image operation failed")

	switch {
	case err == community.ErrGroupNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Group not found",
		)
	case err == community.ErrGroupImageNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Group image not found",
		)
	case err == community.ErrNotGroupMember:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Must be a group member to share images",
		)
	case err == community.ErrImageAlreadyShared:
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Image is already shared to this group",
		)
	case err == community.ErrImageNotPending:
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Image is not in pending status",
		)
	case err == community.ErrInsufficientGroupRole:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Insufficient permissions for this operation",
		)
	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred",
		)
	}
}

// mapGroupImagesToResponse converts domain entities to response DTOs.
func mapGroupImagesToResponse(images []*community.GroupImage) []GroupImageResponse {
	resp := make([]GroupImageResponse, len(images))
	for i, img := range images {
		sharedAt := img.SharedAt()
		resp[i] = GroupImageResponse{
			ID:         img.ID().String(),
			GroupID:    img.GroupID().String(),
			ImageID:    img.ImageID().String(),
			SharedBy:   img.SharedBy().String(),
			Status:     img.Status().String(),
			SharedAt:   &sharedAt,
			ReviewedAt: img.ReviewedAt(),
		}
		if img.ReviewedBy() != nil {
			reviewedBy := img.ReviewedBy().String()
			resp[i].ReviewedBy = &reviewedBy
		}
	}
	return resp
}
