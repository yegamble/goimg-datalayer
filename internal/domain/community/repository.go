package community

import (
	"context"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// GroupRepository is the repository interface for the Group aggregate.
// Implementations live in the infrastructure layer.
type GroupRepository interface {
	// FindByID retrieves a group by its ID.
	// Returns ErrGroupNotFound if the group doesn't exist.
	FindByID(ctx context.Context, id GroupID) (*Group, error)

	// FindBySlug retrieves a group by its slug.
	// Returns ErrGroupNotFound if the group doesn't exist.
	FindBySlug(ctx context.Context, slug GroupSlug) (*Group, error)

	// FindPublicGroups retrieves a paginated list of public and invite-only groups.
	// Private groups are excluded from this query.
	// Returns the groups, total count, and error.
	FindPublicGroups(ctx context.Context, filter GroupFilter, pagination shared.Pagination) ([]*Group, int, error)

	// SearchGroups searches for groups by name or description.
	// Only returns public and invite-only groups (private groups are not searchable).
	// Returns the matching groups, total count, and error.
	SearchGroups(ctx context.Context, query string, pagination shared.Pagination) ([]*Group, int, error)

	// FindByOwner retrieves all groups owned by a specific user.
	FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*Group, int, error)

	// Save persists the group to storage.
	// This handles both creation and updates.
	Save(ctx context.Context, group *Group) error

	// Delete removes the group from storage.
	// This is typically a soft delete.
	Delete(ctx context.Context, id GroupID) error

	// ExistsWithSlug checks if a group with the given slug exists.
	// Used for slug uniqueness validation.
	ExistsWithSlug(ctx context.Context, slug GroupSlug) (bool, error)
}

// GroupMembershipRepository is the repository interface for GroupMembership entities.
type GroupMembershipRepository interface {
	// FindByID retrieves a membership by its ID.
	// Returns ErrMembershipNotFound if the membership doesn't exist.
	FindByID(ctx context.Context, id MembershipID) (*GroupMembership, error)

	// FindByGroupAndUser retrieves a membership for a specific user in a specific group.
	// Returns ErrMembershipNotFound if the membership doesn't exist.
	FindByGroupAndUser(ctx context.Context, groupID GroupID, userID identity.UserID) (*GroupMembership, error)

	// FindByGroup retrieves all memberships for a specific group with filtering and pagination.
	// Returns the memberships, total count, and error.
	FindByGroup(ctx context.Context, groupID GroupID, filter MemberFilter, pagination shared.Pagination) ([]*GroupMembership, int, error)

	// FindByUser retrieves all groups a user is a member of with pagination.
	// Returns the memberships, total count, and error.
	FindByUser(ctx context.Context, userID identity.UserID, pagination shared.Pagination) ([]*GroupMembership, int, error)

	// FindActiveByGroup retrieves all active members of a group.
	// This is a convenience method that filters by MemberStatusActive.
	FindActiveByGroup(ctx context.Context, groupID GroupID) ([]*GroupMembership, error)

	// FindPendingByGroup retrieves all pending members (invited or requested) of a group.
	FindPendingByGroup(ctx context.Context, groupID GroupID) ([]*GroupMembership, error)

	// CountByGroupAndStatus counts memberships by status for a specific group.
	// Used for checking member limits and pending request counts.
	CountByGroupAndStatus(ctx context.Context, groupID GroupID, status MemberStatus) (int, error)

	// Save persists the membership to storage.
	// This handles both creation and updates.
	Save(ctx context.Context, membership *GroupMembership) error

	// Delete removes the membership from storage.
	Delete(ctx context.Context, id MembershipID) error

	// ExistsActiveByGroupAndUser checks if an active membership exists for the given group and user.
	ExistsActiveByGroupAndUser(ctx context.Context, groupID GroupID, userID identity.UserID) (bool, error)
}

// GroupFilter defines filtering options for group queries.
type GroupFilter struct {
	GroupType *GroupType       // Filter by group type (optional)
	OwnerID   *identity.UserID // Filter by owner (optional)
	SortBy    GroupSortBy      // Sort order
}

// GroupSortBy defines sort options for group queries.
type GroupSortBy string

const (
	// GroupSortByRecent sorts groups by creation date (newest first).
	GroupSortByRecent GroupSortBy = "recent"

	// GroupSortByPopular sorts groups by member count (highest first).
	GroupSortByPopular GroupSortBy = "popular"

	// GroupSortByName sorts groups alphabetically by name.
	GroupSortByName GroupSortBy = "name"

	// GroupSortByActivity sorts groups by most recent activity.
	GroupSortByActivity GroupSortBy = "activity"
)

// MemberFilter defines filtering options for membership queries.
type MemberFilter struct {
	Role   *GroupRole    // Filter by role (optional)
	Status *MemberStatus // Filter by status (optional)
}

// GroupActivityRepository is the repository interface for GroupActivity entities.
// Used for audit logging and activity feeds.
type GroupActivityRepository interface {
	// Save persists a group activity to storage.
	// This is an append-only operation - activities are never updated.
	Save(ctx context.Context, activity *GroupActivity) error

	// FindByGroup retrieves activities for a specific group with pagination.
	// Activities are returned in reverse chronological order (newest first).
	// Returns the activities, total count, and error.
	FindByGroup(ctx context.Context, groupID GroupID, pagination shared.Pagination) ([]*GroupActivity, int, error)

	// FindByGroupAndType retrieves activities for a specific group filtered by activity type.
	// Useful for filtering admin actions (banned, removed, etc.) or specific event types.
	// Returns the activities, total count, and error.
	FindByGroupAndType(ctx context.Context, groupID GroupID, activityType ActivityType, pagination shared.Pagination) ([]*GroupActivity, int, error)
}
