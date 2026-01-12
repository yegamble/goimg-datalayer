package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    GroupType
		expected string
	}{
		{"public", GroupTypePublic, "public"},
		{"private", GroupTypePrivate, "private"},
		{"invite_only", GroupTypeInviteOnly, "invite_only"},
		{"unknown", GroupType(999), "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}

func TestGroupType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    GroupType
		expected bool
	}{
		{"public", GroupTypePublic, true},
		{"private", GroupTypePrivate, true},
		{"invite_only", GroupTypeInviteOnly, true},
		{"invalid", GroupType(999), false},
		{"negative", GroupType(-1), false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.IsValid())
		})
	}
}

func TestGroupType_IsDiscoverable(t *testing.T) {
	t.Parallel()

	assert.True(t, GroupTypePublic.IsDiscoverable())
	assert.True(t, GroupTypeInviteOnly.IsDiscoverable())
	assert.False(t, GroupTypePrivate.IsDiscoverable())
}

func TestGroupType_AllowsInstantJoin(t *testing.T) {
	t.Parallel()

	assert.True(t, GroupTypePublic.AllowsInstantJoin())
	assert.False(t, GroupTypePrivate.AllowsInstantJoin())
	assert.False(t, GroupTypeInviteOnly.AllowsInstantJoin())
}

func TestParseGroupType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    GroupType
		wantErr bool
	}{
		{"public", "public", GroupTypePublic, false},
		{"private", "private", GroupTypePrivate, false},
		{"invite_only", "invite_only", GroupTypeInviteOnly, false},
		{"inviteonly", "inviteonly", GroupTypeInviteOnly, false},
		{"invite-only", "invite-only", GroupTypeInviteOnly, false},
		{"uppercase", "PUBLIC", GroupTypePublic, false},
		{"mixed case", "Private", GroupTypePrivate, false},
		{"with spaces", "  public  ", GroupTypePublic, false},
		{"invalid", "invalid", GroupType(0), true},
		{"empty", "", GroupType(0), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseGroupType(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidGroupType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
