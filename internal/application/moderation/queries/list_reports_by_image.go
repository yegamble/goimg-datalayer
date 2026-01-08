package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// ListReportsByImageQuery retrieves all reports for a specific image.
// This is useful for checking if an image has been reported multiple times.
type ListReportsByImageQuery struct {
	ImageID string // Image to get reports for
}

// Implement Query interface
func (ListReportsByImageQuery) isQuery() {}

// ListReportsByImageResult represents all reports for an image.
type ListReportsByImageResult struct {
	ImageID    string      `json:"image_id"`
	Reports    []ReportDTO `json:"reports"`
	TotalCount int         `json:"total_count"`
}

// ListReportsByImageHandler processes ListReportsByImageQuery requests.
// It retrieves all reports for a specific image.
type ListReportsByImageHandler struct {
	reports moderation.ReportRepository
	logger  *zerolog.Logger
}

// NewListReportsByImageHandler creates a new ListReportsByImageHandler with the given dependencies.
func NewListReportsByImageHandler(
	reports moderation.ReportRepository,
	logger *zerolog.Logger,
) *ListReportsByImageHandler {
	return &ListReportsByImageHandler{
		reports: reports,
		logger:  logger,
	}
}

// Handle executes the ListReportsByImageQuery and returns all reports for the image.
//
// Process flow:
//  1. Parse and validate image ID
//  2. Load all reports for the image
//  3. Convert to DTOs
//  4. Return with count
//
// Returns:
//   - *ListReportsByImageResult: All reports for the image
//   - Validation errors for invalid image ID
func (h *ListReportsByImageHandler) Handle(ctx context.Context, q ListReportsByImageQuery) (*ListReportsByImageResult, error) {
	// 1. Parse and validate image ID
	imageID, err := gallery.ParseImageID(q.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", q.ImageID).
			Msg("invalid image id during list reports by image")
		return nil, fmt.Errorf("invalid image id: %w", err)
	}

	// 2. Load all reports for the image
	reports, err := h.reports.FindByImage(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to list reports by image")
		return nil, fmt.Errorf("find reports by image: %w", err)
	}

	// 3. Convert to DTOs
	reportDTOs := make([]ReportDTO, 0, len(reports))
	for _, report := range reports {
		reportDTOs = append(reportDTOs, *reportToDTO(report))
	}

	// 4. Build result
	result := &ListReportsByImageResult{
		ImageID:    imageID.String(),
		Reports:    reportDTOs,
		TotalCount: len(reportDTOs),
	}

	h.logger.Debug().
		Str("image_id", imageID.String()).
		Int("total_count", len(reportDTOs)).
		Msg("reports by image listed successfully")

	return result, nil
}
