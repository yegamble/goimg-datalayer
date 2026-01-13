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
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// GroupAlbumHandler handles group album HTTP endpoints.
// It manages album CRUD operations and image-album associations within groups.
type GroupAlbumHandler struct {
	createAlbum          *commands.CreateGroupAlbumHandler
	updateAlbum          *commands.UpdateGroupAlbumHandler
	deleteAlbum          *commands.DeleteGroupAlbumHandler
	getAlbum             *queries.GetGroupAlbumHandler
	listAlbums           *queries.ListGroupAlbumsHandler
	addImageToAlbum      *commands.AddImageToGroupAlbumHandler
	removeImageFromAlbum *commands.RemoveImageFromGroupAlbumHandler
	logger               zerolog.Logger
}

// NewGroupAlbumHandler creates a new GroupAlbumHandler with the given dependencies.
func NewGroupAlbumHandler(
	createAlbum *commands.CreateGroupAlbumHandler,
	updateAlbum *commands.UpdateGroupAlbumHandler,
	deleteAlbum *commands.DeleteGroupAlbumHandler,
	getAlbum *queries.GetGroupAlbumHandler,
	listAlbums *queries.ListGroupAlbumsHandler,
	addImageToAlbum *commands.AddImageToGroupAlbumHandler,
	removeImageFromAlbum *commands.RemoveImageFromGroupAlbumHandler,
	logger zerolog.Logger,
) *GroupAlbumHandler {
	return &GroupAlbumHandler{
		createAlbum:          createAlbum,
		updateAlbum:          updateAlbum,
		deleteAlbum:          deleteAlbum,
		getAlbum:             getAlbum,
		listAlbums:           listAlbums,
		addImageToAlbum:      addImageToAlbum,
		removeImageFromAlbum: removeImageFromAlbum,
		logger:               logger,
	}
}

// Routes returns group album routes that require authentication.
// These should be mounted under /api/v1/groups/{groupID}/albums.
func (h *GroupAlbumHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Album CRUD routes
	r.Post("/", h.CreateAlbum)            // Create album
	r.Get("/", h.ListAlbums)              // List albums
	r.Get("/{albumID}", h.GetAlbum)       // Get album
	r.Put("/{albumID}", h.UpdateAlbum)    // Update album
	r.Delete("/{albumID}", h.DeleteAlbum) // Delete album

	// Album image management routes
	r.Post("/{albumID}/images", h.AddImageToAlbum)                  // Add image to album
	r.Delete("/{albumID}/images/{imageID}", h.RemoveImageFromAlbum) // Remove image from album

	return r
}

// CreateAlbum handles POST /api/v1/groups/{groupID}/albums
// Creates a new album within the group.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Request: CreateGroupAlbumRequest JSON body
// Response: 201 Created with GroupAlbumResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Not a group member
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in create album handler")
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
	var req CreateGroupAlbumRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid create album request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid album data",
			validationErrors,
		)
		return
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

	// 5. Build create command
	cmd := commands.CreateGroupAlbumCommand{
		GroupID:  groupID,
		ActorID:  actorID,
		Title:    req.Title,
		IsPublic: req.IsPublic,
	}

	// 6. Execute create command
	album, err := h.createAlbum.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create album")
		return
	}

	// 7. Map to response DTO
	resp := mapGroupAlbumToResponse(album)

	h.logger.Info().
		Str("album_id", album.ID().String()).
		Str("group_id", groupIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("group album created successfully")

	if err := EncodeJSON(w, http.StatusCreated, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create album response")
	}
}

// ListAlbums handles GET /api/v1/groups/{groupID}/albums
// Lists albums for a specific group with pagination.
//
// Path parameters:
//   - groupID: UUID of the group
//
// Query parameters:
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with PaginatedGroupAlbumsResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: Group not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
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

	// 3. Get optional actor ID (for authorization in private groups)
	var actorID *identity.UserID
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		id, err := identity.ParseUserID(userCtx.UserID.String())
		if err == nil {
			actorID = &id
		}
	}

	// 4. Build list query
	query := queries.ListGroupAlbumsQuery{
		GroupID:    groupID,
		ActorID:    actorID,
		Pagination: shared.NewPagination(page, perPage),
	}

	// 5. Execute query
	result, err := h.listAlbums.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list albums")
		return
	}

	// 6. Map to response
	resp := PaginatedGroupAlbumsResponse{
		Albums:     mapGroupAlbumsToResponse(result.Albums),
		TotalCount: int64(result.TotalCount),
		Page:       page,
		PerPage:    perPage,
		TotalPages: int64((result.TotalCount + perPage - 1) / perPage),
	}

	h.logger.Debug().
		Str("group_id", groupIDStr).
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(result.Albums)).
		Int("total", result.TotalCount).
		Msg("group albums listed successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list albums response")
	}
}

