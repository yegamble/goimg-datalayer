package identity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestParseRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    identity.Role
		wantErr bool
	}{
		{
			name:    "valid user role",
			input:   "user",
			want:    identity.RoleUser,
			wantErr: false,
		},
		{
			name:    "valid moderator role",
			input:   "moderator",
			want:    identity.RoleModerator,
			wantErr: false,
		},
		{
			name:    "valid admin role",
			input:   "admin",
			want:    identity.RoleAdmin,
			wantErr: false,
		},
		{
			name:    "invalid role",
			input:   "superuser",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "uppercase not valid",
			input:   "USER",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := identity.ParseRole(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, role)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, role)
			}
		})
	}
}

func TestRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role identity.Role
		want string
	}{
		{
			name: "user role",
			role: identity.RoleUser,
			want: "user",
		},
		{
			name: "moderator role",
			role: identity.RoleModerator,
			want: "moderator",
		},
		{
			name: "admin role",
			role: identity.RoleAdmin,
			want: "admin",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.role.String())
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role identity.Role
		want bool
	}{
		{
			name: "user is valid",
			role: identity.RoleUser,
			want: true,
		},
		{
			name: "moderator is valid",
			role: identity.RoleModerator,
			want: true,
		},
		{
			name: "admin is valid",
			role: identity.RoleAdmin,
			want: true,
		},
		{
			name: "empty is invalid",
			role: identity.Role(""),
			want: false,
		},
		{
			name: "random string is invalid",
			role: identity.Role("superuser"),
			want: false,
		},
		{
			name: "uppercase is invalid",
			role: identity.Role("USER"),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.role.IsValid())
		})
	}
}

func TestRole_CanModerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role identity.Role
		want bool
	}{
		{
			name: "user cannot moderate",
			role: identity.RoleUser,
			want: false,
		},
		{
			name: "moderator can moderate",
			role: identity.RoleModerator,
			want: true,
		},
		{
			name: "admin can moderate",
			role: identity.RoleAdmin,
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.role.CanModerate())
		})
	}
}

func TestRole_IsAdmin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role identity.Role
		want bool
	}{
		{
			name: "user is not admin",
			role: identity.RoleUser,
			want: false,
		},
		{
			name: "moderator is not admin",
			role: identity.RoleModerator,
			want: false,
		},
		{
			name: "admin is admin",
			role: identity.RoleAdmin,
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.role.IsAdmin())
		})
	}
}
