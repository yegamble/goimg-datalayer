package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appcommunity "github.com/yegamble/goimg-datalayer/internal/application/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RejectGroupImageCommand represents the intent to reject a pending image in a group.
// Only admins and owners can reject images.
type RejectGroupImageCommand struct {
	GroupImageID community.GroupImageID
	ActorID      identity.UserID // User rejecting the image
	Reason       string          // Optional reason for rejection
}

// Implement Command interface.
func (RejectGroupImageCommand) isCommand() {}

// RejectGroupImageHandler processes image rejection commands.
type RejectGroupImageHandler struct {
	groupImageRepo community.GroupImageRepository
	membershipRepo community.GroupMembershipRepository
	activityRepo   community.GroupActivityRepository
	eventPublisher appcommunity.EventPublisher
	logger         *zerolog.Logger
}

// NewRejectGroupImageHandler creates a new RejectGroupImageHandler with the given dependencies.
func NewRejectGroupImageHandler(
	groupImageRepo community.GroupImageRepository,
	membershipRepo community.GroupMembershipRepository,
	activityRepo community.GroupActivityRepository,
	eventPublisher appcommunity.EventPublisher,
	logger *zerolog.Logger,
) *RejectGroupImageHandler {
	return &RejectGroupImageHandler{
		groupImageRepo: groupImageRepo,
		membershipRepo: membershipRepo,
		activityRepo:   activityRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the reject group image use case.
//
// Process flow:
//  1. Load the group image
//  2. Verify actor is an admin/owner of the group
//  3. Reject the image via domain method
//  4. Persist changes
//  5. Create activity log
//
// Returns:
//   - nil on successful rejection
//   - ErrGroupImageNotFound if the image doesn't exist
//   - ErrInsufficientGroupRole if actor lacks permission
//   - ErrImageNotPending if image is not in pending status
func (h *RejectGroupImageHandler) Handle(ctx context.Context, cmd RejectGroupImageCommand) error {
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
			Msg("unauthorized image rejection attempt")
		return err
	}

	// 3. Reject the image via domain method
	if err := groupImage.Reject(cmd.ActorID); err != nil {
		h.logger.Debug().
			Err(err).
			Str("group_image_id", cmd.GroupImageID.String()).
			Str("current_status", groupImage.Status().String()).
			Msg("failed to reject image")
		return community.ErrImageNotPending
	}

	// 4. Persist changes
	if err := h.groupImageRepo.Save(ctx, groupImage); err != nil {
		h.logger.Error().
			Err(err).
			Str("group_image_id", cmd.GroupImageID.String()).
			Msg("failed to save rejected group image")
		return fmt.Errorf("save group image: %w", err)
	}

	// 5. Create activity log
	imageIDStr := groupImage.ImageID().String()
	targetType := community.TargetTypeImage
	metadata := map[string]interface{}{
		"shared_by": groupImage.SharedBy().String(),
	}
	if cmd.Reason != "" {
		metadata["reason"] = cmd.Reason
	}

	activity, err := community.NewGroupActivity(
		groupImage.GroupID(),
		cmd.ActorID,
		community.ActivityTypeImageRejected,
		&imageIDStr,
		&targetType,
		metadata,
	)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("group_id", groupImage.GroupID().String()).
			Str("image_id", groupImage.ImageID().String()).
			Msg("failed to create image rejected activity")
		// Don't fail the command if audit logging fails
	} else {
		if err := h.activityRepo.Save(ctx, activity); err != nil {
			h.logger.Error().
				Err(err).
				Str("group_id", groupImage.GroupID().String()).
				Str("activity_id", activity.ID().String()).
				Msg("failed to save image rejected activity")
			// Don't fail the command if audit logging fails
		}
	}

	h.logger.Info().
		Str("group_image_id", cmd.GroupImageID.String()).
		Str("group_id", groupImage.GroupID().String()).
		Str("image_id", groupImage.ImageID().String()).
		Str("actor_id", cmd.ActorID.String()).
		Str("reason", cmd.Reason).
		Msg("image rejected successfully")

	return nil
}