// GetAlbum handles GET /api/v1/groups/{groupID}/albums/{albumID}
// Retrieves a single album by its ID.
//
// Path parameters:
//   - groupID: UUID of the group
//   - albumID: UUID of the album
//
// Response: 200 OK with GroupAlbumResponse
// Errors:
//   - 400: Invalid parameters
//   - 404: Album or group not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract album ID from path
	albumIDStr := GetPathParam(r, "albumID")
	if albumIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	albumID, err := community.ParseGroupAlbumID(albumIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("album_id", albumIDStr).Msg("invalid album id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid album ID format",
		)
		return
	}

	// 2. Build query
	query := queries.GetGroupAlbumQuery{
		AlbumID: albumID,
	}

	// 3. Execute query
	album, err := h.getAlbum.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get album")
		return
	}

	// 4. Map to response
	resp := mapGroupAlbumToResponse(album)

	h.logger.Debug().
		Str("album_id", albumIDStr).
		Msg("group album retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get album response")
	}
}

// UpdateAlbum handles PUT /api/v1/groups/{groupID}/albums/{albumID}
// Updates album metadata and settings.
//
// Path parameters:
//   - groupID: UUID of the group
//   - albumID: UUID of the album
//
// Request: UpdateGroupAlbumRequest JSON body
// Response: 200 OK with GroupAlbumResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Insufficient permissions
//   - 404: Album or group not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update album handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract album ID from path
	albumIDStr := GetPathParam(r, "albumID")
	if albumIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	albumID, err := community.ParseGroupAlbumID(albumIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("album_id", albumIDStr).Msg("invalid album id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid album ID format",
		)
		return
	}

	// 3. Decode request body
	var req UpdateGroupAlbumRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update album request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid album update data",
			validationErrors,
		)
		return
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

	// 5. Parse optional cover image ID
	var coverImageID *gallery.ImageID
	if req.CoverImageID != nil && *req.CoverImageID != "" {
		imageID, err := gallery.ParseImageID(*req.CoverImageID)
		if err != nil {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid cover image ID format",
			)
			return
		}
		coverImageID = &imageID
	}

	// 6. Build update command
	cmd := commands.UpdateGroupAlbumCommand{
		AlbumID:      albumID,
		ActorID:      actorID,
		Title:        req.Title,
		Description:  req.Description,
		CoverImageID: coverImageID,
	}

	// 7. Execute update command
	album, err := h.updateAlbum.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update album")
		return
	}

	// 8. Map to response
	resp := mapGroupAlbumToResponse(album)

	h.logger.Info().
		Str("album_id", albumIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("group album updated successfully")

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update album response")
	}
}

// DeleteAlbum handles DELETE /api/v1/groups/{groupID}/albums/{albumID}
// Deletes an album.
//
// Path parameters:
//   - groupID: UUID of the group
//   - albumID: UUID of the album
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions
//   - 404: Album or group not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete album handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract album ID from path
	albumIDStr := GetPathParam(r, "albumID")
	if albumIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	albumID, err := community.ParseGroupAlbumID(albumIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("album_id", albumIDStr).Msg("invalid album id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid album ID format",
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

	// 4. Build delete command
	cmd := commands.DeleteGroupAlbumCommand{
		AlbumID: albumID,
		ActorID: actorID,
	}

	// 5. Execute delete command
	if err := h.deleteAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "delete album")
		return
	}

	// 6. Return 204 No Content
	h.logger.Info().
		Str("album_id", albumIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("group album deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// AddImageToAlbum handles POST /api/v1/groups/{groupID}/albums/{albumID}/images
// Adds an image to a group album.
//
// Path parameters:
//   - groupID: UUID of the group
//   - albumID: UUID of the album
//
// Request: AddImageToGroupAlbumRequest JSON body
// Response: 204 No Content
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: Not a group member or image not in group pool
//   - 404: Album or image not found
//   - 409: Image already in album
//   - 500: Internal server error
func (h *GroupAlbumHandler) AddImageToAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in add image to album handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract album ID from path
	albumIDStr := GetPathParam(r, "albumID")
	if albumIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	albumID, err := community.ParseGroupAlbumID(albumIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("album_id", albumIDStr).Msg("invalid album id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid album ID format",
		)
		return
	}

	// 3. Decode request body
	var req AddImageToGroupAlbumRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid add image to album request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
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
	cmd := commands.AddImageToGroupAlbumCommand{
		AlbumID: albumID,
		ImageID: imageID,
		ActorID: actorID,
	}

	// 6. Execute command
	if err := h.addImageToAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "add image to album")
		return
	}

	// 7. Return 204 No Content
	h.logger.Info().
		Str("album_id", albumIDStr).
		Str("image_id", req.ImageID).
		Str("actor_id", userCtx.UserID.String()).
		Msg("image added to group album successfully")

	w.WriteHeader(http.StatusNoContent)
}

