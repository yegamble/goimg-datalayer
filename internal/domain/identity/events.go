package identity

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// UserCreated is emitted when a new user is created.
type UserCreated struct {
	shared.BaseEvent
	UserID   UserID
	Email    Email
	Username Username
}

// NewUserCreated creates a new UserCreated event.
func NewUserCreated(userID UserID, email Email, username Username) UserCreated {
	return UserCreated{
		BaseEvent: shared.NewBaseEvent("identity.user.created", userID.String()),
		UserID:    userID,
		Email:     email,
		Username:  username,
	}
}

// UserProfileUpdated is emitted when a user's profile is updated.
type UserProfileUpdated struct {
	shared.BaseEvent
	UserID      UserID
	DisplayName string
	Bio         string
}

// NewUserProfileUpdated creates a new UserProfileUpdated event.
func NewUserProfileUpdated(userID UserID, displayName, bio string) UserProfileUpdated {
	return UserProfileUpdated{
		BaseEvent:   shared.NewBaseEvent("identity.user.profile_updated", userID.String()),
		UserID:      userID,
		DisplayName: displayName,
		Bio:         bio,
	}
}

// UserRoleChanged is emitted when a user's role is changed.
type UserRoleChanged struct {
	shared.BaseEvent
	UserID  UserID
	OldRole Role
	NewRole Role
}

// NewUserRoleChanged creates a new UserRoleChanged event.
func NewUserRoleChanged(userID UserID, oldRole, newRole Role) UserRoleChanged {
	return UserRoleChanged{
		BaseEvent: shared.NewBaseEvent("identity.user.role_changed", userID.String()),
		UserID:    userID,
		OldRole:   oldRole,
		NewRole:   newRole,
	}
}

// UserSuspended is emitted when a user is suspended.
type UserSuspended struct {
	shared.BaseEvent
	UserID UserID
	Reason string
}

// NewUserSuspended creates a new UserSuspended event.
func NewUserSuspended(userID UserID, reason string) UserSuspended {
	return UserSuspended{
		BaseEvent: shared.NewBaseEvent("identity.user.suspended", userID.String()),
		UserID:    userID,
		Reason:    reason,
	}
}

// UserActivated is emitted when a user is activated.
type UserActivated struct {
	shared.BaseEvent
	UserID UserID
}

// NewUserActivated creates a new UserActivated event.
func NewUserActivated(userID UserID) UserActivated {
	return UserActivated{
		BaseEvent: shared.NewBaseEvent("identity.user.activated", userID.String()),
		UserID:    userID,
	}
}

// UserPasswordChanged is emitted when a user's password is changed.
type UserPasswordChanged struct {
	shared.BaseEvent
	UserID UserID
}

// NewUserPasswordChanged creates a new UserPasswordChanged event.
func NewUserPasswordChanged(userID UserID) UserPasswordChanged {
	return UserPasswordChanged{
		BaseEvent: shared.NewBaseEvent("identity.user.password_changed", userID.String()),
		UserID:    userID,
	}
}
