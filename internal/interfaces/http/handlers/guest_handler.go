package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// ClaimGuestImageRequest represents the JSON request body for claiming a guest image.
type ClaimGuestImageRequest struct {
	GuestUserID string `json:"guest_user_id"` // The guest user ID that uploaded the image
}

// ClaimGuestImageResponse represents the JSON response for a successful image claim.
type ClaimGuestImageResponse struct {
	ImageID       string `json:"image_id"`
	NewOwnerID    string `json:"new_owner_id"`
	PreviousOwner string `json:"previous_owner"`
	TransferredAt string `json:"transferred_at"`
	Message       string `json:"message"`
}

// GuestHandler handles guest-related HTTP endpoints.
// This includes claiming images uploaded during a guest session.
type GuestHandler struct {
	claimImage *commands.ClaimGuestImageHandler
	logger     zerolog.Logger
}

// NewGuestHandler creates a new GuestHandler with the given dependencies.
func NewGuestHandler(
	claimImage *commands.ClaimGuestImageHandler,
	logger zerolog.Logger,
) *GuestHandler {
	return &GuestHandler{
		claimImage: claimImage,
		logger:     logger,
	}
}

// Routes registers guest routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/guest
func (h *GuestHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Claim an image uploaded during guest session
	r.Post("/images/{imageID}/claim", h.ClaimImage)

	return r
}

// ClaimImage handles POST /api/v1/guest/images/{imageID}/claim
// Claims an image that was uploaded during a guest session and transfers
// ownership to the authenticated registered user.
//
// Request:
//   - Path: imageID - The ID of the image to claim
//   - Body: ClaimGuestImageRequest with guest_user_id
//   - Auth: Required (Bearer token)
//
// Response:
//   - 200 OK: ClaimGuestImageResponse on success
//   - 400 Bad Request: Invalid request body or IDs
//   - 401 Unauthorized: Not authenticated
//   - 403 Forbidden: Cannot claim this image (not guest owner, or requester is guest)
//   - 404 Not Found: Image or user not found
//
// Security:
//   - Only registered users can claim images
//   - Image must be owned by the specified guest user
//   - Guest user ownership is verified before transfer
func (h *GuestHandler) ClaimImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user ID
	userID := middleware.MustGetUserIDString(ctx)

	// Get image ID from URL
	imageID := chi.URLParam(r, "imageID")
	if imageID == "" {
		h.logger.Debug().Msg("missing image ID in claim request")
		middleware.WriteError(w, r, http.StatusBadRequest, "invalid_request", "Image ID is required")
		return
	}

	// Parse request body
	var req ClaimGuestImageRequest
	if err := DecodeJSONBody(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("failed to decode claim request body")
		middleware.WriteError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate guest user ID
	if req.GuestUserID == "" {
		h.logger.Debug().Msg("missing guest user ID in claim request")
		middleware.WriteError(w, r, http.StatusBadRequest, "invalid_request", "Guest user ID is required")
		return
	}

	// Execute command
	cmd := commands.ClaimGuestImageCommand{
		ImageID:     imageID,
		RequesterID: userID,
		GuestUserID: req.GuestUserID,
	}

	result, err := h.claimImage.Handle(ctx, cmd)
	if err != nil {
		h.handleClaimError(w, r, err)
		return
	}

	// Return success response
	resp := ClaimGuestImageResponse{
		ImageID:       result.ImageID,
		NewOwnerID:    result.NewOwnerID,
		PreviousOwner: result.PreviousOwner,
		TransferredAt: result.TransferredAt,
		Message:       result.Message,
	}

	if err := EncodeJSON(w, http.StatusOK, resp); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode claim response")
	}
}

// handleClaimError maps application errors to HTTP error responses.
func (h *GuestHandler) handleClaimError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gallery.ErrImageNotFound):
		middleware.WriteError(w, r, http.StatusNotFound, "not_found", "Image not found")
	case errors.Is(err, gallery.ErrUnauthorizedAccess):
		middleware.WriteError(w, r, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, gallery.ErrCannotModifyDeleted):
		middleware.WriteError(w, r, http.StatusConflict, "conflict", "Cannot claim deleted image")
	default:
		h.logger.Error().Err(err).Msg("unexpected error during image claim")
		middleware.WriteError(w, r, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
