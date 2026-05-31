package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// FeaturedHandler handles featured picks management HTTP endpoints.
// These endpoints are admin-only for managing featured images on the explore page.
type FeaturedHandler struct {
	featureImage   *commands.FeatureImageHandler
	unfeatureImage *commands.UnfeatureImageHandler
	logger         zerolog.Logger
}

// NewFeaturedHandler creates a new FeaturedHandler with the given dependencies.
func NewFeaturedHandler(
	featureImage *commands.FeatureImageHandler,
	unfeatureImage *commands.UnfeatureImageHandler,
	logger zerolog.Logger,
) *FeaturedHandler {
	return &FeaturedHandler{
		featureImage:   featureImage,
		unfeatureImage: unfeatureImage,
		logger:         logger,
	}
}

// Routes registers featured picks management routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/moderation/featured
//
// All routes require admin authentication.
func (h *FeaturedHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.FeatureImage)
	r.Delete("/{imageID}", h.UnfeatureImage)

	return r
}

// FeatureImageRequest represents the request body for featuring an image.
type FeatureImageRequest struct {
	ImageID       string     `json:"image_id" validate:"required,uuid"`
	Reason        string     `json:"reason" validate:"omitempty,max=500"`
	DisplayOrder  int        `json:"display_order" validate:"gte=0"`
	FeaturedFrom  *time.Time `json:"featured_from"`
	FeaturedUntil *time.Time `json:"featured_until"`
}

// FeatureImage handles POST /api/v1/moderation/featured
// Adds an image to the featured picks list (admin only).
//
// Request body:
//   - image_id (string, required): UUID of the image to feature
//   - reason (string, optional): Reason/notes for featuring (max 500 chars)
//   - display_order (int, required): Display priority (0 = highest)
//   - featured_from (string, optional): Start date/time (default: now)
//   - featured_until (string, optional): End date/time (null = no expiration)
//
// Response: 201 Created with FeatureImageResult
// Errors:
//   - 400: Invalid request body or validation errors
//   - 401: Not authenticated
//   - 403: Not authorized (non-admin)
//   - 404: Image not found
//   - 409: Image already featured or not public/active
func (h *FeaturedHandler) FeatureImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract admin user ID from context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in feature image handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request body
	var req FeatureImageRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid feature image request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(
			w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid feature image request",
			validationErrors,
		)
		return
	}

	// 3. Build command
	cmd := commands.FeatureImageCommand{
		ImageID:       req.ImageID,
		FeaturedBy:    userCtx.UserID.String(),
		Reason:        req.Reason,
		DisplayOrder:  req.DisplayOrder,
		FeaturedFrom:  req.FeaturedFrom,
		FeaturedUntil: req.FeaturedUntil,
	}

	// 4. Execute command
	result, err := h.featureImage.Handle(ctx, cmd)
	if err != nil {
		h.mapFeatureError(w, r, err)
		return
	}

	// 5. Return response
	h.logger.Info().
		Str("pick_id", result.PickID).
		Str("image_id", result.ImageID).
		Str("admin_id", userCtx.UserID.String()).
		Int("display_order", result.DisplayOrder).
		Msg("image featured by admin")

	if err := EncodeJSON(w, http.StatusCreated, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode feature response")
	}
}

// UnfeatureImage handles DELETE /api/v1/moderation/featured/{imageID}
// Removes an image from the featured picks list (admin only).
//
// Path parameters:
//   - imageID: UUID of the image to unfeature
//
// Response: 204 No Content
// Errors:
//   - 400: Invalid image ID format
//   - 401: Not authenticated
//   - 403: Not authorized (non-admin)
//   - 404: Image not featured or doesn't exist
func (h *FeaturedHandler) UnfeatureImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract admin user ID from context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in unfeature image handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from path
	imageID := chi.URLParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	// 3. Build and execute command
	cmd := commands.UnfeatureImageCommand{
		ImageID: imageID,
	}

	if err := h.unfeatureImage.Handle(ctx, cmd); err != nil {
		h.mapFeatureError(w, r, err)
		return
	}

	// 4. Return 204 No Content
	h.logger.Info().
		Str("image_id", imageID).
		Str("admin_id", userCtx.UserID.String()).
		Msg("image unfeatured by admin")

	w.WriteHeader(http.StatusNoContent)
}

// mapFeatureError maps domain errors to HTTP responses.
func (h *FeaturedHandler) mapFeatureError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Msg("featured picks operation failed")

	switch {
	case errors.Is(err, gallery.ErrImageNotFound):
		middleware.WriteError(
			w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)

	case errors.Is(err, gallery.ErrFeaturedPickNotFound):
		middleware.WriteError(
			w, r,
			http.StatusNotFound,
			"Not Found",
			"Image is not currently featured",
		)

	case errors.Is(err, gallery.ErrImageAlreadyFeatured):
		middleware.WriteError(
			w, r,
			http.StatusConflict,
			"Conflict",
			"Image is already featured",
		)

	case errors.Is(err, gallery.ErrInvalidVisibility):
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Only public images can be featured",
		)

	case errors.Is(err, gallery.ErrInvalidImageStatus):
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Only active images can be featured",
		)

	case errors.Is(err, gallery.ErrReasonTooLong):
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Reason exceeds 500 characters",
		)

	default:
		middleware.WriteError(
			w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to process request",
		)
	}
}
