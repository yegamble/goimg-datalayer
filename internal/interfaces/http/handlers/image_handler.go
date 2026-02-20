package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	// maxUploadSizeMB is the maximum allowed multipart form size in megabytes.
	maxUploadSizeMB = 50
	// megabyteShift is the bit shift to convert megabytes to bytes (1 MB = 1 << 20 bytes).
	megabyteShift = 20
	// defaultQRCodeSize is the default output size for QR code images in pixels.
	defaultQRCodeSize = 256
	// minQRCodeSize is the minimum supported QR code size in pixels.
	minQRCodeSize = 128
	// maxQRCodeSize is the maximum supported QR code size in pixels.
	maxQRCodeSize = 1024
)

// ImageHandler handles image-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type ImageHandler struct {
	uploadImage           *commands.UploadImageHandler
	updateImage           *commands.UpdateImageHandler
	deleteImage           *commands.DeleteImageHandler
	generateCustomVariant *commands.GenerateCustomVariantHandler
	getImage              *queries.GetImageHandler
	listImages            *queries.ListImagesHandler
	searchImages          *queries.SearchImagesHandler
	storage               StorageProvider
	baseURL               string
	logger                zerolog.Logger
}

// StorageProvider is the interface for retrieving image files from storage.
// This interface is defined here to avoid importing infrastructure packages in tests.
type StorageProvider interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// NewImageHandler creates a new ImageHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewImageHandler(
	uploadImage *commands.UploadImageHandler,
	updateImage *commands.UpdateImageHandler,
	deleteImage *commands.DeleteImageHandler,
	generateCustomVariant *commands.GenerateCustomVariantHandler,
	getImage *queries.GetImageHandler,
	listImages *queries.ListImagesHandler,
	searchImages *queries.SearchImagesHandler,
	storage StorageProvider,
	baseURL string,
	logger zerolog.Logger,
) *ImageHandler {
	return &ImageHandler{
		uploadImage:           uploadImage,
		updateImage:           updateImage,
		deleteImage:           deleteImage,
		generateCustomVariant: generateCustomVariant,
		getImage:              getImage,
		listImages:            listImages,
		searchImages:          searchImages,
		storage:               storage,
		baseURL:               strings.TrimRight(baseURL, "/"),
		logger:                logger,
	}
}

// Routes registers image routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/images
//
// Note: Authentication and rate limiting middleware should be applied
// at the router level before mounting these routes.
//
// Note: The variant endpoint (/{imageID}/variants/{size}) is registered
// separately in router.go with optional authentication.
func (h *ImageHandler) Routes(rateLimiterConfig *middleware.RateLimiterConfig) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication - applied at router level
	// Upload route has special rate limiting
	if rateLimiterConfig != nil {
		r.With(middleware.UploadRateLimiter(*rateLimiterConfig)).Post("/", h.Upload)
	} else {
		r.Post("/", h.Upload)
	}
	r.Get("/", h.List)
	r.Get("/search", h.Search)
	r.Get("/{imageID}", h.Get)
	r.Put("/{imageID}", h.Update)
	r.Delete("/{imageID}", h.Delete)

	// Custom variant generation (Sprint 17)
	if h.generateCustomVariant != nil {
		r.Post("/{imageID}/variants", h.GenerateCustomVariant)
	}

	return r
}

