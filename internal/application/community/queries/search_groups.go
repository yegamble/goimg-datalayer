package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// SearchGroupsQuery searches for groups by name or description.
// Only discoverable groups (public and invite-only) are included in search results.
type SearchGroupsQuery struct {
	Query   string // Search query (searches name and description)
	Page    int    // Page number (1-indexed)
	PerPage int    // Items per page
}

// Implement Query interface.
func (SearchGroupsQuery) isQuery() {}

// SearchGroupsResult contains the paginated search results with total count.
type SearchGroupsResult struct {
	Groups     []*community.Group
	Total      int
	Pagination shared.Pagination
	Query      string
}

// SearchGroupsHandler processes SearchGroupsQuery requests.
type SearchGroupsHandler struct {
	groupRepo community.GroupRepository
}

// NewSearchGroupsHandler creates a new SearchGroupsHandler with the given dependencies.
func NewSearchGroupsHandler(groupRepo community.GroupRepository) *SearchGroupsHandler {
	return &SearchGroupsHandler{
		groupRepo: groupRepo,
	}
}

// Handle executes the SearchGroupsQuery and returns paginated search results.
//
// Returns:
//   - *SearchGroupsResult: Groups matching the query with pagination metadata
//   - error: Repository errors
func (h *SearchGroupsHandler) Handle(ctx context.Context, q SearchGroupsQuery) (*SearchGroupsResult, error) {
	// Sanitize query
	query := strings.TrimSpace(q.Query)
	if query == "" {
		// Return empty results for empty query
		pagination, _ := shared.NewPagination(q.Page, q.PerPage)
		return &SearchGroupsResult{
			Groups:     []*community.Group{},
			Total:      0,
			Pagination: pagination,
			Query:      "",
		}, nil
	}

	// Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Query repository
	groups, total, err := h.groupRepo.SearchGroups(ctx, query, pagination)
	if err != nil {
		return nil, fmt.Errorf("search groups: %w", err)
	}

	// Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &SearchGroupsResult{
		Groups:     groups,
		Total:      total,
		Pagination: pagination,
		Query:      query,
	}, nil
}
