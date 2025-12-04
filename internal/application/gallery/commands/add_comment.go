package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// AddCommentCommand represents the intent to add a comment to an image.
type AddCommentCommand struct {
	UserID  string
	ImageID string
	Content string
}

// AddCommentHandler processes add comment commands.
// It sanitizes the comment content, validates inputs, creates the comment,
// updates the denormalized comment count, and publishes the CommentAdded domain event.
type AddCommentHandler struct {
	images    gallery.ImageRepository
	comments  gallery.CommentRepository
	users     identity.UserRepository
	publisher EventPublisher
	logger    *zerolog.Logger
	sanitizer *bluemonday.Policy
}

// NewAddCommentHandler creates a new AddCommentHandler with the given dependencies.
func NewAddCommentHandler(
	images gallery.ImageRepository,
	comments gallery.CommentRepository,
	users identity.UserRepository,
	publisher EventPublisher,
	logger *zerolog.Logger,
) *AddCommentHandler {
	return &AddCommentHandler{
		images:    images,
		comments:  comments,
		users:     users,
		publisher: publisher,
		logger:    logger,
		sanitizer: bluemonday.StrictPolicy(), // Strips ALL HTML tags
	}
}

// Handle executes the add comment use case.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Parse and validate image ID
//  3. Verify user exists
//  4. Verify image exists
//  5. Check image visibility (must be accessible to user)
//  6. Sanitize comment content (strip HTML/script tags) - CRITICAL SECURITY
//  7. Validate content length (max 5000 chars as per migration, but domain allows 1000)
//  8. Create Comment aggregate via domain factory
//  9. Persist comment
// 10. Update denormalized comment count on image
// 11. Publish CommentAdded domain event after successful save
//
// Returns:
//   - Comment ID string on successful creation
//   - Validation errors from domain value objects
//   - ErrUserNotFound if user doesn't exist
//   - ErrImageNotFound if image doesn't exist
//   - ErrUnauthorizedAccess if image is not visible to user
//   - ErrCommentRequired if content is empty after sanitization
//   - ErrCommentTooLong if content exceeds maximum length
func (h *AddCommentHandler) Handle(ctx context.Context, cmd AddCommentCommand) (string, error) {
	// 1. Parse user ID
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", cmd.UserID).
			Msg("invalid user id for add comment")
		return "", fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Parse image ID
	imageID, err := gallery.ParseImageID(cmd.ImageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", cmd.ImageID).
			Msg("invalid image id for comment")
		return "", fmt.Errorf("invalid image id: %w", err)
	}

	// 3. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Msg("user not found for add comment")
		return "", fmt.Errorf("find user: %w", err)
	}

	// 4. Load image
	image, err := h.images.FindByID(ctx, imageID)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("image not found for comment")
		return "", fmt.Errorf("find image: %w", err)
	}

	// 5. Check image visibility
	// User can comment if:
	// - Image is public, OR
	// - User is the owner
	if image.Visibility() != gallery.VisibilityPublic && !image.IsOwnedBy(userID) {
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Str("visibility", image.Visibility().String()).
			Msg("unauthorized access to comment on image")
		return "", gallery.ErrUnauthorizedAccess
	}

	// 6. Sanitize content - CRITICAL SECURITY MEASURE
	// Strip ALL HTML tags to prevent XSS attacks
	sanitizedContent := h.sanitizer.Sanitize(cmd.Content)
	sanitizedContent = strings.TrimSpace(sanitizedContent)

	// 7. Validate content is not empty after sanitization
	if sanitizedContent == "" {
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("comment content is empty after sanitization")
		return "", gallery.ErrCommentRequired
	}

	// 8. Additional length validation (migration allows 5000, domain allows 1000)
	// Using domain limit (1000) as per domain entity MaxCommentLength
	if len(sanitizedContent) > gallery.MaxCommentLength {
		h.logger.Debug().
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Int("length", len(sanitizedContent)).
			Int("max", gallery.MaxCommentLength).
			Msg("comment content exceeds maximum length")
		return "", fmt.Errorf("%w: got %d characters", gallery.ErrCommentTooLong, len(sanitizedContent))
	}

	// 9. Create comment via domain factory (validates content)
	comment, err := gallery.NewComment(imageID, userID, sanitizedContent)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to create comment")
		return "", fmt.Errorf("create comment: %w", err)
	}

	// 10. Persist comment
	if err := h.comments.Save(ctx, comment); err != nil {
		h.logger.Error().
			Err(err).
			Str("comment_id", comment.ID().String()).
			Str("user_id", userID.String()).
			Str("image_id", imageID.String()).
			Msg("failed to save comment")
		return "", fmt.Errorf("save comment: %w", err)
	}

	// 11. Update denormalized comment count on image
	commentCount, err := h.comments.CountByImage(ctx, imageID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to get comment count")
		return "", fmt.Errorf("get comment count: %w", err)
	}

	image.SetCommentCount(commentCount)

	if err := h.images.Save(ctx, image); err != nil {
		h.logger.Error().
			Err(err).
			Str("image_id", imageID.String()).
			Msg("failed to update image comment count")
		return "", fmt.Errorf("update image comment count: %w", err)
	}

	// 12. Publish CommentAdded domain event AFTER successful save
	for _, event := range comment.Events() {
		if err := h.publisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("comment_id", comment.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish CommentAdded event")
			// Don't fail the operation - the comment was already created
		}
	}
	comment.ClearEvents()

	h.logger.Info().
		Str("comment_id", comment.ID().String()).
		Str("user_id", userID.String()).
		Str("image_id", imageID.String()).
		Int64("comment_count", commentCount).
		Msg("comment added successfully")

	return comment.ID().String(), nil
}
