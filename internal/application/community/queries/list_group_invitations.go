package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListGroupInvitationsQuery retrieves a list of pending invitations for a group.
type ListGroupInvitationsQuery struct {
	GroupID community.GroupID
	Page    int // Page number (1-indexed)
	PerPage int // Items per page
}

// Implement Query interface.
func (ListGroupInvitationsQuery) isQuery() {}

// ListGroupInvitationsResult contains the result with total count.
type ListGroupInvitationsResult struct {
	Invitations []*community.GroupInvitation
	Total       int
}

// ListGroupInvitationsHandler processes ListGroupInvitationsQuery requests.
type ListGroupInvitationsHandler struct {
	invitationRepo community.GroupInvitationRepository
}

// NewListGroupInvitationsHandler creates a new ListGroupInvitationsHandler with the given dependencies.
func NewListGroupInvitationsHandler(invitationRepo community.GroupInvitationRepository) *ListGroupInvitationsHandler {
	return &ListGroupInvitationsHandler{
		invitationRepo: invitationRepo,
	}
}

// Handle executes the ListGroupInvitationsQuery.
//
// Returns:
//   - *ListGroupInvitationsResult: Invitations with count
//   - error: Repository errors
func (h *ListGroupInvitationsHandler) Handle(ctx context.Context, q ListGroupInvitationsQuery) (*ListGroupInvitationsResult, error) {
	// Query repository (fetches all pending)
	invitations, err := h.invitationRepo.FindPendingByGroup(ctx, q.GroupID)
	if err != nil {
		return nil, fmt.Errorf("find pending invitations: %w", err)
	}

	total := len(invitations)

	// Apply in-memory pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	offset := pagination.Offset()
	limit := pagination.Limit()

	var pagedInvitations []*community.GroupInvitation
	if offset >= total {
		pagedInvitations = []*community.GroupInvitation{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		pagedInvitations = invitations[offset:end]
	}

	return &ListGroupInvitationsResult{
		Invitations: pagedInvitations,
		Total:       total,
	}, nil
}
