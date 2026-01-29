package community

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// ActivityType represents the type of activity performed in a group.
type ActivityType string

const (
	// Member lifecycle activities
	ActivityTypeMemberJoined   ActivityType = "member_joined"
	ActivityTypeMemberLeft     ActivityType = "member_left"
	ActivityTypeMemberPromoted ActivityType = "member_promoted"
	ActivityTypeMemberDemoted  ActivityType = "member_demoted"
	ActivityTypeMemberBanned   ActivityType = "member_banned"
	ActivityTypeMemberRemoved  ActivityType = "member_removed"

	// Content activities
	ActivityTypeImageShared   ActivityType = "image_shared"
	ActivityTypeImageApproved ActivityType = "image_approved"
	ActivityTypeImageRejected ActivityType = "image_rejected"

	// Album activities
	ActivityTypeAlbumCreated ActivityType = "album_created"
	ActivityTypeAlbumDeleted ActivityType = "album_deleted"

	// Group management activities
	ActivityTypeSettingsUpdated    ActivityType = "settings_updated"
	ActivityTypeDescriptionUpdated ActivityType = "description_updated"
	ActivityTypeGroupDeleted       ActivityType = "group_deleted"
)

// String returns the string representation of the ActivityType.
func (a ActivityType) String() string {
	return string(a)
}

// IsValid returns true if the activity type is valid.
func (a ActivityType) IsValid() bool {
	switch a {
	case ActivityTypeMemberJoined, ActivityTypeMemberLeft,
		ActivityTypeMemberPromoted, ActivityTypeMemberDemoted,
		ActivityTypeMemberBanned, ActivityTypeMemberRemoved,
		ActivityTypeImageShared, ActivityTypeImageApproved, ActivityTypeImageRejected,
		ActivityTypeAlbumCreated, ActivityTypeAlbumDeleted,
		ActivityTypeSettingsUpdated, ActivityTypeDescriptionUpdated,
		ActivityTypeGroupDeleted:
		return true
	default:
		return false
	}
}

// TargetType represents the type of entity that an activity references.
type TargetType string

const (
	TargetTypeUser  TargetType = "user"
	TargetTypeImage TargetType = "image"
	TargetTypeAlbum TargetType = "album"
	TargetTypeGroup TargetType = "group"
)

// String returns the string representation of the TargetType.
func (t TargetType) String() string {
	return string(t)
}

// IsValid returns true if the target type is valid.
func (t TargetType) IsValid() bool {
	switch t {
	case TargetTypeUser, TargetTypeImage, TargetTypeAlbum, TargetTypeGroup:
		return true
	default:
		return false
	}
}

// GroupActivity represents an activity event in a group for audit logging and activity feeds.
// This is an entity, not an aggregate root.
type GroupActivity struct {
	id           GroupActivityID
	groupID      GroupID
	actorID      identity.UserID
	activityType ActivityType
	targetID     *string // UUID string of the target entity (optional)
	targetType   *TargetType
	metadata     map[string]interface{} // Flexible JSON metadata for activity details
	createdAt    time.Time
}

// NewGroupActivity creates a new GroupActivity with the given parameters.
// This is the factory function that enforces invariants.
func NewGroupActivity(
	groupID GroupID,
	actorID identity.UserID,
	activityType ActivityType,
	targetID *string,
	targetType *TargetType,
	metadata map[string]interface{},
) (*GroupActivity, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("group id cannot be empty")
	}

	if actorID.IsZero() {
		return nil, fmt.Errorf("actor id cannot be empty")
	}

	if !activityType.IsValid() {
		return nil, fmt.Errorf("invalid activity type: %s", activityType)
	}

	// Validate target constraints: both or neither must be set
	if (targetID == nil) != (targetType == nil) {
		return nil, fmt.Errorf("target id and target type must both be set or both be nil")
	}

	if targetType != nil && !targetType.IsValid() {
		return nil, fmt.Errorf("invalid target type: %s", *targetType)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &GroupActivity{
		id:           NewGroupActivityID(),
		groupID:      groupID,
		actorID:      actorID,
		activityType: activityType,
		targetID:     targetID,
		targetType:   targetType,
		metadata:     metadata,
		createdAt:    time.Now().UTC(),
	}, nil
}

