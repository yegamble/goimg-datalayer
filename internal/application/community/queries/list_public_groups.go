package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListPublicGroupsQuery retrieves a paginated list of discoverable groups (public and invite-only).
// Private groups are excluded from this listing.
type ListPublicGroupsQuery struct {
	GroupType *community.GroupType  // Optional: filter by group type
	SortBy    community.GroupSortBy // Sort order (recent, popular, name, activity)
	Page      int                   // Page number (1-indexed)
	PerPage   int                   // Items per page
}

// Implement Query interface.
func (ListPublicGroupsQuery) isQuery() {}

// ListPublicGroupsResult contains the paginated result with total count.
type ListPublicGroupsResult struct {
	Groups     []*community.Group
	Total      int
	Pagination shared.Pagination
}

// ListPublicGroupsHandler processes ListPublicGroupsQuery requests.
type ListPublicGroupsHandler struct {
	groupRepo community.GroupRepository
}

// NewListPublicGroupsHandler creates a new ListPublicGroupsHandler with the given dependencies.
func NewListPublicGroupsHandler(groupRepo community.GroupRepository) *ListPublicGroupsHandler {
	return &ListPublicGroupsHandler{
		groupRepo: groupRepo,
	}
}

// Handle executes the ListPublicGroupsQuery and returns paginated results.
//
// Returns:
//   - *ListPublicGroupsResult: Groups with pagination metadata
//   - error: Repository errors
func (h *ListPublicGroupsHandler) Handle(ctx context.Context, q ListPublicGroupsQuery) (*ListPublicGroupsResult, error) {
	// Build filter
	filter := community.GroupFilter{
		GroupType: q.GroupType,
		SortBy:    q.SortBy,
	}

	// Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Query repository
	groups, total, err := h.groupRepo.FindPublicGroups(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("find public groups: %w", err)
	}

	// Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &ListPublicGroupsResult{
		Groups:     groups,
		Total:      total,
		Pagination: pagination,
	}, nil
}
