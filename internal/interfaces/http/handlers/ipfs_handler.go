package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// IPFSHandler handles IPFS-related HTTP endpoints.
// It delegates to application layer command and query handlers for business logic.
type IPFSHandler struct {
	pinImage      *commands.PinImageToIPFSHandler
	unpinImage    *commands.UnpinImageFromIPFSHandler
	getIPFSStatus *queries.GetImageIPFSStatusHandler
	logger        zerolog.Logger
}

// NewIPFSHandler creates a new IPFSHandler with the given dependencies.
func NewIPFSHandler(
	pinImage *commands.PinImageToIPFSHandler,
	unpinImage *commands.UnpinImageFromIPFSHandler,
	getIPFSStatus *queries.GetImageIPFSStatusHandler,
	logger zerolog.Logger,
) *IPFSHandler {
	return &IPFSHandler{
		pinImage:      pinImage,
		unpinImage:    unpinImage,
		getIPFSStatus: getIPFSStatus,
		logger:        logger,
	}
}

// Routes registers IPFS routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/images/{imageID}/ipfs
//
// Note: Authentication middleware should be applied at the router level.
func (h *IPFSHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// POST /api/v1/images/{imageID}/ipfs - Pin image to IPFS
	r.Post("/", h.Pin)

	// DELETE /api/v1/images/{imageID}/ipfs - Unpin image from IPFS
	r.Delete("/", h.Unpin)

	// GET /api/v1/images/{imageID}/ipfs - Get IPFS status
	r.Get("/", h.GetStatus)

	return r
}

// Pin handles POST /api/v1/images/{imageID}/ipfs
// Pins an image to IPFS for decentralized storage.
//
// Path Parameters:
//   - imageID: UUID of the image to pin
//
// Response: 201 Created with PinImageToIPFSResponse
// Errors:
//   - 400: Invalid image ID
//   - 401: Not authenticated
//   - 403: Not the image owner
//   - 404: Image not found
//   - 409: Image already pinned to IPFS
//   - 500: Internal server error (IPFS upload failed)
func (h *IPFSHandler) Pin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in IPFS pin handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from URL path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Image ID is required",
		)
		return
	}

	// 3. Execute pin command
	cmd := commands.PinImageToIPFSCommand{
		ImageID: imageID,
		UserID:  userCtx.UserID.String(),
	}

	result, err := h.pinImage.Handle(ctx, cmd)
	if err != nil {
		h.handlePinError(w, r, err)
		return
	}

	// 4. Return success response
	response := PinImageToIPFSResponse{
		ImageID:    result.ImageID,
		CID:        result.CID,
		GatewayURL: result.GatewayURL,
		PinnedAt:   result.PinnedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("cid", result.CID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image pinned to IPFS via API")

	if err := EncodeJSON(w, http.StatusCreated, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode IPFS pin response")
	}
}

// Unpin handles DELETE /api/v1/images/{imageID}/ipfs
// Unpins an image from IPFS, keeping only primary storage.
//
// Path Parameters:
//   - imageID: UUID of the image to unpin
//
// Response: 200 OK with UnpinImageFromIPFSResponse
// Errors:
//   - 400: Invalid image ID or image not on IPFS
//   - 401: Not authenticated
//   - 403: Not the image owner
//   - 404: Image not found
//   - 500: Internal server error (IPFS unpin failed)
func (h *IPFSHandler) Unpin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in IPFS unpin handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract image ID from URL path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Image ID is required",
		)
		return
	}

	// 3. Execute unpin command
	cmd := commands.UnpinImageFromIPFSCommand{
		ImageID: imageID,
		UserID:  userCtx.UserID.String(),
	}

	result, err := h.unpinImage.Handle(ctx, cmd)
	if err != nil {
		h.handleUnpinError(w, r, err)
		return
	}

	// 4. Return success response
	response := UnpinImageFromIPFSResponse{
		ImageID:     result.ImageID,
		UnpinnedCID: result.UnpinnedCID,
		Message:     result.Message,
	}

	h.logger.Info().
		Str("image_id", imageID).
		Str("unpinned_cid", result.UnpinnedCID).
		Str("user_id", userCtx.UserID.String()).
		Msg("image unpinned from IPFS via API")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode IPFS unpin response")
	}
}

