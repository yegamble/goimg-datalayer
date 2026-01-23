package identity

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// User profile constraints.
const (
	maxDisplayNameLength = 100 // Maximum display name length
	maxBioLength         = 500 // Maximum bio length
)

// User is the aggregate root for the Identity bounded context.
// It represents a user account with authentication and authorization capabilities.
type User struct {
	id           UserID
	email        Email
	username     Username
	passwordHash PasswordHash
	role         Role
	status       UserStatus
	displayName  string
	bio          string
	infectedFileCount int
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent

	// 2FA fields
	totpSecret  *TOTPSecret         // nil if 2FA not set up
	backupCodes []BackupCode        // 10 one-time use codes
	devices     []DeviceFingerprint // Known devices for unusual login detection

	// Notification preferences
	notificationPreferences NotificationPreferences

	// Guest user fields
	userType  UserType   // registered or guest
	ipAddress *string    // IP address for guest users (nil for registered)
	expiresAt *time.Time // Expiration time for guest users (nil for registered)
}

// NewUser creates a new User with the given email, username, and password hash.
// The user is created with RoleUser and StatusPending by default.
// Emits a UserCreated event.
func NewUser(email Email, username Username, passwordHash PasswordHash) (*User, error) {
	if email.IsEmpty() {
		return nil, fmt.Errorf("email is required")
	}
	if username.IsEmpty() {
		return nil, fmt.Errorf("username is required")
	}
	if passwordHash.IsEmpty() {
		return nil, fmt.Errorf("password hash is required")
	}

	now := time.Now().UTC()
	user := &User{
		id:                      NewUserID(),
		email:                   email,
		username:                username,
		passwordHash:            passwordHash,
		role:                    RoleUser,
		status:                  StatusPending,
		displayName:             username.String(), // Default display name is username
		bio:                     "",
		infectedFileCount:       0,
		createdAt:               now,
		updatedAt:               now,
		events:                  []shared.DomainEvent{},
		notificationPreferences: DefaultNotificationPreferences(),
		userType:                UserTypeRegistered,
		ipAddress:               nil,
		expiresAt:               nil,
	}

	user.addEvent(NewUserCreated(user.id, user.email, user.username))
	return user, nil
}

// NewGuestUser creates a new guest user account with the given IP address.
// Guest accounts have:
// - No email/password (anonymous)
// - Auto-generated username (guest_{uuid})
// - RoleUser with StatusActive
// - 30-day expiration
// Emits a GuestUserCreated event.
func NewGuestUser(ipAddress string) (*User, error) {
	if ipAddress == "" {
		return nil, fmt.Errorf("ip address is required for guest users")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour) // 30 days from now

	userID := NewUserID()
	guestUsername := fmt.Sprintf("guest_%s", userID.String()[:8])
	username, err := NewUsername(guestUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest username: %w", err)
	}

	// Generate a dummy email for guests (required by DB schema)
	guestEmail := fmt.Sprintf("guest_%s@goimg.local", userID.String())
	email, err := NewEmail(guestEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest email: %w", err)
	}

	// Guest users don't have a password, but we need a placeholder
	// We use a random unguessable hash that can never be authenticated
	dummyHash, err := NewPasswordHash(userID.String() + now.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create guest password hash: %w", err)
	}

	user := &User{
		id:                      userID,
		email:                   email,
		username:                username,
		passwordHash:            dummyHash,
		role:                    RoleUser,
		status:                  StatusActive,
		displayName:             guestUsername,
		bio:                     "",
		infectedFileCount:       0,
		createdAt:               now,
		updatedAt:               now,
		events:                  []shared.DomainEvent{},
		notificationPreferences: DefaultNotificationPreferences(),
		userType:                UserTypeGuest,
		ipAddress:               &ipAddress,
		expiresAt:               &expiresAt,
	}

	user.addEvent(NewGuestUserCreated(user.id, ipAddress, expiresAt))
	return user, nil
}

// ReconstructUser reconstitutes a User from persistence without validation or events.
// This should only be used by the repository layer when loading from storage.
func ReconstructUser(
	id UserID,
	email Email,
	username Username,
	passwordHash PasswordHash,
	role Role,
	status UserStatus,
	displayName string,
	bio string,
	infectedFileCount int,
	createdAt, updatedAt time.Time,
	userType UserType,
	ipAddress *string,
	expiresAt *time.Time,
) *User {
	return &User{
		id:                      id,
		email:                   email,
		username:                username,
		passwordHash:            passwordHash,
		role:                    role,
		status:                  status,
		displayName:             displayName,
		bio:                     bio,
		infectedFileCount:       infectedFileCount,
		createdAt:               createdAt,
		updatedAt:               updatedAt,
		events:                  []shared.DomainEvent{},
		totpSecret:              nil,
		backupCodes:             nil,
		devices:                 nil,
		notificationPreferences: DefaultNotificationPreferences(),
		userType:                userType,
		ipAddress:               ipAddress,
		expiresAt:               expiresAt,
	}
}