// Upload handles POST /api/v1/images
// Uploads a new image with multipart/form-data
//
// Request: multipart/form-data with fields:
//   - image (file): The image file (required)
//   - title (string): Image title (required)
//   - description (string): Image description (optional)
//   - visibility (string): public, private, unlisted (default: private)
//   - tags (array): Tag names (optional)
//
// Response: 202 Accepted with UploadImageResponse
// Errors:
//   - 400: Invalid request data or file
//   - 401: Not authenticated
//   - 413: File too large
//   - 415: Unsupported media type
//   - 429: Upload rate limit exceeded
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from context (set by JWT middleware)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in upload handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Parse multipart form (limit to configured max size in memory)
	if err := r.ParseMultipartForm(maxUploadSizeMB << megabyteShift); err != nil {
		h.logger.Debug().Err(err).Msg("failed to parse multipart form")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid multipart form data",
		)
		return
	}

	// 3. Extract image file from form
	file, header, err := r.FormFile("image")
	if err != nil {
		h.logger.Debug().Err(err).Msg("image file not found in form")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Image file is required",
		)
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			h.logger.Warn().Err(cerr).Msg("failed to close uploaded file")
		}
	}()

	// 4. Get file metadata
	fileSize := header.Size
	filename := header.Filename

	// Security: Detect MIME type from content instead of trusting header
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		h.logger.Error().Err(err).Msg("failed to read file header for mime detection")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to process image file",
		)
		return
	}

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		h.logger.Error().Err(err).Msg("failed to reset file pointer")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to process image file",
		)
		return
	}

	mimeType := http.DetectContentType(buffer[:n])
	h.logger.Debug().
		Str("detected_mime", mimeType).
		Str("header_mime", header.Header.Get("Content-Type")).
		Msg("mime type detection")

	// 5. Extract form fields
	title := r.FormValue("title")
	if title == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Title is required",
		)
		return
	}

	description := r.FormValue("description")
	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "private" // Default
	}

	// Parse tags (comma-separated or multiple values)
	tags := r.Form["tags"]

	// 6. Build upload command
	cmd := commands.UploadImageCommand{
		UserID:      userCtx.UserID.String(),
		FileContent: file,
		FileSize:    fileSize,
		Filename:    filename,
		Title:       title,
		Description: description,
		Visibility:  visibility,
		Tags:        tags,
		MimeType:    mimeType,
		// Width and Height will be extracted during processing
		Width:  0,
		Height: 0,
	}

	// 7. Execute upload command
	result, err := h.uploadImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "upload image")
		return
	}

	// 8. Return 202 Accepted with image ID and status
	h.logger.Info().
		Str("image_id", result.ImageID).
		Str("user_id", userCtx.UserID.String()).
		Str("filename", filename).
		Int64("size", fileSize).
		Msg("image uploaded successfully")

	response := UploadImageResponse{
		ID:      result.ImageID,
		Status:  result.Status,
		Message: result.Message,
	}

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode upload response")
	}
}

// Get handles GET /api/v1/images/{imageID}
// Retrieves a single image by its ID.
//
// Path parameters:
//   - imageID: UUID of the image
//
// Response: 200 OK with ImageResponse
// Errors:
//   - 400: Invalid image ID format
//   - 403: Image is private and user is not the owner
//   - 404: Image not found
//   - 500: Internal server error
//
//nolint:dupl // Standard HTTP handler pattern - duplication is intentional for clarity
func (h *ImageHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 2. Extract requesting user ID (optional - for authorization)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	// 3. Build query
	query := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: requestingUserID,
	}

	// 4. Execute query
	image, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get image")
		return
	}

	// 5. Return image DTO
	h.logger.Debug().
		Str("image_id", imageID).
		Str("requesting_user_id", requestingUserID).
		Msg("image retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, image); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get image response")
	}
}

// Update handles PUT /api/v1/images/{imageID}
// Updates image metadata (title, description, visibility, tags).
//
// Path parameters:
//   - imageID: UUID of the image
//
// Request: UpdateImageRequest JSON body
// Response: 200 OK with ImageResponse
// Errors:
//   - 400: Invalid request data
//   - 401: Not authenticated
//   - 403: User is not the owner
//   - 404: Image not found
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Decode request body
	var req UpdateImageRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update image request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid image update data",
			validationErrors,
		)
		return
	}

	// 4. Build update command
	cmd := commands.UpdateImageCommand{
		UserID:      userCtx.UserID.String(),
		ImageID:     imageID,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
		Tags:        req.Tags,
	}

	// 5. Execute update command
	_, err = h.updateImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update image")
		return
	}

	// 6. Fetch updated image to return
	query := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: userCtx.UserID.String(),
	}

	image, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch updated image")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Image updated but failed to retrieve",
		)
		return
	}

	// 7. Return updated image
	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image updated successfully")

	if err := EncodeJSON(w, http.StatusOK, image); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update image response")
	}
}

