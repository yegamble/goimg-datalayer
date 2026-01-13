package community

import (
	"context"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
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

// GroupInvitationRepository is the repository interface for GroupInvitation entities.
// Manages invitation persistence and retrieval for group access control.
type GroupInvitationRepository interface {
	// Save persists the invitation to storage.
	// This handles both creation and updates (e.g., marking as used).
	Save(ctx context.Context, invitation *GroupInvitation) error

	// FindByID retrieves an invitation by its ID.
	// Returns ErrInvitationNotFound if the invitation doesn't exist.
	FindByID(ctx context.Context, id InvitationID) (*GroupInvitation, error)

	// FindByToken retrieves an invitation by its secure token.
	// Used when accepting/declining invitations via the token link.
	// Returns ErrInvitationNotFound if the invitation doesn't exist.
	FindByToken(ctx context.Context, token InvitationToken) (*GroupInvitation, error)

	// FindPendingByGroup retrieves all pending (unused, non-expired) invitations for a group.
	// Used by group admins to view outstanding invitations.
	// Returns the invitations and error.
	FindPendingByGroup(ctx context.Context, groupID GroupID) ([]*GroupInvitation, error)

	// FindPendingByUser retrieves all pending invitations sent to a specific user.
	// This includes both email-based invitations (if email matches user's email)
	// and direct user ID invitations.
	// Returns the invitations and error.
	FindPendingByUser(ctx context.Context, userID identity.UserID) ([]*GroupInvitation, error)

	// Delete removes the invitation from storage.
	// Used when declining an invitation or cleaning up expired invitations.
	Delete(ctx context.Context, id InvitationID) error
}

// GroupImageRepository is the repository interface for GroupImage entities.
// Manages image sharing and moderation queue for groups.
type GroupImageRepository interface {
	// Save persists the group image to storage.
	// This handles both creation and updates (e.g., approval/rejection).
	Save(ctx context.Context, groupImage *GroupImage) error

	// FindByID retrieves a group image by its ID.
	// Returns ErrGroupImageNotFound if the group image doesn't exist.
	FindByID(ctx context.Context, id GroupImageID) (*GroupImage, error)

	// FindByGroupAndImage retrieves a group image by group ID and image ID.
	// Returns ErrGroupImageNotFound if not found.
	FindByGroupAndImage(ctx context.Context, groupID GroupID, imageID gallery.ImageID) (*GroupImage, error)

	// FindByGroup retrieves images for a specific group with optional status filtering.
	// If status is nil, returns all images regardless of status.
	// Returns the images, total count, and error.
	FindByGroup(ctx context.Context, groupID GroupID, status *GroupImageStatus, pagination shared.Pagination) ([]*GroupImage, int, error)

	// FindPendingByGroup retrieves all pending images for a group (moderation queue).
	// Returns images in chronological order (oldest first) for FIFO moderation.
	FindPendingByGroup(ctx context.Context, groupID GroupID, pagination shared.Pagination) ([]*GroupImage, int, error)

	// FindApprovedByGroup retrieves all approved images for a group.
	// Returns images in reverse chronological order (newest first).
	FindApprovedByGroup(ctx context.Context, groupID GroupID, pagination shared.Pagination) ([]*GroupImage, int, error)

	// Delete removes the group image from storage.
	Delete(ctx context.Context, id GroupImageID) error

	// ExistsByGroupAndImage checks if an image is already shared to the group.
	ExistsByGroupAndImage(ctx context.Context, groupID GroupID, imageID gallery.ImageID) (bool, error)
}

// GroupAlbumRepository is the repository interface for GroupAlbum entities.
// Manages album persistence and retrieval for group image organization.
type GroupAlbumRepository interface {
	// Save persists the group album to storage.
	// This handles both creation and updates.
	Save(ctx context.Context, album *GroupAlbum) error

	// FindByID retrieves a group album by its ID.
	// Returns ErrGroupAlbumNotFound if the album doesn't exist.
	FindByID(ctx context.Context, id GroupAlbumID) (*GroupAlbum, error)

	// FindByGroup retrieves albums for a specific group with pagination.
	// Albums are returned in reverse chronological order (newest first).
	// Returns the albums, total count, and error.
	FindByGroup(ctx context.Context, groupID GroupID, pagination shared.Pagination) ([]*GroupAlbum, int, error)

	// FindByGroupAndCreator retrieves albums created by a specific user within a group.
	// Useful for filtering by album creator.
	// Returns the albums, total count, and error.
	FindByGroupAndCreator(
		ctx context.Context,
		groupID GroupID,
		creatorID identity.UserID,
		pagination shared.Pagination,
	) ([]*GroupAlbum, int, error)

	// Delete removes the group album from storage.
	// This is typically a hard delete since albums are just organizational containers.
	Delete(ctx context.Context, id GroupAlbumID) error
}

// GroupAlbumImageRepository defines the interface for managing the many-to-many
// relationship between group albums and images. This is separate from the GroupAlbumRepository
// because it represents a relationship, not an aggregate.
type GroupAlbumImageRepository interface {
	// AddImageToAlbum adds an image to a group album.
	// The addedBy parameter tracks who added the image.
	// Returns an error if the image is already in the album.
	AddImageToAlbum(
		ctx context.Context,
		albumID GroupAlbumID,
		imageID gallery.ImageID,
		addedBy identity.UserID,
	) error

	// RemoveImageFromAlbum removes an image from a group album.
	// Returns no error if the image wasn't in the album (idempotent).
	RemoveImageFromAlbum(
		ctx context.Context,
		albumID GroupAlbumID,
		imageID gallery.ImageID,
	) error

	// IsImageInAlbum checks if an image is in a group album.
	IsImageInAlbum(
		ctx context.Context,
		albumID GroupAlbumID,
		imageID gallery.ImageID,
	) (bool, error)

	// FindImagesInAlbum retrieves all images in a group album with pagination.
	// Returns the images, total count, and any error.
	FindImagesInAlbum(
		ctx context.Context,
		albumID GroupAlbumID,
		pagination shared.Pagination,
	) ([]*gallery.Image, int, error)

	// GetImageAddedBy returns the user who added a specific image to an album.
	// Returns ErrGroupImageNotFound if the association doesn't exist.
	GetImageAddedBy(
		ctx context.Context,
		albumID GroupAlbumID,
		imageID gallery.ImageID,
	) (identity.UserID, error)
}
