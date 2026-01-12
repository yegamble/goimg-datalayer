package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListPendingGroupImagesQuery retrieves pending images for a group (moderation queue).
// Only admins and owners can view the pending queue.
type ListPendingGroupImagesQuery struct {
	GroupID community.GroupID
	ActorID identity.UserID // Actor requesting the list (for authorization)
	Page    int             // Page number (1-indexed)
	PerPage int             // Items per page
}

// Implement Query interface.
func (ListPendingGroupImagesQuery) isQuery() {}

// ListPendingGroupImagesResult contains the paginated result with total count.
type ListPendingGroupImagesResult struct {
	Images     []*community.GroupImage
	Total      int
	Pagination shared.Pagination
}

// ListPendingGroupImagesHandler processes ListPendingGroupImagesQuery requests.
type ListPendingGroupImagesHandler struct {
	groupImageRepo community.GroupImageRepository
	membershipRepo community.GroupMembershipRepository
}

// NewListPendingGroupImagesHandler creates a new handler with the given dependencies.
func NewListPendingGroupImagesHandler(
	groupImageRepo community.GroupImageRepository,
	membershipRepo community.GroupMembershipRepository,
) *ListPendingGroupImagesHandler {
	return &ListPendingGroupImagesHandler{
		groupImageRepo: groupImageRepo,
		membershipRepo: membershipRepo,
	}
}

// Handle executes the ListPendingGroupImagesQuery and returns paginated results.
//
// Authorization: Actor must be admin or owner of the group.
//
// Returns:
//   - *ListPendingGroupImagesResult: Pending images with pagination metadata
//   - ErrInsufficientGroupRole: If actor lacks permission
//   - error: Repository errors
func (h *ListPendingGroupImagesHandler) Handle(ctx context.Context, q ListPendingGroupImagesQuery) (*ListPendingGroupImagesResult, error) {
	// 1. Verify actor has permission (admin or owner)
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, q.GroupID, q.ActorID)
	if err != nil {
		return nil, fmt.Errorf("find actor membership: %w", err)
	}

	if !membership.IsAdmin() && !membership.IsOwner() {
		return nil, community.ErrInsufficientGroupRole
	}

	// 2. Build pagination
	pagination, err := shared.NewPagination(q.Page, q.PerPage)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// 3. Query repository
	images, total, err := h.groupImageRepo.FindPendingByGroup(ctx, q.GroupID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find pending group images: %w", err)
	}

	// 4. Update pagination with total
	pagination = pagination.WithTotal(int64(total))

	return &ListPendingGroupImagesResult{
		Images:     images,
		Total:      total,
		Pagination: pagination,
	}, nil
}
