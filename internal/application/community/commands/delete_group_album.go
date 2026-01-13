package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// DeleteGroupAlbumCommand represents the intent to delete a group album.
// Only the album creator, group admins, or group owner can delete an album.
type DeleteGroupAlbumCommand struct {
	AlbumID community.GroupAlbumID
	ActorID identity.UserID
}

// Implement Command interface.
func (DeleteGroupAlbumCommand) isCommand() {}

// DeleteGroupAlbumHandler processes group album deletion commands.
type DeleteGroupAlbumHandler struct {
	albumRepo      community.GroupAlbumRepository
	membershipRepo community.GroupMembershipRepository
	logger         *zerolog.Logger
}

// NewDeleteGroupAlbumHandler creates a new DeleteGroupAlbumHandler with the given dependencies.
func NewDeleteGroupAlbumHandler(
	albumRepo community.GroupAlbumRepository,
	membershipRepo community.GroupMembershipRepository,
	logger *zerolog.Logger,
) *DeleteGroupAlbumHandler {
	return &DeleteGroupAlbumHandler{
		albumRepo:      albumRepo,
		membershipRepo: membershipRepo,
		logger:         logger,
	}
}

// Handle executes the group album deletion use case.
//
// Process flow:
//  1. Load album to verify it exists
//  2. Load actor's membership to check permissions
//  3. Verify authorization (creator, admin, or owner)
//  4. Delete album (cascade will remove album-image associations)
//
// Authorization:
//   - Actor must be the album creator, a group admin, or the group owner
//
// Returns:
//   - nil on successful deletion
//   - ErrGroupAlbumNotFound if the album doesn't exist
//   - ErrNotGroupMember if actor is not a member
//   - ErrInsufficientGroupRole if actor lacks permissions
func (h *DeleteGroupAlbumHandler) Handle(ctx context.Context, cmd DeleteGroupAlbumCommand) error {
	// 1. Load album to verify it exists
	album, err := h.albumRepo.FindByID(ctx, cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("album not found during deletion")
		return fmt.Errorf("find album: %w", err)
	}

	// 2. Load actor's membership to check permissions
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, album.GroupID(), cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", album.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return community.ErrNotGroupMember
	}

	if !membership.IsActive() {
		h.logger.Warn().
			Str("group_id", album.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor is not an active member")
		return community.ErrNotGroupMember
	}

	// 3. Verify authorization (creator, admin, or owner)
	isCreator := album.IsOwnedBy(cmd.ActorID)
	isAdminOrOwner := membership.IsAdmin() || membership.IsOwner()

	if !isCreator && !isAdminOrOwner {
		h.logger.Warn().
			Str("album_id", cmd.AlbumID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("role", membership.Role().String()).
			Msg("insufficient permissions to delete album")
		return community.ErrInsufficientGroupRole
	}

	// 4. Delete album (cascade will remove album-image associations)
	if err := h.albumRepo.Delete(ctx, cmd.AlbumID); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("failed to delete album")
		return fmt.Errorf("delete album: %w", err)
	}

	h.logger.Info().
		Str("album_id", cmd.AlbumID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Msg("group album deleted successfully")

	return nil
}
