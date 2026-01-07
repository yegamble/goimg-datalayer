package identity

import "context"

// FollowRepository defines the interface for persisting and retrieving Follow relationships.
// Implementations should be provided in the infrastructure layer.
type FollowRepository interface {
	// Save persists a follow relationship to the repository.
	// If the follow relationship already exists, returns ErrFollowAlreadyExists.
	Save(ctx context.Context, follow *Follow) error

	// Delete removes a follow relationship from the repository.
	// Returns ErrFollowNotFound if the relationship does not exist.
	Delete(ctx context.Context, followerID, followedID UserID) error

	// Exists checks whether a follow relationship exists between two users.
	Exists(ctx context.Context, followerID, followedID UserID) (bool, error)

	// FindFollowers retrieves all users following the specified user (pagination supported).
	// Returns a slice of Follow entities and the total count.
	// The Follow entities will have the followerID populated (users who follow the specified user).
	FindFollowers(ctx context.Context, userID UserID, limit, offset int) ([]*Follow, int, error)

	// FindFollowing retrieves all users that the specified user is following (pagination supported).
	// Returns a slice of Follow entities and the total count.
	// The Follow entities will have the followedID populated (users the specified user follows).
	FindFollowing(ctx context.Context, userID UserID, limit, offset int) ([]*Follow, int, error)

	// CountFollowers returns the number of followers for a user.
	CountFollowers(ctx context.Context, userID UserID) (int, error)

	// CountFollowing returns the number of users that the specified user is following.
	CountFollowing(ctx context.Context, userID UserID) (int, error)
}
