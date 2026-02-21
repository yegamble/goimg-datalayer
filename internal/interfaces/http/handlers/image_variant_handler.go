package handlers

import (
	"net/http"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type GenerateCustomVariantRequest struct {
	ConfigID  string `json:"config_id,omitempty"`
	Name      string `json:"name,omitempty"`
	MaxWidth  int    `json:"max_width,omitempty"`
	MaxHeight int    `json:"max_height,omitempty"`
	Format    string `json:"format,omitempty"`
	Quality   int    `json:"quality,omitempty"`
	CropMode  string `json:"crop_mode,omitempty"`
}

func (h *ImageHandler) GenerateCustomVariant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing image ID",
		)
		return
	}

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

	if req.ConfigID == "" {
		if req.MaxWidth == 0 || req.MaxHeight == 0 {
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Validation Failed",
				"Either config_id or max_width/max_height are required",
			)
			return
		}
		if req.Format == "" {
			req.Format = "webp"
		}
		if req.Quality == 0 {
			req.Quality = 85
		}
		if req.CropMode == "" {
			req.CropMode = "fit"
		}
	}

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

	result, err := h.generateCustomVariant.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "generate custom variant")
		return
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("user_id", userCtx.UserID.String()).
		Str("variant_key", result.VariantKey).
		Msg("custom variant generated successfully")

	if err := EncodeJSON(w, http.StatusCreated, result); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode generate custom variant response")
	}
}
