package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RemoveImageFromGroupAlbumCommand represents the intent to remove an image from a group album.
// Authorization rules:
//   - Group admins and owners can remove any image
//   - Regular members can only remove images they added
type RemoveImageFromGroupAlbumCommand struct {
	AlbumID community.GroupAlbumID
	ImageID gallery.ImageID
	ActorID identity.UserID
}

// Implement Command interface.
func (RemoveImageFromGroupAlbumCommand) isCommand() {}

// RemoveImageFromGroupAlbumHandler processes remove image from album commands.
type RemoveImageFromGroupAlbumHandler struct {
	albumRepo      community.GroupAlbumRepository
	albumImageRepo community.GroupAlbumImageRepository
	membershipRepo community.GroupMembershipRepository
	logger         *zerolog.Logger
}

// NewRemoveImageFromGroupAlbumHandler creates a new RemoveImageFromGroupAlbumHandler with the given dependencies.
func NewRemoveImageFromGroupAlbumHandler(
	albumRepo community.GroupAlbumRepository,
	albumImageRepo community.GroupAlbumImageRepository,
	membershipRepo community.GroupMembershipRepository,
	logger *zerolog.Logger,
) *RemoveImageFromGroupAlbumHandler {
	return &RemoveImageFromGroupAlbumHandler{
		albumRepo:      albumRepo,
		albumImageRepo: albumImageRepo,
		membershipRepo: membershipRepo,
		logger:         logger,
	}
}

// Handle executes the remove image from group album use case.
//
// Process flow:
//  1. Load album to verify it exists
//  2. Load actor's membership to verify permissions
//  3. Check if image is in the album
//  4. Verify authorization:
//     - Admin/owner can remove any image
//     - Regular member can only remove images they added
//  5. Remove image from album
//  6. Update album image count
//  7. Persist changes
//
// Authorization:
//   - Group admin or owner can remove any image
//   - Regular member can only remove images they added
//
// Returns:
//   - nil on success
//   - ErrGroupAlbumNotFound if the album doesn't exist
//   - ErrNotGroupMember if actor is not a member
//   - ErrInsufficientGroupRole if actor lacks permissions
//   - Error if image is not in album
func (h *RemoveImageFromGroupAlbumHandler) Handle(ctx context.Context, cmd RemoveImageFromGroupAlbumCommand) error {
	// 1. Load album to verify it exists
	album, err := h.albumRepo.FindByID(ctx, cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("album not found during remove image")
		return fmt.Errorf("find album: %w", err)
	}

	// 2. Load actor's membership to verify permissions
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

	// 3. Check if image is in the album
	inAlbum, err := h.albumImageRepo.IsImageInAlbum(ctx, cmd.AlbumID, cmd.ImageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to check if image is in album")
		return fmt.Errorf("check image in album: %w", err)
	}

	if !inAlbum {
		h.logger.Debug().
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("image not in album")
		return fmt.Errorf("image not in album: %w", community.ErrImageNotInAlbum)
	}

	// 4. Verify authorization
	isAdminOrOwner := membership.IsAdmin() || membership.IsOwner()

	// If not admin/owner, check if actor added this image
	if !isAdminOrOwner {
		addedBy, err := h.albumImageRepo.GetImageAddedBy(ctx, cmd.AlbumID, cmd.ImageID)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", cmd.AlbumID.String()).
				Str("image_id", cmd.ImageID.String()).
				Msg("failed to get image added by")
			return fmt.Errorf("get image added by: %w", err)
		}

		if !addedBy.Equals(cmd.ActorID) {
			h.logger.Warn().
				Str("album_id", cmd.AlbumID.String()).
				Str("image_id", cmd.ImageID.String()).
				Str("actor_id", cmd.ActorID.String()).
				Str("added_by", addedBy.String()).
				Msg("insufficient permissions to remove image")
			return community.ErrInsufficientGroupRole
		}
	}

	// 5. Remove image from album
	if err := h.albumImageRepo.RemoveImageFromAlbum(ctx, cmd.AlbumID, cmd.ImageID); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to remove image from album")
		return fmt.Errorf("remove image from album: %w", err)
	}

	// 6. Update album image count
	album.DecrementImageCount()

	// 7. Persist album changes
	if err := h.albumRepo.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("failed to update album image count")
		return fmt.Errorf("save album: %w", err)
	}

	h.logger.Info().
		Str("album_id", cmd.AlbumID.String()).
		Str("image_id", cmd.ImageID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Int("new_count", album.ImageCount()).
		Msg("image removed from group album successfully")

	return nil
}
