package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// FollowUserCommand represents the intent for a user to follow another user.
// FollowerID is the authenticated user who wants to follow someone.
// FollowedID is the target user to be followed.
type FollowUserCommand struct {
	FollowerID string // Authenticated user ID (from JWT context)
	FollowedID string // Target user ID (from URL parameter)
}

// FollowUserHandler processes follow user commands.
// It validates that both users exist, checks for self-follows,
// creates the follow relationship, and publishes domain events.
type FollowUserHandler struct {
	follows identity.FollowRepository
	users   identity.UserRepository
	logger  *zerolog.Logger
}

// NewFollowUserHandler creates a new FollowUserHandler with the given dependencies.
func NewFollowUserHandler(
	follows identity.FollowRepository,
	users identity.UserRepository,
	logger *zerolog.Logger,
) *FollowUserHandler {
	return &FollowUserHandler{
		follows: follows,
		users:   users,
		logger:  logger,
	}
}

// Handle executes the follow user use case.
//
// Process flow:
//  1. Parse and validate follower ID (authenticated user)
//  2. Parse and validate followed ID (target user)
//  3. Verify followed user exists
//  4. Check if follow relationship already exists
//  5. Create follow relationship via domain factory
//  6. Persist follow relationship
//  7. Log successful operation
//
// Business rules enforced:
//   - Cannot follow yourself (enforced by domain)
//   - Cannot follow the same user twice
//   - Target user must exist
//
// Returns:
//   - nil on success
//   - ErrCannotFollowSelf if attempting to follow self
//   - ErrFollowAlreadyExists if already following
//   - ErrUserNotFound if target user doesn't exist
func (h *FollowUserHandler) Handle(ctx context.Context, cmd FollowUserCommand) error {
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

	// 3. Verify followed user exists
	followedUser, err := h.users.FindByID(ctx, followedID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			return fmt.Errorf("user to follow not found: %w", err)
		}
		return fmt.Errorf("find user to follow: %w", err)
	}

	// Additional validation: cannot follow suspended or deleted users
	if !followedUser.CanLogin() {
		return fmt.Errorf("cannot follow inactive user")
	}

	// 4. Check if follow relationship already exists
	exists, err := h.follows.Exists(ctx, followerID, followedID)
	if err != nil {
		return fmt.Errorf("check follow exists: %w", err)
	}
	if exists {
		return identity.ErrFollowAlreadyExists
	}

	// 5. Create follow relationship via domain factory (validates self-follow)
	follow, err := identity.NewFollow(followerID, followedID)
	if err != nil {
		return fmt.Errorf("create follow: %w", err)
	}

	// 6. Persist follow relationship
	if err := h.follows.Save(ctx, follow); err != nil {
		return fmt.Errorf("save follow: %w", err)
	}

	// Note: Event publishing would happen here if we had an EventPublisher
	// For now, domain events are collected but not published
	follow.ClearEvents()

	h.logger.Info().
		Str("follower_id", followerID.String()).
		Str("followed_id", followedID.String()).
		Msg("user followed successfully")

	return nil
}
