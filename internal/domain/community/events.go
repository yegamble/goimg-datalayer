package community

import (
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Group Lifecycle Events

// GroupCreated is emitted when a new group is created.
type GroupCreated struct {
	shared.BaseEvent
	GroupID   GroupID
	OwnerID   identity.UserID
	Name      GroupName
	Slug      GroupSlug
	GroupType GroupType
}

// GroupDescriptionUpdated is emitted when a group's description is updated.
type GroupDescriptionUpdated struct {
	shared.BaseEvent
	GroupID        GroupID
	NewDescription string
}

// GroupSettingsUpdated is emitted when a group's settings are updated.
type GroupSettingsUpdated struct {
	shared.BaseEvent
	GroupID     GroupID
	OldSettings GroupSettings
	NewSettings GroupSettings
}

// GroupCoverImageChanged is emitted when a group's cover image changes.
type GroupCoverImageChanged struct {
	shared.BaseEvent
	GroupID      GroupID
	CoverImageID gallery.ImageID
}

// GroupDeleted is emitted when a group is deleted.
type GroupDeleted struct {
	shared.BaseEvent
	GroupID   GroupID
	DeletedBy identity.UserID
}

// Group Membership Events

// MemberJoined is emitted when a user joins a group directly (public groups).
type MemberJoined struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
	Role    GroupRole
}

// MemberInvited is emitted when a user is invited to join a group.
type MemberInvited struct {
	shared.BaseEvent
	GroupID   GroupID
	UserID    identity.UserID
	InvitedBy identity.UserID
	Role      GroupRole
}

// MemberRequested is emitted when a user requests to join an invite-only group.
type MemberRequested struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
}

// MemberActivated is emitted when a member's status becomes active.
// This occurs when accepting an invitation or when a join request is approved.
type MemberActivated struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
}

// MemberLeft is emitted when a member leaves a group.
type MemberLeft struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
}

// MemberRemoved is emitted when a member is removed from a group by an admin/owner.
type MemberRemoved struct {
	shared.BaseEvent
	GroupID   GroupID
	UserID    identity.UserID
	RemovedBy identity.UserID
}

// MemberRoleChanged is emitted when a member's role is changed.
type MemberRoleChanged struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
	OldRole GroupRole
	NewRole GroupRole
}

// MemberBanned is emitted when a member is banned from a group.
type MemberBanned struct {
	shared.BaseEvent
	GroupID  GroupID
	UserID   identity.UserID
	BannedBy identity.UserID
	Reason   string
}

// MemberUnbanned is emitted when a member is unbanned from a group.
type MemberUnbanned struct {
	shared.BaseEvent
	GroupID GroupID
	UserID  identity.UserID
}

// Group Content Events

// ImageSharedToGroup is emitted when an image is shared to a group's pool.
type ImageSharedToGroup struct {
	shared.BaseEvent
	GroupID GroupID
	ImageID gallery.ImageID
	UserID  identity.UserID
}

// ImageRemovedFromGroup is emitted when an image is removed from a group's pool.
type ImageRemovedFromGroup struct {
	shared.BaseEvent
	GroupID   GroupID
	ImageID   gallery.ImageID
	RemovedBy identity.UserID
}

// GroupImageApproved is emitted when a pending group image is approved.
type GroupImageApproved struct {
	shared.BaseEvent
	GroupID    GroupID
	ImageID    gallery.ImageID
	ApprovedBy identity.UserID
}

// GroupImageRejected is emitted when a pending group image is rejected.
type GroupImageRejected struct {
	shared.BaseEvent
	GroupID    GroupID
	ImageID    gallery.ImageID
	RejectedBy identity.UserID
	Reason     string
}

// Group Album Events

// GroupAlbumCreated is emitted when a group album is created.
type GroupAlbumCreated struct {
	shared.BaseEvent
	GroupAlbumID GroupAlbumID
	GroupID      GroupID
	CreatedBy    identity.UserID
	Title        string
}

// GroupAlbumUpdated is emitted when a group album is updated.
// Fields are optional - only the changed fields are populated.
type GroupAlbumUpdated struct {
	shared.BaseEvent
	GroupAlbumID    GroupAlbumID
	GroupID         GroupID
	OldTitle        string           // Populated when title changes
	NewTitle        string           // Populated when title changes
	NewDescription  *string          // Populated when description changes
	NewCoverImageID *gallery.ImageID // Populated when cover image changes
	NewIsPublic     *bool            // Populated when visibility changes
}

// GroupAlbumDeleted is emitted when a group album is deleted.
type GroupAlbumDeleted struct {
	shared.BaseEvent
	GroupAlbumID GroupAlbumID
	GroupID      GroupID
	DeletedBy    identity.UserID
}

// ImageAddedToGroupAlbum is emitted when an image is added to a group album.
type ImageAddedToGroupAlbum struct {
	shared.BaseEvent
	GroupAlbumID GroupAlbumID
	GroupID      GroupID
	ImageID      gallery.ImageID
	AddedBy      identity.UserID
}

// ImageRemovedFromGroupAlbum is emitted when an image is removed from a group album.
type ImageRemovedFromGroupAlbum struct {
	shared.BaseEvent
	GroupAlbumID GroupAlbumID
	GroupID      GroupID
	ImageID      gallery.ImageID
	RemovedBy    identity.UserID
}

// Group Invitation Events

// GroupInvitationCreated is emitted when a group invitation is created.
type GroupInvitationCreated struct {
	shared.BaseEvent
	InvitationID InvitationID
	GroupID      GroupID
	InvitedBy    identity.UserID
	Email        *string          // Email address for non-registered users
	UserID       *identity.UserID // User ID for existing users
}

// GroupInvitationAccepted is emitted when a group invitation is accepted.
type GroupInvitationAccepted struct {
	shared.BaseEvent
	InvitationID InvitationID
	GroupID      GroupID
	AcceptedAt   time.Time
}

// GroupInvitationDeclined is emitted when a group invitation is declined.
type GroupInvitationDeclined struct {
	shared.BaseEvent
	InvitationID InvitationID
	GroupID      GroupID
	DeclinedAt   time.Time
}