// Delete handles DELETE /api/v1/images/{imageID}
// Deletes an image and all its variants from storage.
//
// Path parameters:
//   - imageID: UUID of the image
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid image ID format
//   - 401: Not authenticated
//   - 403: User is not the owner
//   - 404: Image not found
//   - 500: Internal server error
func (h *ImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Build delete command
	cmd := commands.DeleteImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	// 4. Execute delete command
	_, err = h.deleteImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "delete image")
		return
	}

	// 5. Return 204 No Content
	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/images
// Lists images with filtering and pagination.
//
// Query parameters:
//   - owner_id: Filter by owner user ID (optional)
//   - visibility: Filter by visibility: public, private, unlisted (optional)
//   - tag: Filter by tag slug (optional)
//   - sort_by: Sort field: created_at, updated_at, view_count, like_count (default: created_at)
//   - sort_order: Sort direction: asc, desc (default: desc)
//   - offset: Pagination offset (default: 0)
//   - limit: Page size, max 100 (default: 20)
//
// Response: 200 OK with PaginatedImagesResponse
// Errors:
//   - 400: Invalid query parameters
//   - 500: Internal server error
//
//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	queryParams := r.URL.Query()

	ownerID := queryParams.Get("owner_id")
	visibility := queryParams.Get("visibility")
	tag := queryParams.Get("tag")
	sortBy := queryParams.Get("sort_by")
	sortOrder := queryParams.Get("sort_order")

	offset, err := parseIntParam(queryParams.Get("offset"), 0)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid offset parameter",
		)
		return
	}

	limit, err := parseIntParam(queryParams.Get("limit"), defaultPerPage)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid limit parameter",
		)
		return
	}

	if limit > maxPerPage {
		limit = maxPerPage // Max page size
	}

	// 2. Extract requesting user ID (for authorization)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	// 3. Build list query
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

	// 4. Execute query
	result, err := h.listImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list images")
		return
	}

	// 5. Return paginated results
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

// Search handles GET /api/v1/images/search
// Full-text search for images with filtering.
//
// Query parameters:
//   - q: Search query (title and description)
//   - tags: Tag filters (comma-separated, AND logic)
//   - owner_id: Filter by owner (optional)
//   - visibility: Filter by visibility (default: public)
//   - sort_by: Sort field: relevance, created_at, view_count, like_count (default: relevance)
//   - page: Page number, 1-indexed (default: 1)
//   - per_page: Items per page, max 100 (default: 20)
//
// Response: 200 OK with SearchImagesResponse
// Errors:
//   - 400: Invalid query parameters
//   - 500: Internal server error
//
//nolint:funlen,cyclop // HTTP handler with multiple query parameters and validation.
func (h *ImageHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse query parameters
	queryParams := r.URL.Query()

	searchQuery := queryParams.Get("q")
	if searchQuery == "" {
		middleware.WriteError(w, r,
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

	// Parse tags (comma-separated)
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

	// 2. Build search query
	query := queries.SearchImagesQuery{
		Query:      searchQuery,
		Tags:       tags,
		OwnerID:    ownerID,
		Visibility: visibility,
		SortBy:     sortBy,
		Page:       page,
		PerPage:    perPage,
	}

	// 3. Execute search
	result, err := h.searchImages.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "search images")
		return
	}

	// 4. Return search results
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

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
func (h *ImageHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("image operation failed")

	// Map to HTTP status codes
	// This is a simplified version - expand based on actual domain errors
	middleware.WriteError(w, r,
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
}

// GetImageVariant handles GET /api/v1/images/{imageID}/variants/{size}
// Retrieves a specific image variant (resized version) and returns binary data.
//
// Path parameters:
//   - imageID: UUID of the image
//   - size: Variant size (thumbnail, small, medium, large, original)
//
// Security: Optional auth - respects image visibility rules (public images accessible to all, private only to owner)
//
// Response: Binary image data with appropriate Content-Type
// Errors:
//   - 400: Invalid image ID or variant size
//   - 403: Image is private and user is not the owner
//   - 404: Image or variant not found
//   - 500: Internal server error
//
// and content type handling
//
//nolint:funlen,cyclop // HTTP handler with variant parsing, authorization, and storage.
func (h *ImageHandler) GetImageVariant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 2. Extract variant size from path
	sizeParam := GetPathParam(r, "size")
	if sizeParam == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing variant size",
		)
		return
	}

	// 3. Parse variant type
	variantType, err := gallery.ParseVariantType(sizeParam)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("size", sizeParam).
			Msg("invalid variant size")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid variant size. Must be one of: thumbnail, small, medium, large, original",
		)
		return
	}

	// 4. Extract requesting user ID (optional - for authorization)
	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	// 5. Get image metadata using existing query handler
	// This enforces visibility rules (public images accessible to all, private only to owner)
	query := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: requestingUserID,
	}

	imageDTO, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get image for variant")
		return
	}

	// 6. Find the requested variant
	var variantDTO *queries.VariantDTO
	for i := range imageDTO.Variants {
		if imageDTO.Variants[i].Type == variantType.String() {
			variantDTO = &imageDTO.Variants[i]
			break
		}
	}

	if variantDTO == nil {
		h.logger.Debug().
			Str("image_id", imageID).
			Str("variant_type", variantType.String()).
			Msg("variant not found")
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image variant not found",
		)
		return
	}

	// 7. Retrieve variant file from storage
	fileReader, err := h.storage.Get(ctx, variantDTO.StorageKey)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("variant_type", variantType.String()).
			Str("storage_key", variantDTO.StorageKey).
			Msg("failed to retrieve variant from storage")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve image variant",
		)
		return
	}
	defer func() {
		if cerr := fileReader.Close(); cerr != nil {
			h.logger.Warn().Err(cerr).
				Str("image_id", imageID).
				Str("variant_type", variantType.String()).
				Msg("failed to close variant file reader")
		}
	}()

	// 8. Set Content-Type header based on variant format
	contentType := formatToMimeType(variantDTO.Format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(variantDTO.FileSize, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	// 9. Stream binary data to response
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, fileReader); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("variant_type", variantType.String()).
			Msg("failed to stream variant data")
		// Can't send error response after writing to body
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("variant_type", variantType.String()).
		Str("content_type", contentType).
		Int64("file_size", variantDTO.FileSize).
		Msg("image variant retrieved successfully")
}

