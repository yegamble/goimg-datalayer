package identity

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	maxDisplayNameLength = 100
	maxBioLength         = 500
)

type User struct {
	id                UserID
	email             Email
	username          Username
	passwordHash      PasswordHash
	role              Role
	status            UserStatus
	displayName       string
	bio               string
	infectedFileCount int
	createdAt         time.Time
	updatedAt         time.Time
	events            []shared.DomainEvent

	totpSecret  *TOTPSecret
	backupCodes []BackupCode
	devices     []DeviceFingerprint

	notificationPreferences NotificationPreferences

	userType  UserType
	ipAddress *string
	expiresAt *time.Time

	emailVerified   bool
	emailVerifiedAt *time.Time
}

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
		displayName:             username.String(),
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

func NewGuestUser(ipAddress string) (*User, error) {
	if ipAddress == "" {
		return nil, fmt.Errorf("ip address is required for guest users")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour)

	userID := NewUserID()
	guestUsername := fmt.Sprintf("guest_%s", userID.String()[:8])
	username, err := NewUsername(guestUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest username: %w", err)
	}

	guestEmail := fmt.Sprintf("guest_%s@goimg.local", userID.String())
	email, err := NewEmail(guestEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest email: %w", err)
	}

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
	emailVerified bool,
	emailVerifiedAt *time.Time,
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
		emailVerified:           emailVerified,
		emailVerifiedAt:         emailVerifiedAt,
	}
}

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
	emailVerified bool,
	emailVerifiedAt *time.Time,
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
		emailVerified:           emailVerified,
		emailVerifiedAt:         emailVerifiedAt,
	}
}

func (u *User) ID() UserID {
	return u.id
}

func (u *User) Email() Email {
	return u.email
}

func (u *User) Username() Username {
	return u.username
}

func (u *User) PasswordHash() PasswordHash {
	return u.passwordHash
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) Status() UserStatus {
	return u.status
}

func (u *User) DisplayName() string {
	return u.displayName
}

func (u *User) Bio() string {
	return u.bio
}

func (u *User) InfectedFileCount() int {
	return u.infectedFileCount
}

func (u *User) IncrementInfectedFileCount() {
	u.infectedFileCount++
	u.updatedAt = time.Now().UTC()
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *User) Events() []shared.DomainEvent {
	return u.events
}

func (u *User) ClearEvents() {
	u.events = []shared.DomainEvent{}
}

func (u *User) UpdateProfile(displayName, bio string) error {
	if len(displayName) > maxDisplayNameLength {
		return fmt.Errorf("display name cannot exceed %d characters", maxDisplayNameLength)
	}

	if len(bio) > maxBioLength {
		return fmt.Errorf("bio cannot exceed %d characters", maxBioLength)
	}

	u.displayName = displayName
	u.bio = bio
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserProfileUpdated(u.id, u.displayName, u.bio))
	return nil
}

func (u *User) ChangeRole(newRole Role) error {
	if !newRole.IsValid() {
		return fmt.Errorf("invalid role")
	}

	if u.role == newRole {
		return nil
	}

	oldRole := u.role
	u.role = newRole
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserRoleChanged(u.id, oldRole, newRole))
	return nil
}

func (u *User) Suspend(reason string) error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}

	if u.status == StatusSuspended {
		return nil
	}

	u.status = StatusSuspended
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserSuspended(u.id, reason))
	return nil
}

func (u *User) Activate() error {
	if u.status == StatusDeleted {
		return ErrUserDeleted
	}

	if u.status == StatusActive {
		return nil
	}

	u.status = StatusActive
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserActivated(u.id))
	return nil
}

func (u *User) VerifyPassword(plaintext string) error {
	return u.passwordHash.Verify(plaintext)
}

func (u *User) ChangePassword(newHash PasswordHash) error {
	if newHash.IsEmpty() {
		return fmt.Errorf("password hash is required")
	}

	u.passwordHash = newHash
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserPasswordChanged(u.id))
	return nil
}

func (u *User) ChangeEmail(newEmail Email) error {
	if newEmail.IsEmpty() {
		return fmt.Errorf("email cannot be empty")
	}
	if u.email == newEmail {
		return nil
	}

	u.email = newEmail
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) CanLogin() bool {
	return u.status.CanLogin()
}

