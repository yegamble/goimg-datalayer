package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// AlbumHandler handles album-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type AlbumHandler struct {
	createAlbum          *commands.CreateAlbumHandler
	updateAlbum          *commands.UpdateAlbumHandler
	deleteAlbum          *commands.DeleteAlbumHandler
	addImageToAlbum      *commands.AddImageToAlbumHandler
	removeImageFromAlbum *commands.RemoveImageFromAlbumHandler
	getAlbum             *queries.GetAlbumHandler
	listAlbums           *queries.ListAlbumsHandler
	listAlbumImages      *queries.ListAlbumImagesHandler
	logger               zerolog.Logger
}

// NewAlbumHandler creates a new AlbumHandler with the given dependencies.
func NewAlbumHandler(
	createAlbum *commands.CreateAlbumHandler,
	updateAlbum *commands.UpdateAlbumHandler,
	deleteAlbum *commands.DeleteAlbumHandler,
	addImageToAlbum *commands.AddImageToAlbumHandler,
	removeImageFromAlbum *commands.RemoveImageFromAlbumHandler,
	getAlbum *queries.GetAlbumHandler,
	listAlbums *queries.ListAlbumsHandler,
	listAlbumImages *queries.ListAlbumImagesHandler,
	logger zerolog.Logger,
) *AlbumHandler {
	return &AlbumHandler{
		createAlbum:          createAlbum,
		updateAlbum:          updateAlbum,
		deleteAlbum:          deleteAlbum,
		addImageToAlbum:      addImageToAlbum,
		removeImageFromAlbum: removeImageFromAlbum,
		getAlbum:             getAlbum,
		listAlbums:           listAlbums,
		listAlbumImages:      listAlbumImages,
		logger:               logger,
	}
}

// Routes registers album routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/albums
func (h *AlbumHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{albumID}", h.Get)
	r.Put("/{albumID}", h.Update)
	r.Delete("/{albumID}", h.Delete)
	r.Post("/{albumID}/images", h.AddImage)
	r.Delete("/{albumID}/images/{imageID}", h.RemoveImage)
	r.Get("/{albumID}/images", h.ListImages)

	return r
}

// Create handles POST /api/v1/albums
// Creates a new album.
//
// Request: CreateAlbumRequest JSON body
// Response: 201 Created with AlbumResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *AlbumHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	// 2. Decode request body
	var req CreateAlbumRequest
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

	// 3. Build create command
	cmd := commands.CreateAlbumCommand{
		UserID:      userCtx.UserID.String(),
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
	}

	// 4. Execute create command
	albumID, err := h.createAlbum.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "create album")
		return
	}

	// 5. Fetch created album to return
	query := queries.GetAlbumQuery{
		AlbumID:          albumID,
		RequestingUserID: userCtx.UserID.String(),
	}

	album, err := h.getAlbum.Handle(ctx, query)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch created album")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Album created but failed to retrieve",
		)
		return
	}

	// 6. Return created album
	h.logger.Info().
		Str("album_id", albumID).
		Str("user_id", userCtx.UserID.String()).
		Msg("album created successfully")

	if err := EncodeJSON(w, http.StatusCreated, album); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode create album response")
	}
}

// Get handles GET /api/v1/albums/{albumID}
// Retrieves a single album by its ID.
//
// Path parameters:
//   - albumID: UUID of the album
//
// Response: 200 OK with AlbumResponse
// Errors:
//   - 400: Invalid album ID format
//   - 403: Album is private and user is not the owner
//   - 404: Album not found
//   - 500: Internal server error
func (h *AlbumHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract album ID from path
	albumID := GetPathParam(r, "albumID")
	if albumID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	// 2. Extract requesting user ID (optional)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	// 3. Build query
	query := queries.GetAlbumQuery{
		AlbumID:          albumID,
		RequestingUserID: requestingUserID,
	}

	// 4. Execute query
	album, err := h.getAlbum.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get album")
		return
	}

	// 5. Return album
	h.logger.Debug().
		Str("album_id", albumID).
		Str("requesting_user_id", requestingUserID).
		Msg("album retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, album); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get album response")
	}
}

// Update handles PUT /api/v1/albums/{albumID}
// Updates album metadata (title, description, visibility).
//
// Path parameters:
//   - albumID: UUID of the album
//
// Request: UpdateAlbumRequest JSON body
// Response: 200 OK with AlbumResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: User is not the owner
//   - 404: Album not found
//   - 500: Internal server error
func (h *AlbumHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	albumID := GetPathParam(r, "albumID")
	if albumID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	// 3. Decode request body
	var req UpdateAlbumRequest
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

	// 4. Build update command
	cmd := commands.UpdateAlbumCommand{
		UserID:      userCtx.UserID.String(),
		AlbumID:     albumID,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
		// Note: CoverImageID not supported in current command structure
	}

	// 5. Execute update command
	if err := h.updateAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "update album")
		return
	}

	// 6. Fetch updated album to return
	query := queries.GetAlbumQuery{
		AlbumID:          albumID,
		RequestingUserID: userCtx.UserID.String(),
	}

	album, err := h.getAlbum.Handle(ctx, query)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch updated album")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Album updated but failed to retrieve",
		)
		return
	}

	// 7. Return updated album
	h.logger.Info().
		Str("album_id", albumID).
		Str("user_id", userCtx.UserID.String()).
		Msg("album updated successfully")

	if err := EncodeJSON(w, http.StatusOK, album); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update album response")
	}
}