// GetImageQRCode handles GET /api/v1/images/{imageID}/qr
// Generates a PNG QR code for the public image preview URL.
//
// Query parameters:
//   - size: QR code image size in pixels (optional, default 256, min 128, max 1024)
//
// Security: Optional auth - only public images are shareable via QR code.
//
// Response: Binary PNG image
// Errors:
//   - 400: Invalid image ID or size
//   - 404: Image not found or not publicly shareable
//   - 500: QR code generation failure
func (h *ImageHandler) GetImageQRCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate image ID.
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	if _, err := gallery.ParseImageID(imageID); err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
		return
	}

	// 2. Parse optional size parameter.
	qrSize, err := parseIntParam(r.URL.Query().Get("size"), defaultQRCodeSize)
	if err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid size parameter",
		)
		return
	}
	if qrSize < minQRCodeSize || qrSize > maxQRCodeSize {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("size must be between %d and %d", minQRCodeSize, maxQRCodeSize),
		)
		return
	}

	// 3. Load image and verify shareability.
	// Use anonymous lookup semantics so QR generation never increments view counts.
	image, err := h.getImage.Handle(ctx, queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: "",
	})
	if err != nil {
		switch {
		case errors.Is(err, gallery.ErrImageNotFound), errors.Is(err, gallery.ErrUnauthorizedAccess):
			middleware.WriteError(w, r,
				http.StatusNotFound,
				"Not Found",
				"Image not found",
			)
		default:
			h.logger.Error().
				Err(err).
				Str("image_id", imageID).
				Msg("failed to load image for QR code generation")
			middleware.WriteError(w, r,
				http.StatusInternalServerError,
				"Internal Server Error",
				"Failed to load image",
			)
		}
		return
	}

	// QR is intended for anonymous sharing via preview page.
	if image.Visibility != gallery.VisibilityPublic.String() {
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)
		return
	}

	// 4. Generate QR for preview URL.
	baseURL := h.baseURL
	if baseURL == "" {
		baseURL = inferBaseURLFromRequest(r)
	}
	previewURL := fmt.Sprintf("%s/images/%s/preview", baseURL, image.ID)

	pngData, err := generateQRCodePNG(previewURL, qrSize)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("preview_url", previewURL).
			Int("size", qrSize).
			Msg("failed to generate image QR code")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to generate QR code",
		)
		return
	}

	// 5. Return PNG binary.
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(pngData)))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"image-%s-qr.png\"", image.ID))
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(pngData); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Msg("failed to write QR code response")
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("preview_url", previewURL).
		Int("size", qrSize).
		Msg("image QR code generated successfully")
}