// RemoveImageFromAlbum handles DELETE /api/v1/groups/{groupID}/albums/{albumID}/images/{imageID}
// Removes an image from a group album.
//
// Path parameters:
//   - groupID: UUID of the group
//   - albumID: UUID of the album
//   - imageID: UUID of the image
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid parameters
//   - 401: Not authenticated
//   - 403: Insufficient permissions
//   - 404: Album or image not found
//   - 500: Internal server error
func (h *GroupAlbumHandler) RemoveImageFromAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in remove image from album handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract IDs from path
	albumIDStr := GetPathParam(r, "albumID")
	if albumIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	imageIDStr := GetPathParam(r, "imageID")
	if imageIDStr == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	albumID, err := community.ParseGroupAlbumID(albumIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("album_id", albumIDStr).Msg("invalid album id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid album ID format",
		)
		return
	}

	imageID, err := gallery.ParseImageID(imageIDStr)
	if err != nil {
		h.logger.Debug().Err(err).Str("image_id", imageIDStr).Msg("invalid image id")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
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
	cmd := commands.RemoveImageFromGroupAlbumCommand{
		AlbumID: albumID,
		ImageID: imageID,
		ActorID: actorID,
	}

	// 5. Execute command
	if err := h.removeImageFromAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "remove image from album")
		return
	}

	// 6. Return 204 No Content
	h.logger.Info().
		Str("album_id", albumIDStr).
		Str("image_id", imageIDStr).
		Str("actor_id", userCtx.UserID.String()).
		Msg("image removed from group album successfully")

	w.WriteHeader(http.StatusNoContent)
}

// mapErrorAndRespond maps application/domain errors to HTTP responses.
func (h *GroupAlbumHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("group album operation failed")

	switch {
	case err == community.ErrGroupNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Group not found",
		)
	case err == community.ErrGroupAlbumNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Group album not found",
		)
	case err == community.ErrNotGroupMember:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Must be a group member to perform this action",
		)
	case err == community.ErrInsufficientGroupRole:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Insufficient permissions for this operation",
		)
	case err == community.ErrGroupImageNotFound:
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found in group pool",
		)
	case err == community.ErrImageAlreadyInAlbum:
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Image is already in this album",
		)
	case err == community.ErrPrivateGroupNoAccess:
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Cannot access private group without membership",
		)
	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred",
		)
	}
}

// mapGroupAlbumToResponse converts a GroupAlbum domain entity to a response DTO.
func mapGroupAlbumToResponse(album *community.GroupAlbum) GroupAlbumResponse {
	resp := GroupAlbumResponse{
		ID:          album.ID().String(),
		GroupID:     album.GroupID().String(),
		CreatedBy:   album.CreatedBy().String(),
		Title:       album.Title(),
		Description: album.Description(),
		ImageCount:  album.ImageCount(),
		IsPublic:    album.IsPublic(),
		CreatedAt:   album.CreatedAt(),
		UpdatedAt:   album.UpdatedAt(),
	}

	if coverID := album.CoverImageID(); coverID != nil {
		coverIDStr := coverID.String()
		resp.CoverImageID = &coverIDStr
	}

	return resp
}

// mapGroupAlbumsToResponse converts multiple GroupAlbum entities to response DTOs.
func mapGroupAlbumsToResponse(albums []*community.GroupAlbum) []GroupAlbumResponse {
	resp := make([]GroupAlbumResponse, len(albums))
	for i, album := range albums {
		resp[i] = mapGroupAlbumToResponse(album)
	}
	return resp
}
