package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    GroupRole
		expected string
	}{
		{"member", GroupRoleMember, "member"},
		{"admin", GroupRoleAdmin, "admin"},
		{"owner", GroupRoleOwner, "owner"},
		{"unknown", GroupRole(999), "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}

func TestGroupRole_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    GroupRole
		expected bool
	}{
		{"member", GroupRoleMember, true},
		{"admin", GroupRoleAdmin, true},
		{"owner", GroupRoleOwner, true},
		{"invalid", GroupRole(999), false},
		{"negative", GroupRole(-1), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.IsValid())
		})
	}
}

func TestGroupRole_Permissions(t *testing.T) {
	t.Parallel()

	t.Run("CanManageMembers", func(t *testing.T) {
		assert.False(t, GroupRoleMember.CanManageMembers())
		assert.True(t, GroupRoleAdmin.CanManageMembers())
		assert.True(t, GroupRoleOwner.CanManageMembers())
	})

	t.Run("CanModerateContent", func(t *testing.T) {
		assert.False(t, GroupRoleMember.CanModerateContent())
		assert.True(t, GroupRoleAdmin.CanModerateContent())
		assert.True(t, GroupRoleOwner.CanModerateContent())
	})

	t.Run("CanUpdateSettings", func(t *testing.T) {
		assert.False(t, GroupRoleMember.CanUpdateSettings())
		assert.True(t, GroupRoleAdmin.CanUpdateSettings())
		assert.True(t, GroupRoleOwner.CanUpdateSettings())
	})

	t.Run("CanDeleteGroup", func(t *testing.T) {
		assert.False(t, GroupRoleMember.CanDeleteGroup())
		assert.False(t, GroupRoleAdmin.CanDeleteGroup())
		assert.True(t, GroupRoleOwner.CanDeleteGroup())
	})
}

func TestGroupRole_IsHigherThan(t *testing.T) {
	t.Parallel()

	assert.False(t, GroupRoleMember.IsHigherThan(GroupRoleMember))
	assert.False(t, GroupRoleMember.IsHigherThan(GroupRoleAdmin))
	assert.False(t, GroupRoleMember.IsHigherThan(GroupRoleOwner))

	assert.True(t, GroupRoleAdmin.IsHigherThan(GroupRoleMember))
	assert.False(t, GroupRoleAdmin.IsHigherThan(GroupRoleAdmin))
	assert.False(t, GroupRoleAdmin.IsHigherThan(GroupRoleOwner))

	assert.True(t, GroupRoleOwner.IsHigherThan(GroupRoleMember))
	assert.True(t, GroupRoleOwner.IsHigherThan(GroupRoleAdmin))
	assert.False(t, GroupRoleOwner.IsHigherThan(GroupRoleOwner))
}

func TestGroupRole_Checks(t *testing.T) {
	t.Parallel()

	t.Run("IsOwner", func(t *testing.T) {
		assert.False(t, GroupRoleMember.IsOwner())
		assert.False(t, GroupRoleAdmin.IsOwner())
		assert.True(t, GroupRoleOwner.IsOwner())
	})

	t.Run("IsAdmin", func(t *testing.T) {
		assert.False(t, GroupRoleMember.IsAdmin())
		assert.True(t, GroupRoleAdmin.IsAdmin())
		assert.False(t, GroupRoleOwner.IsAdmin())
	})
}

func TestParseGroupRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    GroupRole
		wantErr bool
	}{
		{"member", "member", GroupRoleMember, false},
		{"admin", "admin", GroupRoleAdmin, false},
		{"owner", "owner", GroupRoleOwner, false},
		{"uppercase", "MEMBER", GroupRoleMember, false},
		{"mixed case", "Admin", GroupRoleAdmin, false},
		{"with spaces", "  owner  ", GroupRoleOwner, false},
		{"invalid", "invalid", GroupRole(0), true},
		{"empty", "", GroupRole(0), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseGroupRole(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidGroupRole)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
