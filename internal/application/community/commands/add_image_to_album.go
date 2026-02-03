package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// AddImageToGroupAlbumCommand represents the intent to add an image to a group album.
// The image must already be in the group's image pool (approved).
// Any active member can add images to albums.
type AddImageToGroupAlbumCommand struct {
	AlbumID community.GroupAlbumID
	ImageID gallery.ImageID
	ActorID identity.UserID
}

// Implement Command interface.
func (AddImageToGroupAlbumCommand) isCommand() {}

// AddImageToGroupAlbumHandler processes add image to album commands.
type AddImageToGroupAlbumHandler struct {
	albumRepo      community.GroupAlbumRepository
	albumImageRepo community.GroupAlbumImageRepository
	groupImageRepo community.GroupImageRepository
	membershipRepo community.GroupMembershipRepository
	logger         *zerolog.Logger
}

// NewAddImageToGroupAlbumHandler creates a new AddImageToGroupAlbumHandler with the given dependencies.
func NewAddImageToGroupAlbumHandler(
	albumRepo community.GroupAlbumRepository,
	albumImageRepo community.GroupAlbumImageRepository,
	groupImageRepo community.GroupImageRepository,
	membershipRepo community.GroupMembershipRepository,
	logger *zerolog.Logger,
) *AddImageToGroupAlbumHandler {
	return &AddImageToGroupAlbumHandler{
		albumRepo:      albumRepo,
		albumImageRepo: albumImageRepo,
		groupImageRepo: groupImageRepo,
		membershipRepo: membershipRepo,
		logger:         logger,
	}
}

// Handle executes the add image to group album use case.
//
// Process flow:
//  1. Load album to verify it exists
//  2. Load actor's membership to verify permissions
//  3. Verify image is in the group's approved pool
//  4. Check if image is already in album
//  5. Add image to album
//  6. Update album image count
//  7. Persist changes
//
// Authorization:
//   - Actor must be an active member of the group
//
// Business Rules:
//   - Image must be in the group's image pool with 'approved' status
//   - Image cannot be added to album twice
//
// Returns:
//   - nil on success
//   - ErrGroupAlbumNotFound if the album doesn't exist
//   - ErrNotGroupMember if actor is not a member
//   - ErrGroupImageNotFound if image is not in group pool
//   - Error if image is already in album
func (h *AddImageToGroupAlbumHandler) Handle(ctx context.Context, cmd AddImageToGroupAlbumCommand) error {
	// 1. Load album to verify it exists
	album, err := h.albumRepo.FindByID(ctx, cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("album not found during add image")
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

	// 3. Verify image is in the group's approved pool
	// Only approved images can be added to albums
	exists, err := h.groupImageRepo.ExistsByGroupAndImage(ctx, album.GroupID(), cmd.ImageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", album.GroupID().String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to check if image exists in group")
		return fmt.Errorf("check image in group: %w", err)
	}

	if !exists {
		h.logger.Warn().
			Str("group_id", album.GroupID().String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("attempted to add image not in group pool")
		return fmt.Errorf("image not in group pool: %w", community.ErrGroupImageNotFound)
	}

	// 4. Check if image is already in album
	inAlbum, err := h.albumImageRepo.IsImageInAlbum(ctx, cmd.AlbumID, cmd.ImageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to check if image is in album")
		return fmt.Errorf("check image in album: %w", err)
	}

	if inAlbum {
		h.logger.Debug().
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("image already in album")
		return fmt.Errorf("image already in album: %w", community.ErrImageAlreadyInAlbum)
	}

	// 5. Add image to album
	if err := h.albumImageRepo.AddImageToAlbum(ctx, cmd.AlbumID, cmd.ImageID, cmd.ActorID); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to add image to album")
		return fmt.Errorf("add image to album: %w", err)
	}

	// 6. Update album image count
	album.IncrementImageCount()

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
		Msg("image added to group album successfully")

	return nil
}
