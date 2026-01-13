package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// CreateGroupAlbumCommand represents the intent to create a new album within a group.
// The actor must be a member of the group to create an album.
type CreateGroupAlbumCommand struct {
	GroupID  community.GroupID
	ActorID  identity.UserID
	Title    string
	IsPublic bool
}

// Implement Command interface.
func (CreateGroupAlbumCommand) isCommand() {}

// CreateGroupAlbumHandler processes group album creation commands.
// It orchestrates validation, authorization, and album creation.
type CreateGroupAlbumHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	albumRepo      community.GroupAlbumRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewCreateGroupAlbumHandler creates a new CreateGroupAlbumHandler with the given dependencies.
func NewCreateGroupAlbumHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	albumRepo community.GroupAlbumRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *CreateGroupAlbumHandler {
	return &CreateGroupAlbumHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		albumRepo:      albumRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the group album creation use case.
//
// Process flow:
//  1. Verify actor is an active member (implicitly verifies group exists)
//  2. Create album via domain factory
//  3. Persist album
//  4. Publish domain events
//
// Authorization:
//   - Actor must be an active member of the group
//
// Returns:
//   - *community.GroupAlbum on successful creation
//   - ErrGroupNotFound if the group doesn't exist
//   - ErrNotGroupMember if actor is not an active member
func (h *CreateGroupAlbumHandler) Handle(ctx context.Context, cmd CreateGroupAlbumCommand) (*community.GroupAlbum, error) {
	// 1. Verify actor is an active member
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, cmd.GroupID, cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return nil, community.ErrNotGroupMember
	}

	if !membership.IsActive() {
		h.logger.Warn().
			Str("group_id", cmd.GroupID.String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("status", membership.Status().String()).
			Msg("actor is not an active member")
		return nil, community.ErrNotGroupMember
	}

	// 2. Create album via domain factory
	album, err := community.NewGroupAlbum(cmd.GroupID, cmd.ActorID, cmd.Title, cmd.IsPublic)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("title", cmd.Title).
			Msg("failed to create group album")
		return nil, fmt.Errorf("create group album: %w", err)
	}

	// 3. Persist album
	if err := h.albumRepo.Save(ctx, album); err != nil {
		h.logger.Error().
			Err(err).
			Str("album_id", album.ID().String()).
			Str("group_id", cmd.GroupID.String()).
			Msg("failed to save group album")
		return nil, fmt.Errorf("save group album: %w", err)
	}

	// 4. Publish domain events AFTER successful save
	for _, event := range album.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("album_id", album.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish album domain event")
		}
	}
	album.ClearEvents()

	h.logger.Info().
		Str("album_id", album.ID().String()).
		Str("group_id", cmd.GroupID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("title", cmd.Title).
		Msg("group album created successfully")

	return album, nil
}
