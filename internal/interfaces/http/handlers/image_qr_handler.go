package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	defaultQRCodeSize = 256
	minQRCodeSize     = 128
	maxQRCodeSize     = 1024
)

func (h *ImageHandler) GetImageQRCode(w http.ResponseWriter, r *http.Request) {
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

	if _, err := gallery.ParseImageID(imageID); err != nil {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
		return
	}

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

	if image.Visibility != gallery.VisibilityPublic.String() {
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)
		return
	}

	baseURL := h.baseURL
	if baseURL == "" {
		h.logger.Error().Msg("baseURL is not configured, cannot generate QR code with absolute URL")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Server configuration error",
		)
		return
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