// ReconstructGroupActivity reconstitutes a GroupActivity from persistence.
// This bypasses validation and should only be used by the infrastructure layer.
func ReconstructGroupActivity(
	id GroupActivityID,
	groupID GroupID,
	actorID identity.UserID,
	activityType ActivityType,
	targetID *string,
	targetType *TargetType,
	metadata map[string]interface{},
	createdAt time.Time,
) *GroupActivity {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &GroupActivity{
		id:           id,
		groupID:      groupID,
		actorID:      actorID,
		activityType: activityType,
		targetID:     targetID,
		targetType:   targetType,
		metadata:     metadata,
		createdAt:    createdAt,
	}
}

// ID returns the activity's unique identifier.
func (a *GroupActivity) ID() GroupActivityID {
	return a.id
}

// GroupID returns the group this activity belongs to.
func (a *GroupActivity) GroupID() GroupID {
	return a.groupID
}

// ActorID returns the user who performed the activity.
func (a *GroupActivity) ActorID() identity.UserID {
	return a.actorID
}

// ActivityType returns the type of activity.
func (a *GroupActivity) ActivityType() ActivityType {
	return a.activityType
}

// TargetID returns the ID of the target entity (optional).
func (a *GroupActivity) TargetID() *string {
	return a.targetID
}

// TargetType returns the type of the target entity (optional).
func (a *GroupActivity) TargetType() *TargetType {
	return a.targetType
}

// Metadata returns the activity metadata.
func (a *GroupActivity) Metadata() map[string]interface{} {
	// Return a copy to prevent external modification
	metadataCopy := make(map[string]interface{}, len(a.metadata))
	for k, v := range a.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// CreatedAt returns when the activity was created.
func (a *GroupActivity) CreatedAt() time.Time {
	return a.createdAt
}

// Helper factory functions for common activities

// NewMemberBannedActivity creates an activity for a member being banned.
func NewMemberBannedActivity(
	groupID GroupID,
	bannedBy identity.UserID,
	bannedUser identity.UserID,
	reason string,
) (*GroupActivity, error) {
	targetID := bannedUser.String()
	targetType := TargetTypeUser

	metadata := map[string]interface{}{
		"reason": reason,
	}

	return NewGroupActivity(
		groupID,
		bannedBy,
		ActivityTypeMemberBanned,
		&targetID,
		&targetType,
		metadata,
	)
}

// NewMemberRemovedActivity creates an activity for a member being removed.
func NewMemberRemovedActivity(
	groupID GroupID,
	removedBy identity.UserID,
	removedUser identity.UserID,
) (*GroupActivity, error) {
	targetID := removedUser.String()
	targetType := TargetTypeUser

	return NewGroupActivity(
		groupID,
		removedBy,
		ActivityTypeMemberRemoved,
		&targetID,
		&targetType,
		nil,
	)
}

// NewMemberRoleChangedActivity creates an activity for a member role change.
func NewMemberRoleChangedActivity(
	groupID GroupID,
	changedBy identity.UserID,
	targetUser identity.UserID,
	oldRole GroupRole,
	newRole GroupRole,
) (*GroupActivity, error) {
	targetID := targetUser.String()
	targetType := TargetTypeUser

	// Determine if this is a promotion or demotion
	activityType := ActivityTypeMemberPromoted
	// Roles are integers where higher value = higher permission
	if newRole < oldRole {
		activityType = ActivityTypeMemberDemoted
	}

	metadata := map[string]interface{}{
		"old_role": oldRole.String(),
		"new_role": newRole.String(),
	}

	return NewGroupActivity(
		groupID,
		changedBy,
		activityType,
		&targetID,
		&targetType,
		metadata,
	)
}

// NewGroupDeletedActivity creates an activity for a group being deleted.
func NewGroupDeletedActivity(
	groupID GroupID,
	deletedBy identity.UserID,
) (*GroupActivity, error) {
	targetID := groupID.String()
	targetType := TargetTypeGroup

	return NewGroupActivity(
		groupID,
		deletedBy,
		ActivityTypeGroupDeleted,
		&targetID,
		&targetType,
		nil,
	)
}