// ReconstructUserWith2FA reconstitutes a User with 2FA data from persistence.
// This should only be used by the repository layer when loading from storage.
func ReconstructUserWith2FA(
	id UserID,
	email Email,
	username Username,
	passwordHash PasswordHash,
	role Role,
	status UserStatus,
	displayName string,
	bio string,
	infectedFileCount int,
	createdAt, updatedAt time.Time,
	totpSecret *TOTPSecret,
	backupCodes []BackupCode,
	devices []DeviceFingerprint,
	userType UserType,
	ipAddress *string,
	expiresAt *time.Time,
) *User {
	return &User{
		id:                      id,
		email:                   email,
		username:                username,
		passwordHash:            passwordHash,
		role:                    role,
		status:                  status,
		displayName:             displayName,
		bio:                     bio,
		infectedFileCount:       infectedFileCount,
		createdAt:               createdAt,
		updatedAt:               updatedAt,
		events:                  []shared.DomainEvent{},
		totpSecret:              totpSecret,
		backupCodes:             backupCodes,
		devices:                 devices,
		notificationPreferences: DefaultNotificationPreferences(),
		userType:                userType,
		ipAddress:               ipAddress,
		expiresAt:               expiresAt,
	}
}

// ID returns the user's unique identifier.
func (u *User) ID() UserID {
	return u.id
}

// Email returns the user's email address.
func (u *User) Email() Email {
	return u.email
}

// Username returns the user's username.
func (u *User) Username() Username {
	return u.username
}

// PasswordHash returns the user's password hash.
// This method is primarily for persistence and should not be used for business logic.
func (u *User) PasswordHash() PasswordHash {
	return u.passwordHash
}

// Role returns the user's role.
func (u *User) Role() Role {
	return u.role
}

// Status returns the user's status.
func (u *User) Status() UserStatus {
	return u.status
}

// DisplayName returns the user's display name.
func (u *User) DisplayName() string {
	return u.displayName
}

// Bio returns the user's bio.
func (u *User) Bio() string {
	return u.bio
}

// InfectedFileCount returns the number of infected files uploaded by the user.
func (u *User) InfectedFileCount() int {
	return u.infectedFileCount
}

// IncrementInfectedFileCount increments the infected file counter.
func (u *User) IncrementInfectedFileCount() {
	u.infectedFileCount++
	u.updatedAt = time.Now().UTC()
}

// CreatedAt returns when the user was created.
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns when the user was last updated.
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// Events returns the domain events that have occurred on this aggregate.
func (u *User) Events() []shared.DomainEvent {
	return u.events
}

// ClearEvents clears all domain events from this aggregate.
// This should be called after events have been dispatched.
func (u *User) ClearEvents() {
	u.events = []shared.DomainEvent{}
}

// UpdateProfile updates the user's display name and bio.
// Emits a UserProfileUpdated event.
func (u *User) UpdateProfile(displayName, bio string) error {
	// Validate display name length
	if len(displayName) > maxDisplayNameLength {
		return fmt.Errorf("display name cannot exceed %d characters", maxDisplayNameLength)
	}

	// Validate bio length
	if len(bio) > maxBioLength {
		return fmt.Errorf("bio cannot exceed %d characters", maxBioLength)
	}

	u.displayName = displayName
	u.bio = bio
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserProfileUpdated(u.id, u.displayName, u.bio))
	return nil
}

// ChangeRole changes the user's role.
// Emits a UserRoleChanged event.
func (u *User) ChangeRole(newRole Role) error {
	if !newRole.IsValid() {
		return fmt.Errorf("invalid role")
	}

	if u.role == newRole {
		return nil // No-op if role is the same
	}

	oldRole := u.role
	u.role = newRole
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserRoleChanged(u.id, oldRole, newRole))
	return nil
}

// Suspend suspends the user account.
// Emits a UserSuspended event.
func (u *User) Suspend(reason string) error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}

	if u.status == StatusSuspended {
		return nil // Already suspended, no-op
	}

	u.status = StatusSuspended
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserSuspended(u.id, reason))
	return nil
}