// Delete handles DELETE /api/v1/albums/{albumID}
// Deletes an album (but not the images in it).
//
// Path parameters:
//   - albumID: UUID of the album
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid album ID format
//   - 401: Not authenticated
//   - 403: User is not the owner
//   - 404: Album not found
//   - 500: Internal server error
func (h *AlbumHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	albumID := GetPathParam(r, "albumID")
	if albumID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	// 3. Build delete command
	cmd := commands.DeleteAlbumCommand{
		UserID:  userCtx.UserID.String(),
		AlbumID: albumID,
	}

	// 4. Execute delete command
	if err := h.deleteAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "delete album")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("album_id", albumID).
		Str("user_id", userCtx.UserID.String()).
		Msg("album deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/albums
// Lists albums with filtering and pagination.
//
// Query parameters:
//   - owner_id: Filter by owner user ID (optional)
//   - visibility: Filter by visibility: public, private, unlisted (optional)
//   - offset: Pagination offset (default: 0)
//   - limit: Page size, max 100 (default: 20)
//
// Response: 200 OK with PaginatedAlbumsResponse
// Errors:
//   - 400: Invalid query parameters
//   - 500: Internal server error
func (h *AlbumHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	queryParams := r.URL.Query()

	ownerID := queryParams.Get("owner_id")
	visibility := queryParams.Get("visibility")

	offset, err := parseIntParam(queryParams.Get("offset"), 0)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid offset parameter",
		)
		return
	}

	limit, err := parseIntParam(queryParams.Get("limit"), 20)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid limit parameter",
		)
		return
	}

	if limit > 100 {
		limit = 100
	}

	// 2. Extract requesting user ID (for authorization - currently unused but may be needed for filtering)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}
	_ = requestingUserID // TODO: Use for authorization filtering if needed

	// 3. Convert offset/limit to page/perPage
	page := (offset / limit) + 1
	if page < 1 {
		page = 1
	}

	// 4. Build list query
	query := queries.ListAlbumsQuery{
		OwnerUserID: ownerID,
		Visibility:  visibility,
		Page:        page,
		PerPage:     limit,
	}

	// 5. Execute query
	result, err := h.listAlbums.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list albums")
		return
	}

	// 6. Convert query result DTOs to handler DTOs
	albumDTOs := make([]AlbumDTO, len(result.Albums))
	for i, album := range result.Albums {
		albumDTOs[i] = AlbumDTO{
			ID:           album.ID,
			OwnerID:      album.OwnerID,
			Title:        album.Title,
			Description:  album.Description,
			Visibility:   album.Visibility,
			CoverImageID: album.CoverImageID,
			ImageCount:   album.ImageCount,
			CreatedAt:    album.CreatedAt,
			UpdatedAt:    album.UpdatedAt,
		}
	}

	// 7. Calculate hasMore
	hasMore := int64(offset+limit) < result.TotalCount

	// 8. Return paginated results
	h.logger.Debug().
		Str("owner_id", ownerID).
		Int("offset", offset).
		Int("limit", limit).
		Int("results", len(albumDTOs)).
		Int64("total_count", result.TotalCount).
		Msg("albums listed successfully")

	response := PaginatedAlbumsResponse{
		Albums:     albumDTOs,
		TotalCount: result.TotalCount,
		Offset:     offset,
		Limit:      limit,
		HasMore:    hasMore,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list albums response")
	}
}

// AddImage handles POST /api/v1/albums/{albumID}/images
// Adds an image to an album.
//
// Path parameters:
//   - albumID: UUID of the album
//
// Request: AddImageToAlbumRequest JSON body
// Response: 200 OK with success message
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: User doesn't own the album or image
//   - 404: Album or image not found
//   - 409: Image already in album
//   - 500: Internal server error
func (h *AlbumHandler) AddImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in add image handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract album ID from path
	albumID := GetPathParam(r, "albumID")
	if albumID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	// 3. Decode request body
	var req AddImageToAlbumRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid add image to album request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid image data",
			validationErrors,
		)
		return
	}

	// 4. Build command
	cmd := commands.AddImageToAlbumCommand{
		AlbumID: albumID,
		ImageID: req.ImageID,
		UserID:  userCtx.UserID.String(),
	}

	// 5. Execute command
	if err := h.addImageToAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "add image to album")
		return
	}

	// 6. Return success message
	h.logger.Info().
		Str("album_id", albumID).
		Str("image_id", req.ImageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image added to album successfully")

	response := map[string]string{
		"message": "Image added to album successfully",
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode add image response")
	}
}

