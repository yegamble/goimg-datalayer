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

// ShareImageToGroupCommand represents the intent to share an image to a group.
// The image will be pending approval if the group has require_approval enabled.
type ShareImageToGroupCommand struct {
	GroupID community.GroupID
	ImageID gallery.ImageID
	ActorID identity.UserID // User sharing the image
}

// Implement Command interface.
func (ShareImageToGroupCommand) isCommand() {}

// ShareImageToGroupHandler processes image sharing commands.
type ShareImageToGroupHandler struct {
	groupRepo      community.GroupRepository
	membershipRepo community.GroupMembershipRepository
	groupImageRepo community.GroupImageRepository
	activityRepo   community.GroupActivityRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewShareImageToGroupHandler creates a new ShareImageToGroupHandler with the given dependencies.
func NewShareImageToGroupHandler(
	groupRepo community.GroupRepository,
	membershipRepo community.GroupMembershipRepository,
	groupImageRepo community.GroupImageRepository,
	activityRepo community.GroupActivityRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *ShareImageToGroupHandler {
	return &ShareImageToGroupHandler{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		groupImageRepo: groupImageRepo,
		activityRepo:   activityRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// ShareImageToGroupResult contains the result of the share image operation.
type ShareImageToGroupResult struct {
	GroupImageID community.GroupImageID
	Status       community.GroupImageStatus
}

// Handle executes the share image to group use case.
//
// Process flow:
//  1. Load group to check settings
//  2. Verify actor is an active member
//  3. Check if image is already shared to group
//  4. Create group image (pending or approved based on settings)
//  5. Persist changes
//  6. Create activity log
//  7. Publish domain events
//
// Returns:
//   - ShareImageToGroupResult on success
//   - ErrNotGroupMember if actor is not an active member
//   - ErrImageAlreadyShared if image is already in the group
func (h *ShareImageToGroupHandler) Handle(ctx context.Context, cmd ShareImageToGroupCommand) (*ShareImageToGroupResult, error) {
	// 1. Load group to check settings
	group, err := h.groupRepo.FindByID(ctx, cmd.GroupID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Msg("group not found")
		return nil, fmt.Errorf("find group: %w", err)
	}

	// 2. Verify actor is an active member
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

	// 3. Check if image is already shared to group
	exists, err := h.groupImageRepo.ExistsByGroupAndImage(ctx, cmd.GroupID, cmd.ImageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to check if image exists")
		return nil, fmt.Errorf("check image exists: %w", err)
	}

	if exists {
		h.logger.Debug().
			Str("group_id", cmd.GroupID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("image already shared to group")
		return nil, community.ErrImageAlreadyShared
	}

	// 4. Create group image (pending or approved based on settings)
	// Owners and admins bypass approval requirement
	requiresApproval := group.Settings().RequireApproval() && !(membership.IsAdmin() || membership.IsOwner())

	groupImage, err := community.NewGroupImage(
		cmd.GroupID,
		cmd.ImageID,
		cmd.ActorID,
		requiresApproval,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to create group image")
		return nil, fmt.Errorf("create group image: %w", err)
	}

	// 5. Persist changes
	if err := h.groupImageRepo.Save(ctx, groupImage); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to save group image")
		return nil, fmt.Errorf("save group image: %w", err)
	}

	// 6. Create activity log
	imageIDStr := cmd.ImageID.String()
	targetType := community.TargetTypeImage
	activity, err := community.NewGroupActivity(
		cmd.GroupID,
		cmd.ActorID,
		community.ActivityTypeImageShared,
		&imageIDStr,
		&targetType,
		map[string]interface{}{
			"status": groupImage.Status().String(),
		},
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", cmd.GroupID.String()).
			Str("image_id", cmd.ImageID.String()).
			Msg("failed to create image shared activity")
		// Don't fail the command if audit logging fails
	} else {
		if err := h.activityRepo.Save(ctx, activity); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", cmd.GroupID.String()).
				Str("activity_id", activity.ID().String()).
				Msg("failed to save image shared activity")
			// Don't fail the command if audit logging fails
		}
	}

	h.logger.Info().
		Str("group_id", cmd.GroupID.String()).
		Str("image_id", cmd.ImageID.String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("status", groupImage.Status().String()).
		Msg("image shared to group successfully")

	return &ShareImageToGroupResult{
		GroupImageID: groupImage.ID(),
		Status:       groupImage.Status(),
	}, nil
}
