package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UnfollowUserCommand represents the intent for a user to unfollow another user.
// FollowerID is the authenticated user who wants to unfollow someone.
// FollowedID is the target user to be unfollowed.
type UnfollowUserCommand struct {
	FollowerID string // Authenticated user ID (from JWT context)
	FollowedID string // Target user ID (from URL parameter)
}

// UnfollowUserHandler processes unfollow user commands.
// It validates the relationship exists and removes it.
// This operation is idempotent - unfollowing a user you don't follow is a no-op.
type UnfollowUserHandler struct {
	follows identity.FollowRepository
	logger  *zerolog.Logger
}

// NewUnfollowUserHandler creates a new UnfollowUserHandler with the given dependencies.
func NewUnfollowUserHandler(
	follows identity.FollowRepository,
	logger *zerolog.Logger,
) *UnfollowUserHandler {
	return &UnfollowUserHandler{
		follows: follows,
		logger:  logger,
	}
}

// Handle executes the unfollow user use case.
//
// Process flow:
//  1. Parse and validate follower ID (authenticated user)
//  2. Parse and validate followed ID (target user)
//  3. Delete follow relationship
//  4. Log successful operation
//
// This operation is idempotent:
//   - If the follow relationship doesn't exist, the operation succeeds without error
//   - This prevents client-side retry issues and simplifies error handling
//
// Returns:
//   - nil on success (including when relationship doesn't exist)
//   - error if IDs are invalid or database operation fails
func (h *UnfollowUserHandler) Handle(ctx context.Context, cmd UnfollowUserCommand) error {
	// 1. Parse and validate follower ID
	followerID, err := identity.ParseUserID(cmd.FollowerID)
	if err != nil {
		return fmt.Errorf("invalid follower id: %w", err)
	}

	// 2. Parse and validate followed ID
	followedID, err := identity.ParseUserID(cmd.FollowedID)
	if err != nil {
		return fmt.Errorf("invalid followed id: %w", err)
	}

	// 3. Delete follow relationship (idempotent)
	err = h.follows.Delete(ctx, followerID, followedID)
	if err != nil {
		// Make this operation idempotent - if the relationship doesn't exist, treat as success
		if errors.Is(err, identity.ErrFollowNotFound) {
			h.logger.Debug().
				Str("follower_id", followerID.String()).
				Str("followed_id", followedID.String()).
				Msg("unfollow called for non-existent relationship (idempotent)")
			return nil
		}
		return fmt.Errorf("delete follow: %w", err)
	}

	h.logger.Info().
		Str("follower_id", followerID.String()).
		Str("followed_id", followedID.String()).
		Msg("user unfollowed successfully")

	return nil
}
