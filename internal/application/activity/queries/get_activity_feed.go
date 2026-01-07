package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/application/activity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// GetActivityFeedQuery retrieves activities from users that the given user follows.
// This is a read-only operation with no side effects.
type GetActivityFeedQuery struct {
	UserID string // User whose feed to retrieve
	Limit  int    // Maximum number of activities to return
	Offset int    // Pagination offset
}

// GetActivityFeedHandler processes GetActivityFeedQuery requests.
// It retrieves activities from all users that the requesting user follows.
type GetActivityFeedHandler struct {
	activities activity.ActivityRepository
	users      identity.UserRepository
}

// NewGetActivityFeedHandler creates a new GetActivityFeedHandler with the given dependencies.
func NewGetActivityFeedHandler(
	activities activity.ActivityRepository,
	users identity.UserRepository,
) *GetActivityFeedHandler {
	return &GetActivityFeedHandler{
		activities: activities,
		users:      users,
	}
}

// Handle executes the GetActivityFeedQuery and returns the activity feed.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Verify user exists
//  3. Retrieve activities from followed users using the repository's JOIN query
//  4. Convert to DTOs
//  5. Return paginated result
//
// Returns:
//   - *dto.ActivityFeedDTO: The paginated list of activities from followed users
//   - error: ErrUserNotFound if the user does not exist, or other repository errors
func (h *GetActivityFeedHandler) Handle(ctx context.Context, q GetActivityFeedQuery) (*dto.ActivityFeedDTO, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Verify user exists
	_, err = h.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	// 3. Create pagination
	pagination, err := shared.NewPagination(
		(q.Offset/q.Limit)+1, // Convert offset to page number (1-indexed)
		q.Limit,
	)
	if err != nil {
		// Fallback to default pagination if invalid
		pagination = shared.DefaultPagination()
	}

	// 4. Retrieve activities from followed users
	activities, totalCount, err := h.activities.FindFeedForUser(ctx, userID, pagination)
	if err != nil {
		return nil, fmt.Errorf("find feed for user: %w", err)
	}

	// 5. Convert to DTOs
	activityDTOs := make([]dto.ActivityDTO, 0, len(activities))
	for _, act := range activities {
		activityDTOs = append(activityDTOs, dto.FromDomain(act))
	}

	// 6. Build response DTO
	result := &dto.ActivityFeedDTO{
		Activities: activityDTOs,
		TotalCount: totalCount,
		Offset:     q.Offset,
		Limit:      q.Limit,
	}

	return result, nil
}
