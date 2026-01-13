package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// ListGroupAlbumsQuery retrieves albums for a specific group with pagination.
// Authorization depends on group visibility:
//   - Public groups: Anyone can list albums
//   - Private/Invite-only groups: Must be an active member
type ListGroupAlbumsQuery struct {
	GroupID    community.GroupID
	ActorID    *identity.UserID // Optional: nil for anonymous access
	Pagination shared.Pagination
}

// Implement Query interface.
func (ListGroupAlbumsQuery) isQuery() {}

// ListGroupAlbumsResult contains the paginated list of albums and total count.
type ListGroupAlbumsResult struct {
	Albums     []*community.GroupAlbum
	TotalCount int
}

// ListGroupAlbumsHandler processes list group albums query requests.
type ListGroupAlbumsHandler struct {
	groupRepo      community.GroupRepository
	albumRepo      community.GroupAlbumRepository
	membershipRepo community.GroupMembershipRepository
}

// NewListGroupAlbumsHandler creates a new ListGroupAlbumsHandler with the given dependencies.
func NewListGroupAlbumsHandler(
	groupRepo community.GroupRepository,
	albumRepo community.GroupAlbumRepository,
	membershipRepo community.GroupMembershipRepository,
) *ListGroupAlbumsHandler {
	return &ListGroupAlbumsHandler{
		groupRepo:      groupRepo,
		albumRepo:      albumRepo,
		membershipRepo: membershipRepo,
	}
}

// Handle executes the list group albums query.
//
// Process flow:
//  1. Load group to check visibility settings
//  2. Verify authorization based on group type:
//     - Public: allow access
//     - Private/Invite-only: verify actor is active member
//  3. Retrieve albums with pagination
//  4. Return albums and total count
//
// Authorization:
//   - Public groups: Anyone can list albums
//   - Private/Invite-only groups: Must be an active member
//
// Returns:
//   - *ListGroupAlbumsResult on success
//   - ErrGroupNotFound if the group doesn't exist
//   - ErrPrivateGroupNoAccess if unauthorized for private group
//   - ErrNotGroupMember if not a member of restricted group
func (h *ListGroupAlbumsHandler) Handle(ctx context.Context, q ListGroupAlbumsQuery) (*ListGroupAlbumsResult, error) {
	// 1. Load group to check visibility settings
	group, err := h.groupRepo.FindByID(ctx, q.GroupID)
	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 2. Verify authorization based on group type
	if !group.IsPublic() {
		// Private or invite-only groups require membership
		if q.ActorID == nil {
			return nil, community.ErrPrivateGroupNoAccess
		}

		membership, err := h.membershipRepo.FindByGroupAndUser(ctx, q.GroupID, *q.ActorID)
		if err != nil {
			if errors.Is(err, community.ErrMembershipNotFound) {
				return nil, community.ErrNotGroupMember
			}
			return nil, fmt.Errorf("find membership: %w", err)
		}

		if !membership.IsActive() {
			return nil, community.ErrNotGroupMember
		}
	}

	// 3. Retrieve albums with pagination
	albums, totalCount, err := h.albumRepo.FindByGroup(ctx, q.GroupID, q.Pagination)
	if err != nil {
		return nil, fmt.Errorf("find albums by group: %w", err)
	}

	// 4. Return albums and total count
	return &ListGroupAlbumsResult{
		Albums:     albums,
		TotalCount: totalCount,
	}, nil
}
