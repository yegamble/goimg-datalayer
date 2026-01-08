package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// ListNSFWScansByImageQuery retrieves all NSFW scans for a specific image.
// This is a read-only operation with no side effects.
type ListNSFWScansByImageQuery struct {
	ImageID string // Image to get scans for
}

// Implement Query interface.
func (ListNSFWScansByImageQuery) isQuery() {}

// ListNSFWScansByImageResult represents the result of listing NSFW scans.
type ListNSFWScansByImageResult struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int            `json:"total_count"`
}

// ListNSFWScansByImageHandler processes ListNSFWScansByImageQuery requests.
type ListNSFWScansByImageHandler struct {
	scans moderation.NSFWScanRepository
}

// NewListNSFWScansByImageHandler creates a new ListNSFWScansByImageHandler.
func NewListNSFWScansByImageHandler(scans moderation.NSFWScanRepository) *ListNSFWScansByImageHandler {
	return &ListNSFWScansByImageHandler{
		scans: scans,
	}
}

// Handle executes the ListNSFWScansByImageQuery and returns all scans for an image.
//
// Returns:
//   - *ListNSFWScansByImageResult: The list of scans
//   - error: ErrImageNotFound if the image does not exist, or other repository errors
func (h *ListNSFWScansByImageHandler) Handle(
	ctx context.Context,
	q ListNSFWScansByImageQuery,
) (*ListNSFWScansByImageResult, error) {
	// Parse and validate image ID.
	imageID, err := gallery.ParseImageID(q.ImageID)
	if err != nil {
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// Retrieve all scans for the image.
	scanList, err := h.scans.FindByImageIDAll(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("find scans by image id: %w", err)
	}

	// Convert to DTOs.
	dtos := make([]*NSFWScanDTO, len(scanList))
	for i, scan := range scanList {
		dtos[i] = nsfwScanToDTO(scan)
	}

	return &ListNSFWScansByImageResult{
		Scans:      dtos,
		TotalCount: len(dtos),
	}, nil
}