// GenerateCustomVariantRequest represents the request body for generating a custom variant.
type GenerateCustomVariantRequest struct {
	ConfigID  string `json:"config_id,omitempty"`  // Use a saved variant config
	Name      string `json:"name,omitempty"`       // Custom variant name
	MaxWidth  int    `json:"max_width,omitempty"`  // Required if config_id not provided
	MaxHeight int    `json:"max_height,omitempty"` // Required if config_id not provided
	Format    string `json:"format,omitempty"`     // jpeg, png, webp, avif
	Quality   int    `json:"quality,omitempty"`    // 1-100
	CropMode  string `json:"crop_mode,omitempty"`  // fit, fill, crop
}

// GenerateCustomVariant handles POST /api/v1/images/{imageID}/variants
// Generates a custom image variant using user-defined parameters or a saved config.
//
// Path parameters:
//   - imageID: UUID of the image
//
// Request body: GenerateCustomVariantRequest JSON
// Response: 201 Created with CustomVariantResult
// Errors:
//   - 400: Invalid request data or parameters
//   - 401: Not authenticated
//   - 403: User doesn't own the image
//   - 404: Image not found
//   - 500: Processing error
func (h *ImageHandler) GenerateCustomVariant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in generate custom variant handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Decode request body
	var req GenerateCustomVariantRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid generate custom variant request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid variant parameters",
			validationErrors,
		)
		return
	}

	// 4. Validate request - either config_id or direct parameters required
	if req.ConfigID == "" {
		// Validate direct parameters
		if req.MaxWidth == 0 || req.MaxHeight == 0 {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Validation Failed",
				"Either config_id or max_width/max_height are required",
			)
			return
		}
		if req.Format == "" {
			req.Format = "webp" // Default format
		}
		if req.Quality == 0 {
			req.Quality = 85 // Default quality
		}
		if req.CropMode == "" {
			req.CropMode = "fit" // Default crop mode
		}
	}

	// 5. Build command
	cmd := commands.GenerateCustomVariantCommand{
		ImageID:   imageID,
		UserID:    userCtx.UserID.String(),
		ConfigID:  req.ConfigID,
		Name:      req.Name,
		MaxWidth:  req.MaxWidth,
		MaxHeight: req.MaxHeight,
		Format:    req.Format,
		Quality:   req.Quality,
		CropMode:  req.CropMode,
	}

	// 6. Execute command
	result, err := h.generateCustomVariant.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "generate custom variant")
		return
	}

	// 7. Return result
	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Str("variant_key", result.VariantKey).
		Msg("custom variant generated successfully")

	if err := EncodeJSON(w, http.StatusCreated, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode generate custom variant response")
	}
}

// formatToMimeType converts a format string (jpeg, png, gif, webp) to a MIME type.
func formatToMimeType(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// generateQRCodePNG renders a QR code as PNG bytes at the requested square size.
func generateQRCodePNG(content string, size int) ([]byte, error) {
	qrCode, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("encode qr: %w", err)
	}

	scaled, err := barcode.Scale(qrCode, size, size)
	if err != nil {
		return nil, fmt.Errorf("scale qr: %w", err)
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, scaled); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return buffer.Bytes(), nil
}

// inferBaseURLFromRequest builds an absolute base URL from request metadata.
func inferBaseURLFromRequest(r *http.Request) string {
	// Prefer forwarded headers when behind reverse proxies/load balancers.
	proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = r.Host
	}

	return fmt.Sprintf("%s://%s", proto, host)
}

// parseIntParam parses an integer query parameter with a default value.
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
