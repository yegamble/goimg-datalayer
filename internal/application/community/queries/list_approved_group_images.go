package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListApprovedGroupImagesQuery retrieves approved images for a group.
// This is the main image pool visible to members.
type ListApprovedGroupImagesQuery struct {
	GroupID community.GroupID
	Page    int // Page number (1-indexed)
	PerPage int // Items per page
}

// Implement Query interface.
func (ListApprovedGroupImagesQuery) isQuery() {}

// ListApprovedGroupImagesResult contains the paginated result with total count.
type ListApprovedGroupImagesResult struct {
	Images     []*community.GroupImage
	Total      int
	Pagination shared.Pagination
}

// ListApprovedGroupImagesHandler processes ListApprovedGroupImagesQuery requests.
type ListApprovedGroupImagesHandler struct {
	groupImageRepo community.GroupImageRepository
}

// NewListApprovedGroupImagesHandler creates a new handler with the given dependencies.
func NewListApprovedGroupImagesHandler(
	groupImageRepo community.GroupImageRepository,
) *ListApprovedGroupImagesHandler {
	return &ListApprovedGroupImagesHandler{
		groupImageRepo: groupImageRepo,
	}
}

// Handle executes the ListApprovedGroupImagesQuery and returns paginated results.
//
// Returns:
//   - *ListApprovedGroupImagesResult: Approved images with pagination metadata
//   - error: Repository errors
func (h *ListApprovedGroupImagesHandler) Handle(ctx context.Context, q ListApprovedGroupImagesQuery) (*ListApprovedGroupImagesResult, error) {
	// 1. Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// 2. Query repository
	images, total, err := h.groupImageRepo.FindApprovedByGroup(ctx, q.GroupID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find approved group images: %w", err)
	}

	// 3. Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &ListApprovedGroupImagesResult{
		Images:     images,
		Total:      total,
		Pagination: pagination,
	}, nil
}
