package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// CreateGroupCommand represents the intent to create a new group.
// It encapsulates all information needed for group creation.
type CreateGroupCommand struct {
	OwnerID     identity.UserID
	Name        string
	Slug        string
	Description string
	GroupType   string
	Settings    *community.GroupSettings // Optional, will use defaults if nil
}

// Implement Command interface.
func (CreateGroupCommand) isCommand() {}

// CreateGroupHandler processes group creation commands.
// It orchestrates validation, slug uniqueness checks, group creation, and membership creation.
type CreateGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewCreateGroupHandler creates a new CreateGroupHandler with the given dependencies.
func NewCreateGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *CreateGroupHandler {
	return &CreateGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the group creation use case.
//
// Process flow:
//  1. Convert DTOs to domain value objects (validation happens here)
//  2. Check slug uniqueness (business rule)
//  3. Create Group aggregate via domain factory
//  4. Create owner membership
//  5. Persist group and membership
//  6. Publish domain events after successful save
//  7. Return created group
//
// Returns:
//   - *community.Group on successful creation
//   - ErrGroupSlugTaken if slug is already in use
//   - Validation errors from domain value objects
//
//nolint:funlen // Sequential validation and creation steps.
func (h *CreateGroupHandler) Handle(ctx context.Context, cmd CreateGroupCommand) (*community.Group, error) {
	// 1. Convert primitives to domain value objects (validation happens here)
	name, err := community.NewGroupName(cmd.Name)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("name", cmd.Name).
			Msg("invalid group name during creation")
		return nil, fmt.Errorf("invalid group name: %w", err)
	}

	slug, err := community.NewGroupSlug(cmd.Slug)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("slug", cmd.Slug).
			Msg("invalid group slug during creation")
		return nil, fmt.Errorf("invalid group slug: %w", err)
	}

	groupType, err := community.ParseGroupType(cmd.GroupType)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_type", cmd.GroupType).
			Msg("invalid group type during creation")
		return nil, fmt.Errorf("invalid group type: %w", err)
	}

	// 2. Check slug uniqueness (business rule)
	slugExists, err := h.groupRepo.ExistsWithSlug(ctx, slug)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("slug", slug.String()).
			Msg("failed to check slug uniqueness")
		return nil, fmt.Errorf("check slug uniqueness: %w", err)
	}
	if slugExists {
		h.logger.Debug().
			Str("slug", slug.String()).
			Msg("group creation attempt with existing slug")
		return nil, community.ErrGroupSlugTaken
	}

	// 3. Create Group aggregate via domain factory
	group, err := community.NewGroup(cmd.OwnerID, name, slug, groupType)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("owner_id", cmd.OwnerID.String()).
			Str("name", name.String()).
			Str("slug", slug.String()).
			Msg("failed to create group aggregate")
		return nil, fmt.Errorf("create group: %w", err)
	}

	// Apply optional description
	if cmd.Description != "" {
		if err := group.UpdateDescription(cmd.Description); err != nil {
			h.logger.Debug().
				Err(err).
				Int("description_len", len(cmd.Description)).
				Msg("invalid group description")
			return nil, fmt.Errorf("invalid description: %w", err)
		}
	}

	// Apply custom settings if provided
	if cmd.Settings != nil {
		if err := group.UpdateSettings(*cmd.Settings); err != nil {
			h.logger.Debug().
				Err(err).
				Msg("invalid group settings")
			return nil, fmt.Errorf("invalid settings: %w", err)
		}
	}

	// 4. Create owner membership
	ownerMembership, err := community.NewGroupMembership(group.ID(), cmd.OwnerID, community.GroupRoleOwner)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", group.ID().String()).
			Str("owner_id", cmd.OwnerID.String()).
			Msg("failed to create owner membership")
		return nil, fmt.Errorf("create owner membership: %w", err)
	}

	// 5. Persist group and membership
	// Save group first
	if err := h.groupRepo.Save(ctx, group); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", group.ID().String()).
			Str("slug", slug.String()).
			Msg("failed to save group")
		return nil, fmt.Errorf("save group: %w", err)
	}

	// Then save membership
	if err := h.membershipRepo.Save(ctx, ownerMembership); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", group.ID().String()).
			Str("owner_id", cmd.OwnerID.String()).
			Msg("failed to save owner membership")
		return nil, fmt.Errorf("save owner membership: %w", err)
	}

	// 6. Publish domain events AFTER successful save
	// Publish group events
	for _, event := range group.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", group.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish group domain event")
		}
	}
	group.ClearEvents()

	// Publish membership events
	for _, event := range ownerMembership.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", group.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish membership domain event")
		}
	}
	ownerMembership.ClearEvents()

	h.logger.Info().
		Str("group_id", group.ID().String()).
		Str("slug", slug.String()).
		Str("owner_id", cmd.OwnerID.String()).
		Str("group_type", groupType.String()).
		Msg("group created successfully")

	return group, nil
}

// ValidateOwnership checks if the user is the owner of the group.
// This is a helper function for authorization checks in commands.
func ValidateOwnership(group *community.Group, userID identity.UserID) error {
	if !group.IsOwnedBy(userID) {
		return fmt.Errorf("only group owner can perform this action: %w", community.ErrInsufficientGroupRole)
	}
	return nil
}

// ValidateAdminOrOwner checks if the user is an admin or owner of the group.
// This is a helper function for authorization checks in commands.
func ValidateAdminOrOwner(membership *community.GroupMembership) error {
	if !membership.CanManageMembers() {
		return community.ErrInsufficientGroupRole
	}
	return nil
}
