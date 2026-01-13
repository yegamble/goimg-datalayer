package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UpdateGroupAlbumCommand represents the intent to update a group album's metadata.
// Only the album creator, group admins, or group owner can update an album.
type UpdateGroupAlbumCommand struct {
	AlbumID      community.GroupAlbumID
	ActorID      identity.UserID
	Title        *string          // Optional: update title
	Description  *string          // Optional: update description
	CoverImageID *gallery.ImageID // Optional: update cover image (nil to remove)
	IsPublic     *bool            // Optional: update visibility
}

// Implement Command interface.
func (UpdateGroupAlbumCommand) isCommand() {}

// UpdateGroupAlbumHandler processes group album update commands.
type UpdateGroupAlbumHandler struct {
	albumRepo      community.GroupAlbumRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewUpdateGroupAlbumHandler creates a new UpdateGroupAlbumHandler with the given dependencies.
func NewUpdateGroupAlbumHandler(
	albumRepo community.GroupAlbumRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *UpdateGroupAlbumHandler {
	return &UpdateGroupAlbumHandler{
		albumRepo:      albumRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the group album update use case.
//
// Process flow:
//  1. Load album to verify it exists
//  2. Load actor's membership to check permissions
//  3. Verify authorization (creator, admin, or owner)
//  4. Apply updates via domain methods
//  5. Persist changes
//  6. Publish domain events
//
// Authorization:
//   - Actor must be the album creator, a group admin, or the group owner
//
// Returns:
//   - *community.GroupAlbum on successful update
//   - ErrGroupAlbumNotFound if the album doesn't exist
//   - ErrNotGroupMember if actor is not a member
//   - ErrInsufficientGroupRole if actor lacks permissions
func (h *UpdateGroupAlbumHandler) Handle(ctx context.Context, cmd UpdateGroupAlbumCommand) (*community.GroupAlbum, error) {
	// 1. Load album to verify it exists
	album, err := h.albumRepo.FindByID(ctx, cmd.AlbumID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("album not found during update")
		return nil, fmt.Errorf("find album: %w", err)
	}

	// 2. Load actor's membership to check permissions
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, album.GroupID(), cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", album.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return nil, community.ErrNotGroupMember
	}

	if !membership.IsActive() {
		h.logger.Warn().
			Str("group_id", album.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor is not an active member")
		return nil, community.ErrNotGroupMember
	}

	// 3. Verify authorization (creator, admin, or owner)
	isCreator := album.IsOwnedBy(cmd.ActorID)
	isAdminOrOwner := membership.IsAdmin() || membership.IsOwner()

	if !isCreator && !isAdminOrOwner {
		h.logger.Warn().
			Str("album_id", cmd.AlbumID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("role", membership.Role().String()).
			Msg("insufficient permissions to update album")
		return nil, community.ErrInsufficientGroupRole
	}

	// 4. Apply updates via domain methods
	if cmd.Title != nil {
		if err := album.UpdateTitle(*cmd.Title); err != nil {
			h.logger.Debug().
				Err(err).
				Str("album_id", cmd.AlbumID.String()).
				Msg("failed to update album title")
			return nil, fmt.Errorf("update title: %w", err)
		}
	}

	if cmd.Description != nil {
		if err := album.UpdateDescription(*cmd.Description); err != nil {
			h.logger.Debug().
				Err(err).
				Str("album_id", cmd.AlbumID.String()).
				Msg("failed to update album description")
			return nil, fmt.Errorf("update description: %w", err)
		}
	}

	if cmd.CoverImageID != nil {
		album.SetCoverImage(cmd.CoverImageID)
	}

	if cmd.IsPublic != nil {
		album.SetVisibility(*cmd.IsPublic)
	}

	// 5. Persist changes
	if err := h.albumRepo.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", cmd.AlbumID.String()).
			Msg("failed to save album updates")
		return nil, fmt.Errorf("save album: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	for _, event := range album.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", cmd.AlbumID.String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish album domain event")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", cmd.AlbumID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Msg("group album updated successfully")

	return album, nil
}
