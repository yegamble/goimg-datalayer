package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUserStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    UserStatus
		wantErr bool
	}{
		{
			name:    "valid active status",
			input:   "active",
			want:    StatusActive,
			wantErr: false,
		},
		{
			name:    "valid pending status",
			input:   "pending",
			want:    StatusPending,
			wantErr: false,
		},
		{
			name:    "valid suspended status",
			input:   "suspended",
			want:    StatusSuspended,
			wantErr: false,
		},
		{
			name:    "valid deleted status",
			input:   "deleted",
			want:    StatusDeleted,
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

			status, err := ParseUserStatus(tt.input)

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
		status UserStatus
		want   string
	}{
		{
			name:   "active status",
			status: StatusActive,
			want:   "active",
		},
		{
			name:   "pending status",
			status: StatusPending,
			want:   "pending",
		},
		{
			name:   "suspended status",
			status: StatusSuspended,
			want:   "suspended",
		},
		{
			name:   "deleted status",
			status: StatusDeleted,
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
		status UserStatus
		want   bool
	}{
		{
			name:   "active is valid",
			status: StatusActive,
			want:   true,
		},
		{
			name:   "pending is valid",
			status: StatusPending,
			want:   true,
		},
		{
			name:   "suspended is valid",
			status: StatusSuspended,
			want:   true,
		},
		{
			name:   "deleted is valid",
			status: StatusDeleted,
			want:   true,
		},
		{
			name:   "empty is invalid",
			status: UserStatus(""),
			want:   false,
		},
		{
			name:   "random string is invalid",
			status: UserStatus("inactive"),
			want:   false,
		},
		{
			name:   "uppercase is invalid",
			status: UserStatus("ACTIVE"),
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
		status UserStatus
		want   bool
	}{
		{
			name:   "active can login",
			status: StatusActive,
			want:   true,
		},
		{
			name:   "pending cannot login",
			status: StatusPending,
			want:   false,
		},
		{
			name:   "suspended cannot login",
			status: StatusSuspended,
			want:   false,
		},
		{
			name:   "deleted cannot login",
			status: StatusDeleted,
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
