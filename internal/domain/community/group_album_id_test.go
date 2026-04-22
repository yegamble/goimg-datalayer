package community

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupAlbumID(t *testing.T) {
	t.Parallel()

	id := NewGroupAlbumID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseGroupAlbumID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id, err := ParseGroupAlbumID(validUUID)
		require.NoError(t, err)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID", func(t *testing.T) {
		_, err := ParseGroupAlbumID("not-a-uuid")
		require.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := ParseGroupAlbumID("")
		require.Error(t, err)
	})
}

func TestMustParseGroupAlbumID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id := MustParseGroupAlbumID(validUUID)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustParseGroupAlbumID("not-a-uuid")
		})
	})
}

func TestGroupAlbumID_Equals(t *testing.T) {
	t.Parallel()

	id1 := NewGroupAlbumID()
	id2 := NewGroupAlbumID()
	id3, _ := ParseGroupAlbumID(id1.String())

	assert.False(t, id1.Equals(id2))
	assert.True(t, id1.Equals(id3))
	id1Copy := id1
	assert.True(t, id1.Equals(id1Copy)) // Add self-comparison test for coverage
}

func TestGroupAlbumID_IsZero(t *testing.T) {
	t.Parallel()

	zeroID := GroupAlbumID{}
	nonZeroID := NewGroupAlbumID()

	assert.True(t, zeroID.IsZero())
	assert.False(t, nonZeroID.IsZero())
}
