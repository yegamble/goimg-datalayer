package identity

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
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
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
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
		id:           NewUserID(),
		email:        email,
		username:     username,
		passwordHash: passwordHash,
		role:         RoleUser,
		status:       StatusPending,
		displayName:  username.String(), // Default display name is username
		bio:          "",
		createdAt:    now,
		updatedAt:    now,
		events:       []shared.DomainEvent{},
	}

	user.addEvent(NewUserCreated(user.id, user.email, user.username))
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
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		username:     username,
		passwordHash: passwordHash,
		role:         role,
		status:       status,
		displayName:  displayName,
		bio:          bio,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       []shared.DomainEvent{},
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
	if len(displayName) > 100 {
		return fmt.Errorf("display name cannot exceed 100 characters")
	}

	// Validate bio length
	if len(bio) > 500 {
		return fmt.Errorf("bio cannot exceed 500 characters")
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
