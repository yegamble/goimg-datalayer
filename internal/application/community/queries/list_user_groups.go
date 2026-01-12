package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListUserGroupsQuery retrieves a paginated list of groups that a user is a member of.
// This includes groups where the user has any membership status (active, invited, requested, banned).
type ListUserGroupsQuery struct {
	UserID  identity.UserID
	Page    int // Page number (1-indexed)
	PerPage int // Items per page
}

// Implement Query interface.
func (ListUserGroupsQuery) isQuery() {}

// ListUserGroupsResult contains the paginated result with total count.
// Includes both the membership and the group data for convenience.
type ListUserGroupsResult struct {
	Memberships []*community.GroupMembership
	Total       int
	Pagination  shared.Pagination
}

// ListUserGroupsHandler processes ListUserGroupsQuery requests.
type ListUserGroupsHandler struct {
	membershipRepo community.GroupMembershipRepository
}

// NewListUserGroupsHandler creates a new ListUserGroupsHandler with the given dependencies.
func NewListUserGroupsHandler(membershipRepo community.GroupMembershipRepository) *ListUserGroupsHandler {
	return &ListUserGroupsHandler{
		membershipRepo: membershipRepo,
	}
}

// Handle executes the ListUserGroupsQuery and returns paginated results.
//
// Returns:
//   - *ListUserGroupsResult: Memberships with pagination metadata
//   - error: Repository errors
func (h *ListUserGroupsHandler) Handle(ctx context.Context, q ListUserGroupsQuery) (*ListUserGroupsResult, error) {
	// Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Query repository
	memberships, total, err := h.membershipRepo.FindByUser(ctx, q.UserID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find user groups: %w", err)
	}

	// Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &ListUserGroupsResult{
		Memberships: memberships,
		Total:       total,
		Pagination:  pagination,
	}, nil
}
