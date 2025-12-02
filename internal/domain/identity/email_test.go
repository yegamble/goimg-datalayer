package identity_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:    "valid email",
			input:   "user@example.com",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			input:   "user@mail.example.com",
			want:    "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with plus",
			input:   "user+tag@example.com",
			want:    "user+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with dots",
			input:   "first.last@example.com",
			want:    "first.last@example.com",
			wantErr: nil,
		},
		{
			name:    "whitespace trimmed",
			input:   "  user@example.com  ",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "uppercase normalized",
			input:   "User@Example.COM",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: identity.ErrEmailEmpty,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: identity.ErrEmailEmpty,
		},
		{
			name:    "missing @",
			input:   "notanemail",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "missing local part",
			input:   "@example.com",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "missing domain",
			input:   "user@",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "missing TLD",
			input:   "user@example",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "invalid characters",
			input:   "user name@example.com",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 250) + "@test.com",
			wantErr: identity.ErrEmailTooLong,
		},
		{
			name:    "exactly 255 characters",
			input:   strings.Repeat("a", 243) + "@test.com",
			want:    strings.Repeat("a", 243) + "@test.com",
			wantErr: nil,
		},
		{
			name:    "disposable email - mailinator",
			input:   "user@mailinator.com",
			wantErr: identity.ErrEmailDisposable,
		},
		{
			name:    "disposable email - tempmail",
			input:   "user@tempmail.com",
			wantErr: identity.ErrEmailDisposable,
		},
		{
			name:    "disposable email - guerrillamail",
			input:   "user@guerrillamail.com",
			wantErr: identity.ErrEmailDisposable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			email, err := identity.NewEmail(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, email.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, email.String())
				assert.False(t, email.IsEmpty())
			}
		})
	}
}

func TestEmail_Domain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "simple domain",
			email: "user@example.com",
			want:  "example.com",
		},
		{
			name:  "subdomain",
			email: "user@mail.example.com",
			want:  "mail.example.com",
		},
		{
			name:  "multiple subdomains",
			email: "user@a.b.c.example.com",
			want:  "a.b.c.example.com",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			email, err := identity.NewEmail(tt.email)
			require.NoError(t, err)

			assert.Equal(t, tt.want, email.Domain())
		})
	}
}

func TestEmail_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("zero value is empty", func(t *testing.T) {
		t.Parallel()

		var email identity.Email
		assert.True(t, email.IsEmpty())
	})

	t.Run("valid email is not empty", func(t *testing.T) {
		t.Parallel()

		email, err := identity.NewEmail("user@example.com")
		require.NoError(t, err)

		assert.False(t, email.IsEmpty())
	})
}

//nolint:dupl // Value object Equals tests follow intentionally similar patterns
func TestEmail_Equals(t *testing.T) {
	t.Parallel()

	t.Run("same emails are equal", func(t *testing.T) {
		t.Parallel()

		email1, err := identity.NewEmail("user@example.com")
		require.NoError(t, err)
		email2, err := identity.NewEmail("user@example.com")
		require.NoError(t, err)

		assert.True(t, email1.Equals(email2))
		assert.True(t, email2.Equals(email1))
	})

	t.Run("different emails are not equal", func(t *testing.T) {
		t.Parallel()

		email1, err := identity.NewEmail("user1@example.com")
		require.NoError(t, err)
		email2, err := identity.NewEmail("user2@example.com")
		require.NoError(t, err)

		assert.False(t, email1.Equals(email2))
	})

	t.Run("normalized emails are equal", func(t *testing.T) {
		t.Parallel()

		email1, err := identity.NewEmail("User@Example.COM")
		require.NoError(t, err)
		email2, err := identity.NewEmail("user@example.com")
		require.NoError(t, err)

		assert.True(t, email1.Equals(email2))
	})

	t.Run("zero values are equal", func(t *testing.T) {
		t.Parallel()

		var email1, email2 identity.Email
		assert.True(t, email1.Equals(email2))
	})
}

func TestEmail_String(t *testing.T) {
	t.Parallel()

	email, err := identity.NewEmail("user@example.com")
	require.NoError(t, err)

	assert.Equal(t, "user@example.com", email.String())
}
