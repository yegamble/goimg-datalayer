package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// DeleteCommentCommand represents the intent to delete a comment.
type DeleteCommentCommand struct {
	CommentID string
	UserID    string // User requesting the deletion
}

// DeleteCommentHandler processes delete comment commands.
// It validates that the user owns the comment or has moderator/admin privileges,
// soft deletes the comment, updates the denormalized comment count,
// and publishes the CommentDeleted domain event.
type DeleteCommentHandler struct {
	images    gallery.ImageRepository
	comments  gallery.CommentRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
}

// NewDeleteCommentHandler creates a new DeleteCommentHandler with the given dependencies.
func NewDeleteCommentHandler(
	images gallery.ImageRepository,
	comments gallery.CommentRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *DeleteCommentHandler {
	return &DeleteCommentHandler{
		images:    images,
		comments:  comments,
		users:     users,
		publisher: publisher,
		logger:    logger,
	}
}

// Handle executes the delete comment use case.
//
// Process flow:
//  1. Parse and validate comment ID
//  2. Parse and validate user ID
//  3. Verify user exists
//  4. Load comment
//  5. Check authorization (user owns comment OR user is moderator/admin)
//  6. Soft delete comment
//  7. Update denormalized comment count on image
//  8. Publish CommentDeleted domain event after successful save
//
// Returns:
//   - nil on successful deletion
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
//   - ErrCommentNotFound if comment doesn't exist
//   - ErrUnauthorizedAccess if user is not authorized to delete
//
//nolint:cyclop // Command handler requires sequential validation: comment ID, user ID, permission checks, and deletion
func (h *DeleteCommentHandler) Handle(ctx context.Context, cmd DeleteCommentCommand) error {
	// 1. Parse comment ID
	commentID, err := gallery.ParseCommentID(cmd.CommentID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("comment_id", cmd.CommentID).
			Msg("invalid comment id for delete")
		return fmt.Errorf("invalid comment id: %w", err)
	}

	// 2. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for delete comment")
		return fmt.Errorf("invalid user id: %w", err)
	}

	// 3. Load user
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for delete comment")
		return fmt.Errorf("find user: %w", err)
	}

	// 4. Load comment
	comment, err := h.comments.FindByID(ctx, commentID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("comment_id", commentID.String()).
			Msg("comment not found for delete")
		return fmt.Errorf("find comment: %w", err)
	}

	// 5. Check authorization
	// User can delete if:
	// - User authored the comment, OR
	// - User is a moderator or admin
	isAuthor := comment.IsAuthoredBy(userID)
	isModerator := user.Role() == identity.RoleModerator || user.Role() == identity.RoleAdmin

	if !isAuthor && !isModerator {
		h.logger.Warn().
			Str("user_id", userID.String()).
			Str("comment_id", commentID.String()).
			Str("comment_author_id", comment.UserID().String()).
			Str("user_role", user.Role().String()).
			Msg("unauthorized attempt to delete comment")
		return gallery.ErrUnauthorizedAccess
	}

	// 6. Soft delete comment
	if err := h.comments.Delete(ctx, commentID); err != nil {
		h.logger.Error().
			Err(err).
			Str("comment_id", commentID.String()).
			Str("user_id", userID.String()).
			Msg("failed to delete comment")
		return fmt.Errorf("delete comment: %w", err)
	}

	// 7. Update denormalized comment count on image
	imageID := comment.ImageID()
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to find image for comment count update")
		return fmt.Errorf("find image: %w", err)
	}

	commentCount, err := h.comments.CountByImage(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to get comment count")
		return fmt.Errorf("get comment count: %w", err)
	}

	image.SetCommentCount(commentCount)

	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to update image comment count")
		return fmt.Errorf("update image comment count: %w", err)
	}

	// 8. Publish CommentDeleted domain event
	event := &gallery.CommentDeleted{
		BaseEvent: shared.NewBaseEvent("gallery.comment.deleted", commentID.String()),
		CommentID: commentID,
		ImageID:   imageID,
		UserID:    comment.UserID(),
		DeletedAt: time.Now().UTC(),
	}

	if err := h.publisher.Publish(ctx, event); err != nil {
		h.logger.Error().
			Err(err).
			Str("comment_id", commentID.String()).
			Msg("failed to publish CommentDeleted event")
		// Don't fail the operation - the comment was already deleted
	}

	h.logger.Info().
		Str("comment_id", commentID.String()).
		Str("deleted_by_user_id", userID.String()).
		Str("comment_author_id", comment.UserID().String()).
		Str("image_id", imageID.String()).
		Int64("comment_count", commentCount).
		Msg("comment deleted successfully")

	return nil
}
