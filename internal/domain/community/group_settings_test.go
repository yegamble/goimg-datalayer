package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		requireApproval    bool
		allowMemberInvites bool
		allowMemberAlbums  bool
		maxMembers         int
		wantErr            error
	}{
		{
			name:               "valid settings - unlimited members",
			requireApproval:    false,
			allowMemberInvites: true,
			allowMemberAlbums:  true,
			maxMembers:         0,
			wantErr:            nil,
		},
		{
			name:               "valid settings - limited members",
			requireApproval:    true,
			allowMemberInvites: false,
			allowMemberAlbums:  false,
			maxMembers:         100,
			wantErr:            nil,
		},
		{
			name:               "invalid - negative max members",
			requireApproval:    false,
			allowMemberInvites: true,
			allowMemberAlbums:  true,
			maxMembers:         -1,
			wantErr:            ErrInvalidMaxMembers,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			settings, err := NewGroupSettings(tt.requireApproval, tt.allowMemberInvites, tt.allowMemberAlbums, tt.maxMembers)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.requireApproval, settings.RequireApproval())
				assert.Equal(t, tt.allowMemberInvites, settings.AllowMemberInvites())
				assert.Equal(t, tt.allowMemberAlbums, settings.AllowMemberAlbums())
				assert.Equal(t, tt.maxMembers, settings.MaxMembers())
			}
		})
	}
}

func TestDefaultGroupSettings(t *testing.T) {
	t.Parallel()

	settings := DefaultGroupSettings()

	assert.False(t, settings.RequireApproval())
	assert.True(t, settings.AllowMemberInvites())
	assert.True(t, settings.AllowMemberAlbums())
	assert.Equal(t, 0, settings.MaxMembers())
	assert.False(t, settings.HasMemberLimit())
}

func TestGroupSettings_HasMemberLimit(t *testing.T) {
	t.Parallel()

	unlimited, _ := NewGroupSettings(false, true, true, 0)
	limited, _ := NewGroupSettings(false, true, true, 100)

	assert.False(t, unlimited.HasMemberLimit())
	assert.True(t, limited.HasMemberLimit())
}

func TestGroupSettings_Equals(t *testing.T) {
	t.Parallel()

	settings1, _ := NewGroupSettings(true, false, true, 100)
	settings2, _ := NewGroupSettings(true, false, true, 100)
	settings3, _ := NewGroupSettings(false, true, false, 0)

	assert.True(t, settings1.Equals(settings2))
	assert.False(t, settings1.Equals(settings3))
}

func TestGroupSettings_WithMethods(t *testing.T) {
	t.Parallel()

	original, _ := NewGroupSettings(false, true, true, 0)

	t.Run("WithRequireApproval", func(t *testing.T) {
		modified := original.WithRequireApproval(true)
		assert.True(t, modified.RequireApproval())
		assert.True(t, modified.AllowMemberInvites())
		assert.True(t, modified.AllowMemberAlbums())
		assert.Equal(t, 0, modified.MaxMembers())
		// Original unchanged
		assert.False(t, original.RequireApproval())
	})

	t.Run("WithAllowMemberInvites", func(t *testing.T) {
		modified := original.WithAllowMemberInvites(false)
		assert.False(t, modified.RequireApproval())
		assert.False(t, modified.AllowMemberInvites())
		assert.True(t, modified.AllowMemberAlbums())
		assert.Equal(t, 0, modified.MaxMembers())
		// Original unchanged
		assert.True(t, original.AllowMemberInvites())
	})

	t.Run("WithAllowMemberAlbums", func(t *testing.T) {
		modified := original.WithAllowMemberAlbums(false)
		assert.False(t, modified.RequireApproval())
		assert.True(t, modified.AllowMemberInvites())
		assert.False(t, modified.AllowMemberAlbums())
		assert.Equal(t, 0, modified.MaxMembers())
		// Original unchanged
		assert.True(t, original.AllowMemberAlbums())
	})

	t.Run("WithMaxMembers - valid", func(t *testing.T) {
		modified, err := original.WithMaxMembers(50)
		require.NoError(t, err)
		assert.False(t, modified.RequireApproval())
		assert.True(t, modified.AllowMemberInvites())
		assert.True(t, modified.AllowMemberAlbums())
		assert.Equal(t, 50, modified.MaxMembers())
		// Original unchanged
		assert.Equal(t, 0, original.MaxMembers())
	})

	t.Run("WithMaxMembers - invalid", func(t *testing.T) {
		_, err := original.WithMaxMembers(-1)
		require.ErrorIs(t, err, ErrInvalidMaxMembers)
	})
}
