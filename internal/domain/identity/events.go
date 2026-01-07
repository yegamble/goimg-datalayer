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

// 2FA Events

// UserTOTPEnabled is emitted when a user enables two-factor authentication.
type UserTOTPEnabled struct {
	shared.BaseEvent
	UserID UserID
}

// NewUserTOTPEnabled creates a new UserTOTPEnabled event.
func NewUserTOTPEnabled(userID UserID) UserTOTPEnabled {
	return UserTOTPEnabled{
		BaseEvent: shared.NewBaseEvent("identity.user.totp_enabled", userID.String()),
		UserID:    userID,
	}
}

// UserTOTPDisabled is emitted when a user disables two-factor authentication.
type UserTOTPDisabled struct {
	shared.BaseEvent
	UserID UserID
}

// NewUserTOTPDisabled creates a new UserTOTPDisabled event.
func NewUserTOTPDisabled(userID UserID) UserTOTPDisabled {
	return UserTOTPDisabled{
		BaseEvent: shared.NewBaseEvent("identity.user.totp_disabled", userID.String()),
		UserID:    userID,
	}
}

// UserBackupCodeUsed is emitted when a user uses a backup code.
type UserBackupCodeUsed struct {
	shared.BaseEvent
	UserID         UserID
	RemainingCodes int
}

// NewUserBackupCodeUsed creates a new UserBackupCodeUsed event.
func NewUserBackupCodeUsed(userID UserID, remainingCodes int) UserBackupCodeUsed {
	return UserBackupCodeUsed{
		BaseEvent:      shared.NewBaseEvent("identity.user.backup_code_used", userID.String()),
		UserID:         userID,
		RemainingCodes: remainingCodes,
	}
}

// UserBackupCodesRegenerated is emitted when a user regenerates their backup codes.
type UserBackupCodesRegenerated struct {
	shared.BaseEvent
	UserID UserID
}

// NewUserBackupCodesRegenerated creates a new UserBackupCodesRegenerated event.
func NewUserBackupCodesRegenerated(userID UserID) UserBackupCodesRegenerated {
	return UserBackupCodesRegenerated{
		BaseEvent: shared.NewBaseEvent("identity.user.backup_codes_regenerated", userID.String()),
		UserID:    userID,
	}
}

// UserUnusualLogin is emitted when a user logs in from an unusual device/location.
type UserUnusualLogin struct {
	shared.BaseEvent
	UserID      UserID
	IPAddress   string
	DeviceName  string
	Fingerprint string
}

// NewUserUnusualLogin creates a new UserUnusualLogin event.
func NewUserUnusualLogin(userID UserID, ipAddress, deviceName, fingerprint string) UserUnusualLogin {
	return UserUnusualLogin{
		BaseEvent:   shared.NewBaseEvent("identity.user.unusual_login", userID.String()),
		UserID:      userID,
		IPAddress:   ipAddress,
		DeviceName:  deviceName,
		Fingerprint: fingerprint,
	}
}

// UserDeviceTrusted is emitted when a user marks a device as trusted.
type UserDeviceTrusted struct {
	shared.BaseEvent
	UserID      UserID
	Fingerprint string
	DeviceName  string
}

// NewUserDeviceTrusted creates a new UserDeviceTrusted event.
func NewUserDeviceTrusted(userID UserID, fingerprint, deviceName string) UserDeviceTrusted {
	return UserDeviceTrusted{
		BaseEvent:   shared.NewBaseEvent("identity.user.device_trusted", userID.String()),
		UserID:      userID,
		Fingerprint: fingerprint,
		DeviceName:  deviceName,
	}
}