// Activate activates the user account.
// Emits a UserActivated event.
func (u *User) Activate() error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}

	if u.status == StatusActive {
		return nil // Already active, no-op
	}

	u.status = StatusActive
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserActivated(u.id))
	return nil
}

// VerifyPassword verifies that the given plaintext password matches the stored hash.
func (u *User) VerifyPassword(plaintext string) error {
	return u.passwordHash.Verify(plaintext)
}

// ChangePassword changes the user's password to the new hash.
// Emits a UserPasswordChanged event.
func (u *User) ChangePassword(newHash PasswordHash) error {
	if newHash.IsEmpty() {
		return fmt.Errorf("password hash is required")
	}

	u.passwordHash = newHash
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserPasswordChanged(u.id))
	return nil
}

// CanLogin returns true if the user can log in (status is active).
func (u *User) CanLogin() bool {
	return u.status.CanLogin()
}

// addEvent adds a domain event to the aggregate's event list.
func (u *User) addEvent(event shared.DomainEvent) {
	u.events = append(u.events, event)
}

// 2FA Methods

// TOTPSecret returns the user's TOTP secret (nil if not set up).
func (u *User) TOTPSecret() *TOTPSecret {
	return u.totpSecret
}

// BackupCodes returns the user's backup codes.
func (u *User) BackupCodes() []BackupCode {
	return u.backupCodes
}

// Devices returns the user's known devices.
func (u *User) Devices() []DeviceFingerprint {
	return u.devices
}

// IsTOTPEnabled returns whether 2FA is enabled and verified for this user.
func (u *User) IsTOTPEnabled() bool {
	return u.totpSecret != nil && u.totpSecret.IsEnabled()
}

// IsTOTPSetupPending returns whether 2FA setup was started but not verified.
func (u *User) IsTOTPSetupPending() bool {
	return u.totpSecret != nil && u.totpSecret.IsSetupPending()
}

// SetupTOTP initializes 2FA setup with the given encrypted secret.
// The user must verify their first TOTP code before 2FA is fully enabled.
func (u *User) SetupTOTP(encryptedSecret []byte) error {
	if u.IsTOTPEnabled() {
		return ErrTOTPAlreadyEnabled
	}

	secret, err := NewTOTPSecret(encryptedSecret, u.email.String())
	if err != nil {
		return err
	}

	u.totpSecret = &secret
	u.updatedAt = time.Now().UTC()
	return nil
}

// EnableTOTP enables 2FA after the user verifies their first TOTP code.
// Also generates backup codes if not already set.
// Emits a UserTOTPEnabled event.
func (u *User) EnableTOTP(backupCodes []BackupCode) error {
	if u.totpSecret == nil {
		return ErrTOTPNotEnabled
	}

	if u.IsTOTPEnabled() {
		return ErrTOTPAlreadyEnabled
	}

	u.totpSecret.Enable()
	u.backupCodes = backupCodes
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserTOTPEnabled(u.id))
	return nil
}

// DisableTOTP disables 2FA for the user.
// Emits a UserTOTPDisabled event.
func (u *User) DisableTOTP() error {
	if !u.IsTOTPEnabled() && !u.IsTOTPSetupPending() {
		return nil // Idempotent
	}

	u.totpSecret = nil
	u.backupCodes = nil
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserTOTPDisabled(u.id))
	return nil
}

// UseBackupCode validates and consumes a backup code.
// Returns an error if the code is invalid or already used.
// Emits a UserBackupCodeUsed event.
func (u *User) UseBackupCode(plaintext string) error {
	if !u.IsTOTPEnabled() {
		return ErrTOTPNotEnabled
	}

	if len(u.backupCodes) == 0 {
		return ErrBackupCodesExhausted
	}

	for i := range u.backupCodes {
		if u.backupCodes[i].IsUsed() {
			continue
		}

		if err := u.backupCodes[i].Verify(plaintext); err == nil {
			u.backupCodes[i].MarkUsed()
			u.updatedAt = time.Now().UTC()

			remaining := CountUnusedBackupCodes(u.backupCodes)
			u.addEvent(NewUserBackupCodeUsed(u.id, remaining))
			return nil
		}
	}

	return ErrBackupCodeInvalid
}

// RegenerateBackupCodes replaces all backup codes with new ones.
// Emits a UserBackupCodesRegenerated event.
func (u *User) RegenerateBackupCodes(newCodes []BackupCode) error {
	if !u.IsTOTPEnabled() {
		return ErrTOTPNotEnabled
	}

	u.backupCodes = newCodes
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserBackupCodesRegenerated(u.id))
	return nil
}

