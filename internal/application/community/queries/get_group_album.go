package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetGroupAlbumQuery retrieves a specific group album by ID.
// Authorization rules:
//   - Public albums in public groups: Anyone can view
//   - Private albums: Only group members can view
//   - Albums in private groups: Only group members can view
type GetGroupAlbumQuery struct {
	AlbumID community.GroupAlbumID
	ActorID *identity.UserID // Optional: nil for anonymous access
}

// Implement Query interface.
func (GetGroupAlbumQuery) isQuery() {}

// GetGroupAlbumHandler processes get group album query requests.
type GetGroupAlbumHandler struct {
	groupRepo      community.GroupRepository
	albumRepo      community.GroupAlbumRepository
	membershipRepo community.GroupMembershipRepository
}

// NewGetGroupAlbumHandler creates a new GetGroupAlbumHandler with the given dependencies.
func NewGetGroupAlbumHandler(
	groupRepo community.GroupRepository,
	albumRepo community.GroupAlbumRepository,
	membershipRepo community.GroupMembershipRepository,
) *GetGroupAlbumHandler {
	return &GetGroupAlbumHandler{
		groupRepo:      groupRepo,
		albumRepo:      albumRepo,
		membershipRepo: membershipRepo,
	}
}

// Handle executes the get group album query.
//
// Process flow:
//  1. Load album to verify it exists
//  2. Load group to check visibility settings
//  3. Verify authorization based on album and group visibility:
//     - Public album in public group: allow access
//     - Private album: verify actor is active member
//     - Album in private group: verify actor is active member
//  4. Return album
//
// Authorization:
//   - Public albums in public groups: Anyone can view
//   - Private albums OR albums in private groups: Must be active member
//
// Returns:
//   - *community.GroupAlbum on success
//   - ErrGroupAlbumNotFound if the album doesn't exist
//   - ErrGroupNotFound if the group doesn't exist
//   - ErrPrivateGroupNoAccess if unauthorized
//   - ErrNotGroupMember if not a member when required
func (h *GetGroupAlbumHandler) Handle(ctx context.Context, q GetGroupAlbumQuery) (*community.GroupAlbum, error) {
	// 1. Load album to verify it exists
	album, err := h.albumRepo.FindByID(ctx, q.AlbumID)
	if err != nil {
		return nil, fmt.Errorf("find album: %w", err)
	}

	// 2. Load group to check visibility settings
	group, err := h.groupRepo.FindByID(ctx, album.GroupID())
	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 3. Verify authorization based on album and group visibility
	// Public album in public group: anyone can view
	if group.IsPublic() && album.IsPublic() {
		return album, nil
	}

	// Private album OR album in non-public group: require membership
	if q.ActorID == nil {
		return nil, community.ErrPrivateGroupNoAccess
	}

	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, album.GroupID(), *q.ActorID)
	if err != nil {
		if errors.Is(err, community.ErrMembershipNotFound) {
			return nil, community.ErrNotGroupMember
		}
		return nil, fmt.Errorf("find membership: %w", err)
	}

	if !membership.IsActive() {
		return nil, community.ErrNotGroupMember
	}

	// 4. Return album
	return album, nil
}
