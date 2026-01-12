package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemberStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    MemberStatus
		expected string
	}{
		{"invited", MemberStatusInvited, "invited"},
		{"requested", MemberStatusRequested, "requested"},
		{"active", MemberStatusActive, "active"},
		{"banned", MemberStatusBanned, "banned"},
		{"unknown", MemberStatus(999), "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}

func TestMemberStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    MemberStatus
		expected bool
	}{
		{"invited", MemberStatusInvited, true},
		{"requested", MemberStatusRequested, true},
		{"active", MemberStatusActive, true},
		{"banned", MemberStatusBanned, true},
		{"invalid", MemberStatus(999), false},
		{"negative", MemberStatus(-1), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.IsValid())
		})
	}
}

func TestMemberStatus_Checks(t *testing.T) {
	t.Parallel()

	t.Run("IsActive", func(t *testing.T) {
		assert.False(t, MemberStatusInvited.IsActive())
		assert.False(t, MemberStatusRequested.IsActive())
		assert.True(t, MemberStatusActive.IsActive())
		assert.False(t, MemberStatusBanned.IsActive())
	})

	t.Run("IsPending", func(t *testing.T) {
		assert.True(t, MemberStatusInvited.IsPending())
		assert.True(t, MemberStatusRequested.IsPending())
		assert.False(t, MemberStatusActive.IsPending())
		assert.False(t, MemberStatusBanned.IsPending())
	})

	t.Run("IsBanned", func(t *testing.T) {
		assert.False(t, MemberStatusInvited.IsBanned())
		assert.False(t, MemberStatusRequested.IsBanned())
		assert.False(t, MemberStatusActive.IsBanned())
		assert.True(t, MemberStatusBanned.IsBanned())
	})

	t.Run("CanAcceptInvitation", func(t *testing.T) {
		assert.True(t, MemberStatusInvited.CanAcceptInvitation())
		assert.False(t, MemberStatusRequested.CanAcceptInvitation())
		assert.False(t, MemberStatusActive.CanAcceptInvitation())
		assert.False(t, MemberStatusBanned.CanAcceptInvitation())
	})
}

func TestParseMemberStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    MemberStatus
		wantErr bool
	}{
		{"invited", "invited", MemberStatusInvited, false},
		{"requested", "requested", MemberStatusRequested, false},
		{"active", "active", MemberStatusActive, false},
		{"banned", "banned", MemberStatusBanned, false},
		{"uppercase", "ACTIVE", MemberStatusActive, false},
		{"mixed case", "Invited", MemberStatusInvited, false},
		{"with spaces", "  active  ", MemberStatusActive, false},
		{"invalid", "invalid", MemberStatus(0), true},
		{"empty", "", MemberStatus(0), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMemberStatus(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidMemberStatus)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
