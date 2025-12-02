package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Role
		wantErr bool
	}{
		{
			name:    "valid user role",
			input:   "user",
			want:    RoleUser,
			wantErr: false,
		},
		{
			name:    "valid moderator role",
			input:   "moderator",
			want:    RoleModerator,
			wantErr: false,
		},
		{
			name:    "valid admin role",
			input:   "admin",
			want:    RoleAdmin,
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

			role, err := ParseRole(tt.input)

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
		role Role
		want string
	}{
		{
			name: "user role",
			role: RoleUser,
			want: "user",
		},
		{
			name: "moderator role",
			role: RoleModerator,
			want: "moderator",
		},
		{
			name: "admin role",
			role: RoleAdmin,
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
		role Role
		want bool
	}{
		{
			name: "user is valid",
			role: RoleUser,
			want: true,
		},
		{
			name: "moderator is valid",
			role: RoleModerator,
			want: true,
		},
		{
			name: "admin is valid",
			role: RoleAdmin,
			want: true,
		},
		{
			name: "empty is invalid",
			role: Role(""),
			want: false,
		},
		{
			name: "random string is invalid",
			role: Role("superuser"),
			want: false,
		},
		{
			name: "uppercase is invalid",
			role: Role("USER"),
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
		role Role
		want bool
	}{
		{
			name: "user cannot moderate",
			role: RoleUser,
			want: false,
		},
		{
			name: "moderator can moderate",
			role: RoleModerator,
			want: true,
		},
		{
			name: "admin can moderate",
			role: RoleAdmin,
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
		role Role
		want bool
	}{
		{
			name: "user is not admin",
			role: RoleUser,
			want: false,
		},
		{
			name: "moderator is not admin",
			role: RoleModerator,
			want: false,
		},
		{
			name: "admin is admin",
			role: RoleAdmin,
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
