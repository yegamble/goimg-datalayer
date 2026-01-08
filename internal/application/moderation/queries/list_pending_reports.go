package queries

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListPendingReportsQuery retrieves a paginated list of pending reports.
// This is used for the moderator queue to review reports.
type ListPendingReportsQuery struct {
	Page    int // Page number (1-indexed)
	PerPage int // Items per page (max 100)
}

// Implement Query interface
func (ListPendingReportsQuery) isQuery() {}

// ListPendingReportsResult represents the paginated result of pending reports.
type ListPendingReportsResult struct {
	Reports    []ReportDTO `json:"reports"`
	TotalCount int64       `json:"total_count"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// ListPendingReportsHandler processes ListPendingReportsQuery requests.
// It retrieves pending reports with pagination for the moderator queue.
type ListPendingReportsHandler struct {
	reports moderation.ReportRepository
	logger  *zerolog.Logger
}

// NewListPendingReportsHandler creates a new ListPendingReportsHandler with the given dependencies.
func NewListPendingReportsHandler(
	reports moderation.ReportRepository,
	logger *zerolog.Logger,
) *ListPendingReportsHandler {
	return &ListPendingReportsHandler{
		reports: reports,
		logger:  logger,
	}
}

// Handle executes the ListPendingReportsQuery and returns paginated results.
//
// Process flow:
//  1. Validate and normalize pagination parameters
//  2. Load pending reports from repository
//  3. Convert to DTOs
//  4. Return with pagination metadata
//
// Returns:
//   - *ListPendingReportsResult: Paginated list of pending reports
//   - Validation errors for invalid parameters
func (h *ListPendingReportsHandler) Handle(ctx context.Context, q ListPendingReportsQuery) (*ListPendingReportsResult, error) {
	// 1. Validate and normalize pagination
	page := q.Page
	if page < 1 {
		page = 1
	}

	perPage := q.PerPage
	if perPage <= 0 {
		perPage = shared.DefaultPerPage
	}
	if perPage > shared.MaxPerPage {
		perPage = shared.MaxPerPage
	}

	pagination, err := shared.NewPagination(page, perPage)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Int("page", page).
			Int("per_page", perPage).
			Msg("invalid pagination parameters")
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// 2. Load pending reports from repository
	reports, totalCount, err := h.reports.FindPending(ctx, pagination)
	if err != nil {
		h.logger.Error().
			Err(err).
			Int("page", page).
			Int("per_page", perPage).
			Msg("failed to list pending reports")
		return nil, fmt.Errorf("find pending reports: %w", err)
	}

	// 3. Convert to DTOs
	reportDTOs := make([]ReportDTO, 0, len(reports))
	for _, report := range reports {
		reportDTOs = append(reportDTOs, *reportToDTO(report))
	}

	// 4. Build result with pagination metadata
	paginationWithTotal := pagination.WithTotal(totalCount)
	result := &ListPendingReportsResult{
		Reports:    reportDTOs,
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
		TotalPages: paginationWithTotal.TotalPages(),
		HasNext:    paginationWithTotal.HasNext(),
		HasPrev:    paginationWithTotal.HasPrev(),
	}

	h.logger.Debug().
		Int("page", page).
		Int("per_page", perPage).
		Int("results", len(reportDTOs)).
		Int64("total_count", totalCount).
		Msg("pending reports listed successfully")

	return result, nil
}
