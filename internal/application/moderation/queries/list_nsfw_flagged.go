package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListNSFWFlaggedQuery retrieves all scans that detected NSFW content.
// This is useful for moderation dashboards.
type ListNSFWFlaggedQuery struct {
	Page     int // Page number (1-based)
	PageSize int // Number of items per page
}

// Implement Query interface.
func (ListNSFWFlaggedQuery) isQuery() {}

// ListNSFWFlaggedResult represents the result of listing NSFW flagged scans.
type ListNSFWFlaggedResult struct {
	Scans      []*NSFWScanDTO `json:"scans"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int64          `json:"total_pages"`
}

// ListNSFWFlaggedHandler processes ListNSFWFlaggedQuery requests.
type ListNSFWFlaggedHandler struct {
	scans moderation.NSFWScanRepository
}

// NewListNSFWFlaggedHandler creates a new ListNSFWFlaggedHandler.
func NewListNSFWFlaggedHandler(scans moderation.NSFWScanRepository) *ListNSFWFlaggedHandler {
	return &ListNSFWFlaggedHandler{
		scans: scans,
	}
}

// Handle executes the ListNSFWFlaggedQuery and returns NSFW flagged scans.
//
// Returns:
//   - *ListNSFWFlaggedResult: The list of NSFW flagged scans with pagination
//   - error: Repository errors
func (h *ListNSFWFlaggedHandler) Handle(ctx context.Context, q ListNSFWFlaggedQuery) (*ListNSFWFlaggedResult, error) {
	// Create pagination.
	pagination, err := shared.NewPagination(q.Page, q.PageSize)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Retrieve NSFW flagged scans.
	scanList, totalCount, err := h.scans.FindNSFWImages(ctx, pagination)
	if err != nil {
		return nil, fmt.Errorf("find NSFW flagged scans: %w", err)
	}

	// Convert to DTOs.
	dtos := make([]*NSFWScanDTO, len(scanList))
	for i, scan := range scanList {
		dtos[i] = nsfwScanToDTO(scan)
	}

	// Calculate total pages.
	var totalPages int64
	if q.PageSize > 0 {
		totalPages = (totalCount + int64(q.PageSize) - 1) / int64(q.PageSize)
	}

	return &ListNSFWFlaggedResult{
		Scans:      dtos,
		TotalCount: totalCount,
		Page:       q.Page,
		PageSize:   q.PageSize,
		TotalPages: totalPages,
	}, nil
}