// RemoveImage handles DELETE /api/v1/albums/{albumID}/images/{imageID}
// Removes an image from an album.
//
// Path parameters:
//   - albumID: UUID of the album
//   - imageID: UUID of the image to remove
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid album or image ID
//   - 401: Not authenticated
//   - 403: User doesn't own the album
//   - 404: Album or image not found
//   - 500: Internal server error
func (h *AlbumHandler) RemoveImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in remove image handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract album ID and image ID from path
	albumID := GetPathParam(r, "albumID")
	imageID := GetPathParam(r, "imageID")

	if albumID == "" || imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID or image ID",
		)
		return
	}

	// 3. Build command
	cmd := commands.RemoveImageFromAlbumCommand{
		AlbumID: albumID,
		ImageID: imageID,
		UserID:  userCtx.UserID.String(),
	}

	// 4. Execute command
	if err := h.removeImageFromAlbum.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "remove image from album")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("album_id", albumID).
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image removed from album successfully")

	w.WriteHeader(http.StatusNoContent)
}

// ListImages handles GET /api/v1/albums/{albumID}/images
// Lists all images in an album with pagination.
//
// Path parameters:
//   - albumID: UUID of the album
//
// Query parameters:
//   - offset: Pagination offset (default: 0)
//   - limit: Page size, max 100 (default: 20)
//
// Response: 200 OK with PaginatedImagesResponse
// Errors:
//   - 400: Invalid parameters
//   - 403: Album is private and user is not the owner
//   - 404: Album not found
//   - 500: Internal server error
func (h *AlbumHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract album ID from path
	albumID := GetPathParam(r, "albumID")
	if albumID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing album ID",
		)
		return
	}

	// 2. Parse query parameters
	queryParams := r.URL.Query()

	offset, err := parseIntParam(queryParams.Get("offset"), 0)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid offset parameter",
		)
		return
	}

	limit, err := parseIntParam(queryParams.Get("limit"), 20)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid limit parameter",
		)
		return
	}

	if limit > 100 {
		limit = 100
	}

	// 3. Extract requesting user ID (optional)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	// 4. Convert offset/limit to page/perPage
	page := (offset / limit) + 1
	if page < 1 {
		page = 1
	}

	// 5. Build query
	query := queries.ListAlbumImagesQuery{
		AlbumID:          albumID,
		RequestingUserID: requestingUserID,
		Page:             page,
		PerPage:          limit,
	}

	// 6. Execute query
	result, err := h.listAlbumImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list album images")
		return
	}

	// 7. Convert query result image DTOs to handler DTOs
	imageDTOs := make([]ImageDTO, len(result.Images))
	for i, img := range result.Images {
		imageDTOs[i] = ImageDTO{
			ID:               img.ID,
			OwnerID:          img.OwnerID,
			Title:            img.Title,
			Description:      img.Description,
			OriginalFilename: img.OriginalFilename,
			MimeType:         img.MimeType,
			Width:            img.Width,
			Height:           img.Height,
			FileSize:         img.FileSize,
			StorageKey:       img.StorageKey,
			StorageProvider:  img.StorageProvider,
			Visibility:       img.Visibility,
			Status:           img.Status,
			Variants:         convertVariantDTOs(img.Variants),
			Tags:             convertTagDTOs(img.Tags),
			ViewCount:        img.ViewCount,
			LikeCount:        img.LikeCount,
			CommentCount:     img.CommentCount,
			CreatedAt:        img.CreatedAt,
			UpdatedAt:        img.UpdatedAt,
		}
	}

	// 8. Calculate hasMore
	hasMore := int64(offset+limit) < result.TotalCount

	// 9. Return paginated results
	h.logger.Debug().
		Str("album_id", albumID).
		Int("offset", offset).
		Int("limit", limit).
		Int("results", len(imageDTOs)).
		Int64("total_count", result.TotalCount).
		Msg("album images listed successfully")

	response := PaginatedImagesResponse{
		Images:     imageDTOs,
		TotalCount: result.TotalCount,
		Offset:     offset,
		Limit:      limit,
		HasMore:    hasMore,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode list album images response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses.
func (h *AlbumHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("album operation failed")

	// Simplified error mapping - expand based on actual domain errors
	middleware.WriteError(w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}

// convertVariantDTOs converts query layer variant DTOs to handler layer DTOs.
func convertVariantDTOs(variants []queries.VariantDTO) []VariantDTO {
	result := make([]VariantDTO, len(variants))
	for i, v := range variants {
		result[i] = VariantDTO{
			Type:       v.Type,
			StorageKey: v.StorageKey,
			Width:      v.Width,
			Height:     v.Height,
			FileSize:   v.FileSize,
			Format:     v.Format,
		}
	}
	return result
}

// convertTagDTOs converts query layer tag DTOs to handler layer DTOs.
func convertTagDTOs(tags []queries.TagDTO) []TagDTO {
	result := make([]TagDTO, len(tags))
	for i, t := range tags {
		result[i] = TagDTO{
			Name: t.Name,
			Slug: t.Slug,
		}
	}
	return result
}
