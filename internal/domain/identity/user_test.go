package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("creates user with valid inputs", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)

		require.NoError(t, err)
		assert.False(t, user.ID().IsZero())
		assert.Equal(t, email, user.Email())
		assert.Equal(t, username, user.Username())
		assert.Equal(t, RoleUser, user.Role())
		assert.Equal(t, StatusPending, user.Status())
		assert.Equal(t, username.String(), user.DisplayName())
		assert.Empty(t, user.Bio())
		assert.False(t, user.CreatedAt().IsZero())
		assert.False(t, user.UpdatedAt().IsZero())
		assert.Len(t, user.Events(), 1)
	})

	t.Run("emits UserCreated event", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		events := user.Events()
		require.Len(t, events, 1)

		event, ok := events[0].(UserCreated)
		require.True(t, ok)
		assert.Equal(t, "identity.user.created", event.EventType())
		assert.Equal(t, user.ID(), event.UserID)
		assert.Equal(t, email, event.Email)
		assert.Equal(t, username, event.Username)
	})

	t.Run("fails with empty email", func(t *testing.T) {
		t.Parallel()

		var emptyEmail Email
		_, err := NewUser(emptyEmail, username, passwordHash)

		require.Error(t, err)
	})

	t.Run("fails with empty username", func(t *testing.T) {
		t.Parallel()

		var emptyUsername Username
		_, err := NewUser(email, emptyUsername, passwordHash)

		require.Error(t, err)
	})

	t.Run("fails with empty password hash", func(t *testing.T) {
		t.Parallel()

		var emptyHash PasswordHash
		_, err := NewUser(email, username, emptyHash)

		require.Error(t, err)
	})
}

func TestReconstructUser(t *testing.T) {
	t.Parallel()

	id := NewUserID()
	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")
	role := RoleAdmin
	status := StatusActive
	displayName := "Test User"
	bio := "This is a test bio"
	createdAt := time.Now().UTC().Add(-24 * time.Hour)
	updatedAt := time.Now().UTC()

	user := ReconstructUser(id, email, username, passwordHash, role, status, displayName, bio, createdAt, updatedAt)

	assert.Equal(t, id, user.ID())
	assert.Equal(t, email, user.Email())
	assert.Equal(t, username, user.Username())
	assert.Equal(t, role, user.Role())
	assert.Equal(t, status, user.Status())
	assert.Equal(t, displayName, user.DisplayName())
	assert.Equal(t, bio, user.Bio())
	assert.Equal(t, createdAt, user.CreatedAt())
	assert.Equal(t, updatedAt, user.UpdatedAt())
	assert.Len(t, user.Events(), 0) // No events on reconstruction
}

func TestUser_UpdateProfile(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("updates profile successfully", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		err = user.UpdateProfile("New Display Name", "New bio")
		require.NoError(t, err)

		assert.Equal(t, "New Display Name", user.DisplayName())
		assert.Equal(t, "New bio", user.Bio())
		assert.Len(t, user.Events(), 1)

		event, ok := user.Events()[0].(UserProfileUpdated)
		require.True(t, ok)
		assert.Equal(t, "identity.user.profile_updated", event.EventType())
	})

	t.Run("fails with display name too long", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		longName := string(make([]byte, 101))
		err = user.UpdateProfile(longName, "Bio")

		require.Error(t, err)
	})

	t.Run("fails with bio too long", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		longBio := string(make([]byte, 501))
		err = user.UpdateProfile("Name", longBio)

		require.Error(t, err)
	})
}

func TestUser_ChangeRole(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("changes role successfully", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		err = user.ChangeRole(RoleAdmin)
		require.NoError(t, err)

		assert.Equal(t, RoleAdmin, user.Role())
		assert.Len(t, user.Events(), 1)

		event, ok := user.Events()[0].(UserRoleChanged)
		require.True(t, ok)
		assert.Equal(t, "identity.user.role_changed", event.EventType())
		assert.Equal(t, RoleUser, event.OldRole)
		assert.Equal(t, RoleAdmin, event.NewRole)
	})

	t.Run("no-op when role is the same", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		err = user.ChangeRole(RoleUser)
		require.NoError(t, err)

		assert.Len(t, user.Events(), 0)
	})

	t.Run("fails with invalid role", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		err = user.ChangeRole(Role("invalid"))
		require.Error(t, err)
	})
}

