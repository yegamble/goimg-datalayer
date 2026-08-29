package handlers

import (
	"net/http"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

//nolint:dupl // Standard HTTP handler pattern - duplication is intentional for clarity
func (h *ImageHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	query := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: requestingUserID,
	}

	image, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get image")
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("requesting_user_id", requestingUserID).
		Msg("image retrieved successfully")

	if err := EncodeJSON(w, http.StatusOK, image); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode get image response")
	}
}

//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in update handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	var req UpdateImageRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid update image request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(
			w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid image update data",
			validationErrors,
		)
		return
	}

	cmd := commands.UpdateImageCommand{
		UserID:      userCtx.UserID.String(),
		ImageID:     imageID,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
		Tags:        req.Tags,
	}

	_, err = h.updateImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "update image")
		return
	}

	getQuery := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: userCtx.UserID.String(),
	}

	image, err := h.getImage.Handle(ctx, getQuery)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch updated image")
		middleware.WriteError(
			w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Image updated but failed to retrieve",
		)
		return
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image updated successfully")

	if err := EncodeJSON(w, http.StatusOK, image); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode update image response")
	}
}

func (h *ImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in delete handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(
			w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	cmd := commands.DeleteImageCommand{
		UserID:  userCtx.UserID.String(),
		ImageID: imageID,
	}

	_, err = h.deleteImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "delete image")
		return
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}
