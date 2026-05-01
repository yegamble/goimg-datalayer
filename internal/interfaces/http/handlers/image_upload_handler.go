package handlers

import (
	"io"
	"net/http"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	maxUploadSizeMB = 50
	megabyteShift   = 20
)

//nolint:funlen // HTTP handler with validation and response.
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	if !userCtx.EmailVerified {
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Email verification required to upload images",
		)
		return
	}

	// #nosec G120 // Max upload size is bounded and configurable
	if err := r.ParseMultipartForm(maxUploadSizeMB << megabyteShift); err != nil {
		h.logger.Debug().Err(err).Msg("failed to parse multipart form")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid multipart form data",
		)
		return
	}

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

	fileSize := header.Size
	filename := header.Filename

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
		visibility = "private"
	}

	tags := r.Form["tags"]

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
		Width:       0,
		Height:      0,
	}

	result, err := h.uploadImage.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "upload image")
		return
	}

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
