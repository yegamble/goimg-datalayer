package identity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestParseUserStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    identity.UserStatus
		wantErr bool
	}{
		{
			name:    "valid active status",
			input:   "active",
			want:    identity.StatusActive,
			wantErr: false,
		},
		{
			name:    "valid pending status",
			input:   "pending",
			want:    identity.StatusPending,
			wantErr: false,
		},
		{
			name:    "valid suspended status",
			input:   "suspended",
			want:    identity.StatusSuspended,
			wantErr: false,
		},
		{
			name:    "valid deleted status",
			input:   "deleted",
			want:    identity.StatusDeleted,
			wantErr: false,
		},
		{
			name:    "invalid status",
			input:   "inactive",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "uppercase not valid",
			input:   "ACTIVE",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, err := identity.ParseUserStatus(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, status)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, status)
			}
		})
	}
}

func TestUserStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status identity.UserStatus
		want   string
	}{
		{
			name:   "active status",
			status: identity.StatusActive,
			want:   "active",
		},
		{
			name:   "pending status",
			status: identity.StatusPending,
			want:   "pending",
		},
		{
			name:   "suspended status",
			status: identity.StatusSuspended,
			want:   "suspended",
		},
		{
			name:   "deleted status",
			status: identity.StatusDeleted,
			want:   "deleted",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestUserStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status identity.UserStatus
		want   bool
	}{
		{
			name:   "active is valid",
			status: identity.StatusActive,
			want:   true,
		},
		{
			name:   "pending is valid",
			status: identity.StatusPending,
			want:   true,
		},
		{
			name:   "suspended is valid",
			status: identity.StatusSuspended,
			want:   true,
		},
		{
			name:   "deleted is valid",
			status: identity.StatusDeleted,
			want:   true,
		},
		{
			name:   "empty is invalid",
			status: identity.UserStatus(""),
			want:   false,
		},
		{
			name:   "random string is invalid",
			status: identity.UserStatus("inactive"),
			want:   false,
		},
		{
			name:   "uppercase is invalid",
			status: identity.UserStatus("ACTIVE"),
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestUserStatus_CanLogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status identity.UserStatus
		want   bool
	}{
		{
			name:   "active can login",
			status: identity.StatusActive,
			want:   true,
		},
		{
			name:   "pending cannot login",
			status: identity.StatusPending,
			want:   false,
		},
		{
			name:   "suspended cannot login",
			status: identity.StatusSuspended,
			want:   false,
		},
		{
			name:   "deleted cannot login",
			status: identity.StatusDeleted,
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.CanLogin())
		})
	}
}
