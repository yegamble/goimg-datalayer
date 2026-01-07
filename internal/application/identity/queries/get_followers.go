// Package queries implements read operations for the identity bounded context.
package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetFollowersQuery retrieves all followers for a given user with pagination.
// This is a read-only operation with no side effects.
type GetFollowersQuery struct {
	UserID string // User whose followers to retrieve
	Limit  int    // Maximum number of followers to return
	Offset int    // Pagination offset
}

// GetFollowersHandler processes GetFollowersQuery requests.
// It retrieves the follow relationships and enriches them with user profile data.
type GetFollowersHandler struct {
	follows identity.FollowRepository
	users   identity.UserRepository
}

// NewGetFollowersHandler creates a new GetFollowersHandler with the given dependencies.
func NewGetFollowersHandler(
	follows identity.FollowRepository,
	users identity.UserRepository,
) *GetFollowersHandler {
	return &GetFollowersHandler{
		follows: follows,
		users:   users,
	}
}

// Handle executes the GetFollowersQuery and returns the followers list with user data.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Retrieve follow relationships from repository
//  3. For each follow, fetch the follower's user profile
//  4. Convert to DTOs with user information
//  5. Return paginated result
//
// Returns:
//   - *dto.FollowersListDTO: The paginated list of followers with user info
//   - error: ErrUserNotFound if the user does not exist, or other repository errors
func (h *GetFollowersHandler) Handle(ctx context.Context, q GetFollowersQuery) (*dto.FollowersListDTO, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Verify user exists (optional but good practice)
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 3. Retrieve follow relationships
	follows, totalCount, err := h.follows.FindFollowers(ctx, userID, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("find followers: %w", err)
	}

	// 4. Enrich with user profile data
	followerDTOs := make([]dto.FollowUserDTO, 0, len(follows))
	for _, follow := range follows {
		// Fetch follower user profile
		followerUser, err := h.users.FindByID(ctx, follow.FollowerID())
		if err != nil {
			// Log but don't fail - user might have been deleted
			// Skip this follower in the results
			continue
		}

		// Convert to DTO
		followerDTOs = append(followerDTOs, dto.FollowUserDTO{
			UserID:      followerUser.ID().String(),
			Username:    followerUser.Username().String(),
			DisplayName: followerUser.DisplayName(),
			Bio:         followerUser.Bio(),
			FollowedAt:  follow.CreatedAt(),
		})
	}

	// 5. Build response DTO
	result := &dto.FollowersListDTO{
		Followers:  followerDTOs,
		TotalCount: totalCount,
		Offset:     q.Offset,
		Limit:      q.Limit,
	}

	return result, nil
}
