package identity_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:    "valid username",
			input:   "johndoe",
			want:    "johndoe",
			wantErr: nil,
		},
		{
			name:    "valid with numbers",
			input:   "user123",
			want:    "user123",
			wantErr: nil,
		},
		{
			name:    "valid with underscores",
			input:   "john_doe_123",
			want:    "john_doe_123",
			wantErr: nil,
		},
		{
			name:    "valid mixed case preserved",
			input:   "JohnDoe",
			want:    "JohnDoe",
			wantErr: nil,
		},
		{
			name:    "exactly 3 characters",
			input:   "abc",
			want:    "abc",
			wantErr: nil,
		},
		{
			name:    "exactly 32 characters",
			input:   strings.Repeat("a", 32),
			want:    strings.Repeat("a", 32),
			wantErr: nil,
		},
		{
			name:    "whitespace trimmed",
			input:   "  johndoe  ",
			want:    "johndoe",
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: identity.ErrUsernameEmpty,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: identity.ErrUsernameEmpty,
		},
		{
			name:    "too short - 2 characters",
			input:   "ab",
			wantErr: identity.ErrUsernameTooShort,
		},
		{
			name:    "too short - 1 character",
			input:   "a",
			wantErr: identity.ErrUsernameTooShort,
		},
		{
			name:    "too long - 33 characters",
			input:   strings.Repeat("a", 33),
			wantErr: identity.ErrUsernameTooLong,
		},
		{
			name:    "contains spaces",
			input:   "john doe",
			wantErr: identity.ErrUsernameInvalid,
		},
		{
			name:    "contains hyphen",
			input:   "john-doe",
			wantErr: identity.ErrUsernameInvalid,
		},
		{
			name:    "contains dot",
			input:   "john.doe",
			wantErr: identity.ErrUsernameInvalid,
		},
		{
			name:    "contains special characters",
			input:   "john@doe",
			wantErr: identity.ErrUsernameInvalid,
		},
		{
			name:    "contains emoji",
			input:   "john😀",
			wantErr: identity.ErrUsernameInvalid,
		},
		{
			name:    "reserved - admin",
			input:   "admin",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - root",
			input:   "root",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - system",
			input:   "system",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - administrator",
			input:   "administrator",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - moderator",
			input:   "moderator",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - case insensitive ADMIN",
			input:   "ADMIN",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - case insensitive Admin",
			input:   "Admin",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - support",
			input:   "support",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - guest",
			input:   "guest",
			wantErr: identity.ErrUsernameReserved,
		},
		{
			name:    "reserved - anonymous",
			input:   "anonymous",
			wantErr: identity.ErrUsernameReserved,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			username, err := identity.NewUsername(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, username.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, username.String())
				assert.False(t, username.IsEmpty())
			}
		})
	}
}

func TestUsername_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("zero value is empty", func(t *testing.T) {
		t.Parallel()

		var username identity.Username
		assert.True(t, username.IsEmpty())
	})

	t.Run("valid username is not empty", func(t *testing.T) {
		t.Parallel()

		username, err := identity.NewUsername("johndoe")
		require.NoError(t, err)

		assert.False(t, username.IsEmpty())
	})
}

func TestUsername_Equals(t *testing.T) {
	t.Parallel()

	t.Run("same usernames are equal", func(t *testing.T) {
		t.Parallel()

		username1, err := identity.NewUsername("johndoe")
		require.NoError(t, err)
		username2, err := identity.NewUsername("johndoe")
		require.NoError(t, err)

		assert.True(t, username1.Equals(username2))
		assert.True(t, username2.Equals(username1))
	})

	t.Run("different usernames are not equal", func(t *testing.T) {
		t.Parallel()

		username1, err := identity.NewUsername("johndoe")
		require.NoError(t, err)
		username2, err := identity.NewUsername("janedoe")
		require.NoError(t, err)

		assert.False(t, username1.Equals(username2))
	})

	t.Run("different case usernames are not equal", func(t *testing.T) {
		t.Parallel()

		username1, err := identity.NewUsername("JohnDoe")
		require.NoError(t, err)
		username2, err := identity.NewUsername("johndoe")
		require.NoError(t, err)

		assert.False(t, username1.Equals(username2))
	})

	t.Run("zero values are equal", func(t *testing.T) {
		t.Parallel()

		var username1, username2 identity.Username
		assert.True(t, username1.Equals(username2))
	})
}

func TestUsername_String(t *testing.T) {
	t.Parallel()

	username, err := identity.NewUsername("johndoe")
	require.NoError(t, err)

	assert.Equal(t, "johndoe", username.String())
}
