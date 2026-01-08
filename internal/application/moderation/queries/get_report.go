package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// GetReportQuery retrieves a single report by its unique ID.
// This is a read-only operation with no side effects.
type GetReportQuery struct {
	ReportID string // Report to retrieve
}

// Implement Query interface
func (GetReportQuery) isQuery() {}

// ReportDTO represents a report data transfer object for HTTP responses.
type ReportDTO struct {
	ID          string     `json:"id"`
	ReporterID  string     `json:"reporter_id"`
	ImageID     string     `json:"image_id"`
	Reason      string     `json:"reason"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	ResolvedBy  *string    `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Resolution  string     `json:"resolution,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GetReportHandler processes GetReportQuery requests.
// It retrieves a single report and converts it to a DTO.
type GetReportHandler struct {
	reports moderation.ReportRepository
}

// NewGetReportHandler creates a new GetReportHandler with the given dependencies.
func NewGetReportHandler(reports moderation.ReportRepository) *GetReportHandler {
	return &GetReportHandler{
		reports: reports,
	}
}

// Handle executes the GetReportQuery and returns the report data.
//
// Returns:
//   - *ReportDTO: The report data
//   - error: ErrReportNotFound if the report does not exist, or other repository errors
func (h *GetReportHandler) Handle(ctx context.Context, q GetReportQuery) (*ReportDTO, error) {
	// Parse and validate report ID
	reportID, err := moderation.ParseReportID(q.ReportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report id: %w", err)
	}

	// Retrieve report from repository
	report, err := h.reports.FindByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("find report by id: %w", err)
	}

	// Convert to DTO
	return reportToDTO(report), nil
}

// reportToDTO converts a domain Report to a ReportDTO.
func reportToDTO(r *moderation.Report) *ReportDTO {
	dto := &ReportDTO{
		ID:          r.ID().String(),
		ReporterID:  r.ReporterID().String(),
		ImageID:     r.ImageID().String(),
		Reason:      r.Reason().String(),
		Description: r.Description(),
		Status:      r.Status().String(),
		Resolution:  r.Resolution(),
		CreatedAt:   r.CreatedAt(),
	}

	if r.ResolvedBy() != nil {
		resolvedByStr := r.ResolvedBy().String()
		dto.ResolvedBy = &resolvedByStr
	}

	if r.ResolvedAt() != nil {
		dto.ResolvedAt = r.ResolvedAt()
	}

	return dto
}
