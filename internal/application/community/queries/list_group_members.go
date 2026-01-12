package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListGroupMembersQuery retrieves a paginated list of members in a group.
// Can be filtered by role and status.
type ListGroupMembersQuery struct {
	GroupID community.GroupID
	Role    *community.GroupRole   // Optional: filter by role
	Status  *community.MemberStatus // Optional: filter by status
	Page    int                    // Page number (1-indexed)
	PerPage int                    // Items per page
}

// Implement Query interface.
func (ListGroupMembersQuery) isQuery() {}

// ListGroupMembersResult contains the paginated result with total count.
type ListGroupMembersResult struct {
	Members    []*community.GroupMembership
	Total      int
	Pagination shared.Pagination
}

// ListGroupMembersHandler processes ListGroupMembersQuery requests.
type ListGroupMembersHandler struct {
	membershipRepo community.GroupMembershipRepository
}

// NewListGroupMembersHandler creates a new ListGroupMembersHandler with the given dependencies.
func NewListGroupMembersHandler(membershipRepo community.GroupMembershipRepository) *ListGroupMembersHandler {
	return &ListGroupMembersHandler{
		membershipRepo: membershipRepo,
	}
}

// Handle executes the ListGroupMembersQuery and returns paginated results.
//
// Returns:
//   - *ListGroupMembersResult: Members with pagination metadata
//   - error: Repository errors
//
func (h *ListGroupMembersHandler) Handle(ctx context.Context, q ListGroupMembersQuery) (*ListGroupMembersResult, error) {
	// Build filter
	filter := community.MemberFilter{
		Role:   q.Role,
		Status: q.Status,
	}

	// Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Query repository
	members, total, err := h.membershipRepo.FindByGroup(ctx, q.GroupID, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("find group members: %w", err)
	}

	// Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &ListGroupMembersResult{
		Members:    members,
		Total:      total,
		Pagination: pagination,
	}, nil
}
