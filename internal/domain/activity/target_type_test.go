package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetType(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		assert.True(t, TargetTypeImage.Valid())
		assert.True(t, TargetTypeUser.Valid())
		assert.True(t, TargetTypeAlbum.Valid())
		assert.True(t, TargetTypeComment.Valid())
		assert.False(t, TargetType("invalid").Valid())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "image", TargetTypeImage.String())
	})

	t.Run("ParseTargetType valid", func(t *testing.T) {
		target, err := ParseTargetType("user")
		require.NoError(t, err)
		assert.Equal(t, TargetTypeUser, target)
	})

	t.Run("ParseTargetType invalid", func(t *testing.T) {
		_, err := ParseTargetType("invalid")
		require.Error(t, err)
	})
}