func (u *User) addEvent(event shared.DomainEvent) {
	u.events = append(u.events, event)
}

func (u *User) TOTPSecret() *TOTPSecret {
	return u.totpSecret
}

func (u *User) BackupCodes() []BackupCode {
	return u.backupCodes
}

func (u *User) Devices() []DeviceFingerprint {
	return u.devices
}

func (u *User) IsTOTPEnabled() bool {
	return u.totpSecret != nil && u.totpSecret.IsEnabled()
}

func (u *User) IsTOTPSetupPending() bool {
	return u.totpSecret != nil && u.totpSecret.IsSetupPending()
}

func (u *User) SetupTOTP(encryptedSecret []byte) error {
	if u.IsTOTPEnabled() {
		return ErrTOTPAlreadyEnabled
	}

	secret, err := NewTOTPSecret(encryptedSecret, u.email.String())
	if err != nil {
		return fmt.Errorf("create totp secret: %w", err)
	}

	u.totpSecret = &secret
	u.updatedAt = time.Now().UTC()
	return nil
}

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

func (u *User) DisableTOTP() error {
	if !u.IsTOTPEnabled() && !u.IsTOTPSetupPending() {
		return nil
	}

	u.totpSecret = nil
	u.backupCodes = nil
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserTOTPDisabled(u.id))
	return nil
}

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

func (u *User) RegenerateBackupCodes(newCodes []BackupCode) error {
	if !u.IsTOTPEnabled() {
		return ErrTOTPNotEnabled
	}

	u.backupCodes = newCodes
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserBackupCodesRegenerated(u.id))
	return nil
}

func (u *User) UnusedBackupCodeCount() int {
	return CountUnusedBackupCodes(u.backupCodes)
}

func (u *User) TrackDevice(fingerprint DeviceFingerprint) bool {
	for i := range u.devices {
		if u.devices[i].MatchesHash(fingerprint.FingerprintHash()) {
			u.devices[i].UpdateLastSeen()
			return u.devices[i].IsTrusted()
		}
	}

	u.devices = append(u.devices, fingerprint)
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewUserUnusualLogin(
		u.id,
		fingerprint.IPAddress(),
		fingerprint.DeviceName(),
		fingerprint.FingerprintHash(),
	))

	return false
}

func (u *User) TrustDevice(fingerprintHash string) error {
	for i := range u.devices {
		if u.devices[i].MatchesHash(fingerprintHash) {
			if u.devices[i].IsTrusted() {
				return nil
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

func (u *User) HasTrustedDevices() bool {
	for _, d := range u.devices {
		if d.IsTrusted() {
			return true
		}
	}
	return false
}

func (u *User) Requires2FA() bool {
	return u.IsTOTPEnabled()
}

func (u *User) NotificationPreferences() NotificationPreferences {
	return u.notificationPreferences
}

func (u *User) UpdateNotificationPreferences(prefs NotificationPreferences) {
	u.notificationPreferences = prefs
	u.updatedAt = time.Now().UTC()
}

func (u *User) UserType() UserType {
	return u.userType
}

func (u *User) IPAddress() *string {
	return u.ipAddress
}

func (u *User) ExpiresAt() *time.Time {
	return u.expiresAt
}

func (u *User) IsGuest() bool {
	return u.userType.IsGuest()
}

func (u *User) IsExpired() bool {
	if !u.IsGuest() || u.expiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.expiresAt)
}

func (u *User) EmailVerified() bool {
	return u.emailVerified
}

func (u *User) EmailVerifiedAt() *time.Time {
	return u.emailVerifiedAt
}

func (u *User) VerifyEmail() {
	if u.emailVerified {
		return
	}
	now := time.Now().UTC()
	u.emailVerified = true
	u.emailVerifiedAt = &now
	u.updatedAt = now
}

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
	u.status = StatusPending
	u.displayName = username.String()
	u.updatedAt = time.Now().UTC()

	u.addEvent(NewGuestConvertedToRegistered(oldUserID, u.id, email, username))
	return nil
}