// UnusedBackupCodeCount returns the number of unused backup codes.
func (u *User) UnusedBackupCodeCount() int {
	return CountUnusedBackupCodes(u.backupCodes)
}

// TrackDevice records a login from a device.
// Returns true if this is a known device, false if unusual (new device).
func (u *User) TrackDevice(fingerprint DeviceFingerprint) bool {
	// Check if device already exists
	for i := range u.devices {
		if u.devices[i].MatchesHash(fingerprint.FingerprintHash()) {
			u.devices[i].UpdateLastSeen()
			return u.devices[i].IsTrusted()
		}
	}

	// New device - add to list
	u.devices = append(u.devices, fingerprint)
	u.updatedAt = time.Now().UTC()

	// Emit unusual login event
	u.addEvent(NewUserUnusualLogin(
		u.id,
		fingerprint.IPAddress(),
		fingerprint.DeviceName(),
		fingerprint.FingerprintHash(),
	))

	return false // New device is unusual
}

// TrustDevice marks a device as trusted.
// Emits a UserDeviceTrusted event.
func (u *User) TrustDevice(fingerprintHash string) error {
	for i := range u.devices {
		if u.devices[i].MatchesHash(fingerprintHash) {
			if u.devices[i].IsTrusted() {
				return nil // Already trusted, no-op
			}

			u.devices[i].MarkTrusted()
			u.updatedAt = time.Now().UTC()

			u.addEvent(NewUserDeviceTrusted(
				u.id,
				fingerprintHash,
				u.devices[i].DeviceName(),
			))
			return nil
		}
	}

	return fmt.Errorf("device not found")
}

// RemoveDevice removes a device from the trusted list.
func (u *User) RemoveDevice(fingerprintHash string) error {
	for i := range u.devices {
		if u.devices[i].MatchesHash(fingerprintHash) {
			u.devices = append(u.devices[:i], u.devices[i+1:]...)
			u.updatedAt = time.Now().UTC()
			return nil
		}
	}

	return fmt.Errorf("device not found")
}

// HasTrustedDevices returns whether the user has any trusted devices.
func (u *User) HasTrustedDevices() bool {
	for _, d := range u.devices {
		if d.IsTrusted() {
			return true
		}
	}
	return false
}

// Requires2FA returns whether 2FA verification is required for login.
// This is true if TOTP is enabled for this user.
func (u *User) Requires2FA() bool {
	return u.IsTOTPEnabled()
}

// Notification Preferences Methods

// NotificationPreferences returns the user's notification preferences.
func (u *User) NotificationPreferences() NotificationPreferences {
	return u.notificationPreferences
}

// UpdateNotificationPreferences updates the user's notification preferences.
// Emits a UserNotificationPreferencesUpdated event.
func (u *User) UpdateNotificationPreferences(prefs NotificationPreferences) {
	u.notificationPreferences = prefs
	u.updatedAt = time.Now().UTC()
	// Could emit an event here if needed for auditing
}

// Guest User Methods

// UserType returns the user's type (registered or guest).
func (u *User) UserType() UserType {
	return u.userType
}

// IPAddress returns the user's IP address (only for guest users, nil for registered).
func (u *User) IPAddress() *string {
	return u.ipAddress
}

// ExpiresAt returns the expiration time for guest users (nil for registered users).
func (u *User) ExpiresAt() *time.Time {
	return u.expiresAt
}

// IsGuest returns true if this is a guest user account.
func (u *User) IsGuest() bool {
	return u.userType.IsGuest()
}

// IsExpired returns true if this guest account has expired.
// Always returns false for registered users.
func (u *User) IsExpired() bool {
	if !u.IsGuest() || u.expiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.expiresAt)
}

// ConvertToRegistered converts a guest account to a registered account.
// This is used when a guest user claims their uploads and registers.
// Emits a GuestConvertedToRegistered event.
func (u *User) ConvertToRegistered(email Email, username Username, passwordHash PasswordHash) error {
	if !u.IsGuest() {
		return ErrUserNotGuest
	}

	if email.IsEmpty() {
		return fmt.Errorf("email is required")
	}
	if username.IsEmpty() {
		return fmt.Errorf("username is required")
	}
	if passwordHash.IsEmpty() {
		return fmt.Errorf("password hash is required")
	}

	oldUserID := u.id
	u.email = email
	u.username = username
	u.passwordHash = passwordHash
	u.userType = UserTypeRegistered
	u.ipAddress = nil
	u.expiresAt = nil
	u.status = StatusPending // Require email verification
	u.displayName = username.String()
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewGuestConvertedToRegistered(oldUserID, u.id, email, username))
	return nil
}
