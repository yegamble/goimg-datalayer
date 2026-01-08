package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	appgallery "github.com/yegamble/goimg-datalayer/internal/application/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetImageIPFSStatusQuery retrieves the IPFS status for an image.
// This includes the CID, pin status, and gateway URLs.
type GetImageIPFSStatusQuery struct {
	ImageID          string // ID of the image to check
	RequestingUserID string // ID of the user requesting (must be owner or image public)
}

// ImageIPFSStatusDTO represents the IPFS status of an image.
type ImageIPFSStatusDTO struct {
	ImageID    string  `json:"image_id"`
	IsPinned   bool    `json:"is_pinned"`
	CID        *string `json:"cid,omitempty"`         // nil if not on IPFS
	IPFSURI    *string `json:"ipfs_uri,omitempty"`    // ipfs://CID format
	GatewayURL *string `json:"gateway_url,omitempty"` // Public gateway URL
	PinnedAt   *string `json:"pinned_at,omitempty"`   // When pinned (RFC 3339)
}

// GetImageIPFSStatusHandler processes GetImageIPFSStatusQuery requests.
// It retrieves the IPFS status for an image with authorization checks.
type GetImageIPFSStatusHandler struct {
	images gallery.ImageRepository
	ipfs   appgallery.IPFSService
	logger *zerolog.Logger
}

// NewGetImageIPFSStatusHandler creates a new GetImageIPFSStatusHandler.
func NewGetImageIPFSStatusHandler(
	images gallery.ImageRepository,
	ipfs appgallery.IPFSService,
	logger *zerolog.Logger,
) *GetImageIPFSStatusHandler {
	return &GetImageIPFSStatusHandler{
		images: images,
		ipfs:   ipfs,
		logger: logger,
	}
}

// Handle executes the GetImageIPFSStatusQuery and returns the IPFS status.
//
// Process flow:
//  1. Parse and validate image ID
//  2. Load image from repository
//  3. Check authorization (owner or public image)
//  4. Extract IPFS metadata if available
//  5. Optionally verify pin status with IPFS node
//  6. Return DTO with IPFS details
//
// Returns:
//   - *ImageIPFSStatusDTO: The IPFS status
//   - ErrImageNotFound: If the image doesn't exist
//   - ErrUnauthorizedAccess: If user lacks permission to view
func (h *GetImageIPFSStatusHandler) Handle(ctx context.Context, q GetImageIPFSStatusQuery) (*ImageIPFSStatusDTO, error) {
	// 1. Parse and validate image ID
	imageID, err := gallery.ParseImageID(q.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", q.ImageID).
			Msg("invalid image id during IPFS status check")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// Parse requesting user ID (optional)
	var requestingUserID identity.UserID
	if q.RequestingUserID != "" {
		requestingUserID, err = identity.ParseUserID(q.RequestingUserID)
		if err != nil {
			h.logger.Debug().
				Err(err).
				Str("requesting_user_id", q.RequestingUserID).
				Msg("invalid requesting user id")
			return nil, fmt.Errorf("invalid requesting user id: %w", err)
		}
	}

	// 2. Load image from repository
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		if errors.Is(err, gallery.ErrImageNotFound) {
			h.logger.Debug().
				Str("image_id", imageID.String()).
				Msg("image not found during IPFS status check")
			return nil, fmt.Errorf("find image: %w", err)
		}
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to load image for IPFS status check")
		return nil, fmt.Errorf("find image: %w", err)
	}

	// 3. Check authorization
	if err := h.checkAuthorization(image, requestingUserID); err != nil {
		return nil, err
	}

	// 4. Build response DTO
	dto := &ImageIPFSStatusDTO{
		ImageID:  imageID.String(),
		IsPinned: image.HasIPFS(),
	}

	// 5. If image has IPFS metadata, include details
	if image.HasIPFS() {
		meta := image.IPFSMetadata()
		cid := meta.CID()
		ipfsURI := meta.URI()
		gatewayURL := h.ipfs.GatewayURL(cid)

		dto.CID = &cid
		dto.IPFSURI = &ipfsURI
		dto.GatewayURL = &gatewayURL

		if pinnedAt := meta.PinnedAt(); pinnedAt != nil {
			formatted := pinnedAt.Format("2006-01-02T15:04:05Z07:00")
			dto.PinnedAt = &formatted
		}
	}

	h.logger.Debug().
		Str("image_id", imageID.String()).
		Bool("is_pinned", dto.IsPinned).
		Msg("IPFS status retrieved successfully")

	return dto, nil
}

// checkAuthorization verifies that the requesting user can view the IPFS status.
// Users can view IPFS status if they own the image or if the image is public.
func (h *GetImageIPFSStatusHandler) checkAuthorization(
	image *gallery.Image,
	requestingUserID identity.UserID,
) error {
	// Owner can always see IPFS status
	if !requestingUserID.IsZero() && image.IsOwnedBy(requestingUserID) {
		return nil
	}

	// Public images allow IPFS status to be viewed
	if image.Visibility().IsPublic() && image.IsViewable() {
		return nil
	}

	h.logger.Debug().
		Str("image_id", image.ID().String()).
		Str("visibility", image.Visibility().String()).
		Msg("unauthorized IPFS status check")
	return gallery.ErrUnauthorizedAccess
}
