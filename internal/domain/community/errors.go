package community

import "errors"

// Domain errors for the community bounded context.
// These errors represent business rule violations and validation failures.
// Use fmt.Errorf("operation: %w", err) to wrap with additional context.
var (
	// Entity not found errors.
	ErrGroupNotFound      = errors.New("group not found")
	ErrMembershipNotFound = errors.New("membership not found")
	ErrGroupAlbumNotFound = errors.New("group album not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrGroupImageNotFound = errors.New("group image not found")

	// Group lifecycle errors.
	ErrGroupDeleted = errors.New("group has been deleted")

	// Validation errors - Group name.
	ErrGroupNameRequired = errors.New("group name is required")
	ErrGroupNameTooShort = errors.New("group name must be at least 3 characters")
	ErrGroupNameTooLong  = errors.New("group name exceeds 100 characters")

	// Validation errors - Group slug.
	ErrGroupSlugRequired = errors.New("group slug is required")
	ErrGroupSlugTooShort = errors.New("group slug must be at least 3 characters")
	ErrGroupSlugTooLong  = errors.New("group slug exceeds 110 characters")
	ErrGroupSlugInvalid  = errors.New("group slug contains invalid characters (use only a-z, 0-9, hyphens)")
	ErrGroupSlugTaken    = errors.New("group slug is already taken")

	// Validation errors - Group description.
	ErrGroupDescTooLong = errors.New("group description exceeds 1000 characters")

	// Validation errors - Group type.
	ErrInvalidGroupType = errors.New("invalid group type")

	// Validation errors - Group settings.
	ErrInvalidMaxMembers = errors.New("max members must be 0 (unlimited) or greater than 0")

	// Membership errors.
	ErrNotGroupMember        = errors.New("user is not a member of this group")
	ErrAlreadyGroupMember    = errors.New("user is already a member")
	ErrInsufficientGroupRole = errors.New("insufficient permissions for this action")
	ErrCannotLeaveAsOwner    = errors.New("group owner cannot leave (transfer ownership first)")
	ErrOwnerCannotLeave      = errors.New("owner cannot leave the group")
	ErrMemberBanned          = errors.New("user is banned from this group")
	ErrMemberLimitReached    = errors.New("group has reached maximum member capacity")

	// Role errors.
	ErrInvalidGroupRole    = errors.New("invalid group role")
	ErrCannotDemoteOwner   = errors.New("cannot demote group owner")
	ErrCannotRemoveOwner   = errors.New("cannot remove group owner")
	ErrCannotBanOwner      = errors.New("cannot ban group owner")
	ErrMemberAlreadyActive = errors.New("member is already active")
	ErrMemberAlreadyBanned = errors.New("member is already banned")
	ErrInvalidMemberStatus = errors.New("invalid member status")

	// Invitation errors.
	ErrInvitationExpired     = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed = errors.New("invitation has already been used")
	ErrInvitationInvalid     = errors.New("invitation is invalid")

	// Group album errors.
	ErrGroupAlbumTitleRequired = errors.New("group album title is required")
	ErrGroupAlbumTitleTooLong  = errors.New("group album title exceeds 255 characters")
	ErrGroupAlbumDescTooLong   = errors.New("group album description exceeds 2000 characters")

	// Group image errors.
	ErrImageAlreadyShared      = errors.New("image is already shared to this group")
	ErrImageNotPending         = errors.New("image is not in pending status")
	ErrInvalidGroupImageStatus = errors.New("invalid group image status")
	ErrImageAlreadyInAlbum     = errors.New("image is already in this album")

	// Business rule violations.
	ErrUnauthorizedGroupAccess = errors.New("unauthorized to access this group")
	ErrPrivateGroupNoAccess    = errors.New("cannot access private group without membership")
)
