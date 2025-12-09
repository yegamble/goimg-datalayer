package fixtures

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// UserFixture provides test user data and factory functions.
type UserFixture struct {
	ID           uuid.UUID
	Email        string
	Username     string
	Password     string
	Role         identity.Role
	Status       identity.UserStatus
	DisplayName  string
	Bio          string
	PasswordHash identity.PasswordHash
}

// ValidUser returns a valid user fixture with sensible defaults.
func ValidUser(t *testing.T) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("Password123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Username:     "testuser",
		Password:     "Password123!",
		Role:         identity.RoleUser,
		Status:       identity.StatusActive,
		DisplayName:  "Test User",
		Bio:          "This is a test user",
		PasswordHash: passwordHash,
	}
}

// AdminUser returns a user fixture with admin role.
func AdminUser(t *testing.T) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("AdminPass123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		Username:     "admin",
		Password:     "AdminPass123!",
		Role:         identity.RoleAdmin,
		Status:       identity.StatusActive,
		DisplayName:  "Admin User",
		Bio:          "Administrator account",
		PasswordHash: passwordHash,
	}
}

// ModeratorUser returns a user fixture with moderator role.
func ModeratorUser(t *testing.T) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("ModPass123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        "moderator@example.com",
		Username:     "moderator",
		Password:     "ModPass123!",
		Role:         identity.RoleModerator,
		Status:       identity.StatusActive,
		DisplayName:  "Moderator User",
		Bio:          "Moderator account",
		PasswordHash: passwordHash,
	}
}

// SuspendedUser returns a user fixture with suspended status.
func SuspendedUser(t *testing.T) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("Password123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        "suspended@example.com",
		Username:     "suspended",
		Password:     "Password123!",
		Role:         identity.RoleUser,
		Status:       identity.StatusSuspended,
		DisplayName:  "Suspended User",
		Bio:          "This user is suspended",
		PasswordHash: passwordHash,
	}
}

// PendingUser returns a user fixture with pending status.
func PendingUser(t *testing.T) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("Password123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        "pending@example.com",
		Username:     "pending",
		Password:     "Password123!",
		Role:         identity.RoleUser,
		Status:       identity.StatusPending,
		DisplayName:  "Pending User",
		Bio:          "This user is pending verification",
		PasswordHash: passwordHash,
	}
}

// ToEntity converts the fixture to a domain User entity.
func (f *UserFixture) ToEntity(t *testing.T) *identity.User {
	t.Helper()

	email, err := identity.NewEmail(f.Email)
	require.NoError(t, err)

	username, err := identity.NewUsername(f.Username)
	require.NoError(t, err)

	// Use NewUser for initial creation
	user, err := identity.NewUser(email, username, f.PasswordHash)
	require.NoError(t, err)

	// Clear events since we're creating a fixture, not a real user creation
	user.ClearEvents()

	return user
}

// WithEmail returns a copy of the fixture with a custom email.
func (f *UserFixture) WithEmail(email string) *UserFixture {
	clone := *f
	clone.Email = email
	return &clone
}

// WithUsername returns a copy of the fixture with a custom username.
func (f *UserFixture) WithUsername(username string) *UserFixture {
	clone := *f
	clone.Username = username
	return &clone
}

// WithRole returns a copy of the fixture with a custom role.
func (f *UserFixture) WithRole(role identity.Role) *UserFixture {
	clone := *f
	clone.Role = role
	return &clone
}

// WithStatus returns a copy of the fixture with a custom status.
func (f *UserFixture) WithStatus(status identity.UserStatus) *UserFixture {
	clone := *f
	clone.Status = status
	return &clone
}

// UniqueUser generates a user fixture with unique email and username.
// Useful for tests that need multiple unique users.
func UniqueUser(t *testing.T, prefix string) *UserFixture {
	t.Helper()

	passwordHash, err := identity.NewPasswordHash("Password123!")
	require.NoError(t, err)

	return &UserFixture{
		ID:           uuid.New(),
		Email:        prefix + "@example.com",
		Username:     prefix,
		Password:     "Password123!",
		Role:         identity.RoleUser,
		Status:       identity.StatusActive,
		DisplayName:  prefix + " User",
		Bio:          "Unique test user",
		PasswordHash: passwordHash,
	}
}