// GetStatus handles GET /api/v1/images/{imageID}/ipfs
// Returns the IPFS status for an image.
//
// Path Parameters:
//   - imageID: UUID of the image to check
//
// Response: 200 OK with ImageIPFSStatusResponse
// Errors:
//   - 400: Invalid image ID
//   - 401: Not authenticated (for private images)
//   - 403: Unauthorized (private image, not owner)
//   - 404: Image not found
//   - 500: Internal server error
func (h *IPFSHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user ID from context (optional for public images)
	var userID string
	userCtx, err := GetUserFromContext(ctx)
	if err == nil {
		userID = userCtx.UserID.String()
	}

	// 2. Extract image ID from URL path
	imageID := GetPathParam(r, "imageID")
	if imageID == "" {
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Image ID is required",
		)
		return
	}

	// 3. Execute query
	query := queries.GetImageIPFSStatusQuery{
		ImageID:          imageID,
		RequestingUserID: userID,
	}

	result, err := h.getIPFSStatus.Handle(ctx, query)
	if err != nil {
		h.handleStatusError(w, r, err)
		return
	}

	// 4. Return success response
	response := ImageIPFSStatusResponse{
		ImageID:    result.ImageID,
		IsPinned:   result.IsPinned,
		CID:        result.CID,
		IPFSURI:    result.IPFSURI,
		GatewayURL: result.GatewayURL,
		PinnedAt:   result.PinnedAt,
	}

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode IPFS status response")
	}
}

// handlePinError converts application errors to appropriate HTTP responses.
func (h *IPFSHandler) handlePinError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gallery.ErrImageNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)
	case errors.Is(err, gallery.ErrUnauthorizedAccess):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to pin this image",
		)
	case containsString(err.Error(), "already pinned"):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			err.Error(),
		)
	case containsString(err.Error(), "invalid image id"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
	default:
		h.logger.Error().Err(err).Msg("unexpected error in IPFS pin")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to pin image to IPFS",
		)
	}
}

// handleUnpinError converts application errors to appropriate HTTP responses.
func (h *IPFSHandler) handleUnpinError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gallery.ErrImageNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)
	case errors.Is(err, gallery.ErrUnauthorizedAccess):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to unpin this image",
		)
	case containsString(err.Error(), "not pinned"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Image is not pinned to IPFS",
		)
	case containsString(err.Error(), "invalid image id"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
	default:
		h.logger.Error().Err(err).Msg("unexpected error in IPFS unpin")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to unpin image from IPFS",
		)
	}
}

// handleStatusError converts application errors to appropriate HTTP responses.
func (h *IPFSHandler) handleStatusError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gallery.ErrImageNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Image not found",
		)
	case errors.Is(err, gallery.ErrUnauthorizedAccess):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"You do not have permission to view this image's IPFS status",
		)
	case containsString(err.Error(), "invalid image id"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid image ID format",
		)
	default:
		h.logger.Error().Err(err).Msg("unexpected error in IPFS status")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to get IPFS status",
		)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

// findSubstring is a simple substring search.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Response DTOs for IPFS endpoints.

// PinImageToIPFSResponse is the response for a successful IPFS pin operation.
type PinImageToIPFSResponse struct {
	ImageID    string `json:"image_id"`
	CID        string `json:"cid"`
	GatewayURL string `json:"gateway_url"`
	PinnedAt   string `json:"pinned_at"`
}

// UnpinImageFromIPFSResponse is the response for a successful IPFS unpin operation.
type UnpinImageFromIPFSResponse struct {
	ImageID     string `json:"image_id"`
	UnpinnedCID string `json:"unpinned_cid"`
	Message     string `json:"message"`
}

// ImageIPFSStatusResponse is the response for an IPFS status query.
type ImageIPFSStatusResponse struct {
	ImageID    string  `json:"image_id"`
	IsPinned   bool    `json:"is_pinned"`
	CID        *string `json:"cid,omitempty"`
	IPFSURI    *string `json:"ipfs_uri,omitempty"`
	GatewayURL *string `json:"gateway_url,omitempty"`
	PinnedAt   *string `json:"pinned_at,omitempty"`
}
