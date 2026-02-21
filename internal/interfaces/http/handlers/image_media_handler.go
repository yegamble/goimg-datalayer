package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

//nolint:funlen,cyclop // HTTP handler with variant parsing, authorization, and storage.
func (h *ImageHandler) GetImageVariant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	sizeParam := GetPathParam(r, "size")
	if sizeParam == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing variant size",
		)
		return
	}

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

	var requestingUserID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	query := queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: requestingUserID,
	}

	imageDTO, err := h.getImage.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "get image for variant")
		return
	}

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

	contentType := formatToMimeType(variantDTO.Format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(variantDTO.FileSize, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, fileReader); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("variant_type", variantType.String()).
			Msg("failed to stream variant data")
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("variant_type", variantType.String()).
		Str("content_type", contentType).
		Int64("file_size", variantDTO.FileSize).
		Msg("image variant retrieved successfully")
}

func (h *ImageHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

	var requestingUserID string
	if userCtx, err := GetUserFromContext(ctx); err == nil {
		requestingUserID = userCtx.UserID.String()
	}

	imageDTO, err := h.getImage.Handle(ctx, queries.GetImageQuery{
		ImageID:          imageID,
		RequestingUserID: requestingUserID,
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
			h.mapErrorAndRespond(w, r, err, "download image")
		}
		return
	}

	fileReader, err := h.storage.Get(ctx, imageDTO.StorageKey)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Str("storage_key", imageDTO.StorageKey).
			Msg("failed to retrieve image from storage for download")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve image",
		)
		return
	}
	defer func() {
		if cerr := fileReader.Close(); cerr != nil {
			h.logger.Warn().Err(cerr).
				Str("image_id", imageID).
				Msg("failed to close image reader after download")
		}
	}()

	filename := imageDTO.OriginalFilename
	if filename == "" {
		filename = imageID
	}

	contentType := imageDTO.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(imageDTO.FileSize, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "private, no-store")

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, fileReader); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID).
			Msg("failed to stream image data for download")
		return
	}

	h.logger.Debug().
		Str("image_id", imageID).
		Str("filename", filename).
		Str("content_type", contentType).
		Int64("file_size", imageDTO.FileSize).
		Msg("image downloaded successfully")
}

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
