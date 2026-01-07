package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GetFollowingQuery retrieves all users that a given user is following with pagination.
// This is a read-only operation with no side effects.
type GetFollowingQuery struct {
	UserID string // User whose following list to retrieve
	Limit  int    // Maximum number of followed users to return
	Offset int    // Pagination offset
}

// GetFollowingHandler processes GetFollowingQuery requests.
// It retrieves the follow relationships and enriches them with user profile data.
type GetFollowingHandler struct {
	follows identity.FollowRepository
	users   identity.UserRepository
}

// NewGetFollowingHandler creates a new GetFollowingHandler with the given dependencies.
func NewGetFollowingHandler(
	follows identity.FollowRepository,
	users identity.UserRepository,
) *GetFollowingHandler {
	return &GetFollowingHandler{
		follows: follows,
		users:   users,
	}
}

// Handle executes the GetFollowingQuery and returns the following list with user data.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Retrieve follow relationships from repository
//  3. For each follow, fetch the followed user's profile
//  4. Convert to DTOs with user information
//  5. Return paginated result
//
// Returns:
//   - *dto.FollowingListDTO: The paginated list of followed users with user info
//   - error: ErrUserNotFound if the user does not exist, or other repository errors
func (h *GetFollowingHandler) Handle(ctx context.Context, q GetFollowingQuery) (*dto.FollowingListDTO, error) {
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
	follows, totalCount, err := h.follows.FindFollowing(ctx, userID, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("find following: %w", err)
	}

	// 4. Enrich with user profile data
	followingDTOs := make([]dto.FollowUserDTO, 0, len(follows))
	for _, follow := range follows {
		// Fetch followed user profile
		followedUser, err := h.users.FindByID(ctx, follow.FollowedID())
		if err != nil {
			// Log but don't fail - user might have been deleted
			// Skip this user in the results
			continue
		}

		// Convert to DTO
		followingDTOs = append(followingDTOs, dto.FollowUserDTO{
			UserID:      followedUser.ID().String(),
			Username:    followedUser.Username().String(),
			DisplayName: followedUser.DisplayName(),
			Bio:         followedUser.Bio(),
			FollowedAt:  follow.CreatedAt(),
		})
	}

	// 5. Build response DTO
	result := &dto.FollowingListDTO{
		Following:  followingDTOs,
		TotalCount: totalCount,
		Offset:     q.Offset,
		Limit:      q.Limit,
	}

	return result, nil
}