func TestUser_Suspend(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("suspends user successfully", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		err = user.Suspend("Violation of terms")
		require.NoError(t, err)

		assert.Equal(t, StatusSuspended, user.Status())
		assert.Len(t, user.Events(), 1)

		event, ok := user.Events()[0].(UserSuspended)
		require.True(t, ok)
		assert.Equal(t, "identity.user.suspended", event.EventType())
		assert.Equal(t, "Violation of terms", event.Reason)
	})

	t.Run("no-op when already suspended", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		err = user.Suspend("First suspension")
		require.NoError(t, err)
		user.ClearEvents()

		err = user.Suspend("Second suspension")
		require.NoError(t, err)

		assert.Len(t, user.Events(), 0)
	})

	t.Run("fails when user is deleted", func(t *testing.T) {
		t.Parallel()

		user := ReconstructUser(
			NewUserID(), email, username, passwordHash,
			RoleUser, StatusDeleted, "Test", "", time.Now(), time.Now(),
		)

		err := user.Suspend("Reason")
		require.ErrorIs(t, err, ErrUserDeleted)
	})
}

func TestUser_Activate(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("activates user successfully", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		err = user.Activate()
		require.NoError(t, err)

		assert.Equal(t, StatusActive, user.Status())
		assert.Len(t, user.Events(), 1)

		event, ok := user.Events()[0].(UserActivated)
		require.True(t, ok)
		assert.Equal(t, "identity.user.activated", event.EventType())
	})

	t.Run("no-op when already active", func(t *testing.T) {
		t.Parallel()

		user := ReconstructUser(
			NewUserID(), email, username, passwordHash,
			RoleUser, StatusActive, "Test", "", time.Now(), time.Now(),
		)
		user.ClearEvents()

		err := user.Activate()
		require.NoError(t, err)

		assert.Len(t, user.Events(), 0)
	})

	t.Run("fails when user is deleted", func(t *testing.T) {
		t.Parallel()

		user := ReconstructUser(
			NewUserID(), email, username, passwordHash,
			RoleUser, StatusDeleted, "Test", "", time.Now(), time.Now(),
		)

		err := user.Activate()
		require.ErrorIs(t, err, ErrUserDeleted)
	})
}

func TestUser_VerifyPassword(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	password := "SecureP@ssw0rd123"
	passwordHash, _ := NewPasswordHash(password)

	t.Run("verifies correct password", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		err = user.VerifyPassword(password)
		assert.NoError(t, err)
	})

	t.Run("fails with incorrect password", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		err = user.VerifyPassword("WrongPassword123")
		require.ErrorIs(t, err, ErrPasswordMismatch)
	})
}

func TestUser_ChangePassword(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	t.Run("changes password successfully", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)
		user.ClearEvents()

		newHash, _ := NewPasswordHash("NewSecureP@ssw0rd456")
		err = user.ChangePassword(newHash)
		require.NoError(t, err)

		// Verify old password no longer works
		err = user.VerifyPassword("SecureP@ssw0rd123")
		require.ErrorIs(t, err, ErrPasswordMismatch)

		// Verify new password works
		err = user.VerifyPassword("NewSecureP@ssw0rd456")
		assert.NoError(t, err)

		assert.Len(t, user.Events(), 1)
		event, ok := user.Events()[0].(UserPasswordChanged)
		require.True(t, ok)
		assert.Equal(t, "identity.user.password_changed", event.EventType())
	})

	t.Run("fails with empty password hash", func(t *testing.T) {
		t.Parallel()

		user, err := NewUser(email, username, passwordHash)
		require.NoError(t, err)

		var emptyHash PasswordHash
		err = user.ChangePassword(emptyHash)
		require.Error(t, err)
	})
}

func TestUser_CanLogin(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	tests := []struct {
		name   string
		status UserStatus
		want   bool
	}{
		{
			name:   "active user can login",
			status: StatusActive,
			want:   true,
		},
		{
			name:   "pending user cannot login",
			status: StatusPending,
			want:   false,
		},
		{
			name:   "suspended user cannot login",
			status: StatusSuspended,
			want:   false,
		},
		{
			name:   "deleted user cannot login",
			status: StatusDeleted,
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			user := ReconstructUser(
				NewUserID(), email, username, passwordHash,
				RoleUser, tt.status, "Test", "", time.Now(), time.Now(),
			)

			assert.Equal(t, tt.want, user.CanLogin())
		})
	}
}

func TestUser_ClearEvents(t *testing.T) {
	t.Parallel()

	email, _ := NewEmail("test@example.com")
	username, _ := NewUsername("testuser")
	passwordHash, _ := NewPasswordHash("SecureP@ssw0rd123")

	user, err := NewUser(email, username, passwordHash)
	require.NoError(t, err)

	assert.Len(t, user.Events(), 1)

	user.ClearEvents()
	assert.Len(t, user.Events(), 0)
}
