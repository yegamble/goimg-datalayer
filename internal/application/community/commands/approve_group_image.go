package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// ApproveGroupImageCommand represents the intent to approve a pending image in a group.
// Only admins and owners can approve images.
type ApproveGroupImageCommand struct {
	GroupImageID community.GroupImageID
	ActorID      identity.UserID // User approving the image
}

// Implement Command interface.
func (ApproveGroupImageCommand) isCommand() {}

// ApproveGroupImageHandler processes image approval commands.
type ApproveGroupImageHandler struct {
	groupImageRepo community.GroupImageRepository
	membershipRepo community.GroupMembershipRepository
	activityRepo   community.GroupActivityRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewApproveGroupImageHandler creates a new ApproveGroupImageHandler with the given dependencies.
func NewApproveGroupImageHandler(
	groupImageRepo community.GroupImageRepository,
	membershipRepo community.GroupMembershipRepository,
	activityRepo community.GroupActivityRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *ApproveGroupImageHandler {
	return &ApproveGroupImageHandler{
		groupImageRepo: groupImageRepo,
		membershipRepo: membershipRepo,
		activityRepo:   activityRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the approve group image use case.
//
// Process flow:
//  1. Load the group image
//  2. Verify actor is an admin/owner of the group
//  3. Approve the image via domain method
//  4. Persist changes
//  5. Create activity log
//
// Returns:
//   - nil on successful approval
//   - ErrGroupImageNotFound if the image doesn't exist
//   - ErrInsufficientGroupRole if actor lacks permission
//   - ErrImageNotPending if image is not in pending status
func (h *ApproveGroupImageHandler) Handle(ctx context.Context, cmd ApproveGroupImageCommand) error {
	// 1. Load the group image
	groupImage, err := h.groupImageRepo.FindByID(ctx, cmd.GroupImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_image_id", cmd.GroupImageID.String()).
			Msg("group image not found")
		return fmt.Errorf("find group image: %w", err)
	}

	// 2. Verify actor is an admin/owner of the group
	membership, err := h.membershipRepo.FindByGroupAndUser(ctx, groupImage.GroupID(), cmd.ActorID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_id", groupImage.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Msg("actor membership not found")
		return fmt.Errorf("find actor membership: %w", err)
	}

	if err := ValidateAdminOrOwner(membership); err != nil {
		h.logger.Warn().
			Str("group_id", groupImage.GroupID().String()).
			Str("actor_id", cmd.ActorID.String()).
			Str("actor_role", membership.Role().String()).
			Msg("unauthorized image approval attempt")
		return fmt.Errorf("validate approval permission: %w", err)
	}

	// 3. Approve the image via domain method
	if err := groupImage.Approve(cmd.ActorID); err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_image_id", cmd.GroupImageID.String()).
			Str("current_status", groupImage.Status().String()).
			Msg("failed to approve image")
		return community.ErrImageNotPending
	}

	// 4. Persist changes
	if err := h.groupImageRepo.Save(ctx, groupImage); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_image_id", cmd.GroupImageID.String()).
			Msg("failed to save approved group image")
		return fmt.Errorf("save group image: %w", err)
	}

	// 5. Create activity log
	imageIDStr := groupImage.ImageID().String()
	targetType := community.TargetTypeImage
	activity, err := community.NewGroupActivity(
		groupImage.GroupID(),
		cmd.ActorID,
		community.ActivityTypeImageApproved,
		&imageIDStr,
		&targetType,
		map[string]interface{}{
			"shared_by": groupImage.SharedBy().String(),
		},
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", groupImage.GroupID().String()).
			Str("image_id", groupImage.ImageID().String()).
			Msg("failed to create image approved activity")
		// Don't fail the command if audit logging fails
	} else {
		if err := h.activityRepo.Save(ctx, activity); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", groupImage.GroupID().String()).
				Str("activity_id", activity.ID().String()).
				Msg("failed to save image approved activity")
			// Don't fail the command if audit logging fails
		}
	}

	h.logger.Info().
		Str("group_image_id", cmd.GroupImageID.String()).
		Str("group_id", groupImage.GroupID().String()).
		Str("image_id", groupImage.ImageID().String()).
		Str("actor_id", cmd.ActorID.String()).
		Msg("image approved successfully")

	return nil
}
