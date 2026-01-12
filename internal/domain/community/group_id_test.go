package community

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupID(t *testing.T) {
	t.Parallel()

	id := NewGroupID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseGroupID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id, err := ParseGroupID(validUUID)
		require.NoError(t, err)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID", func(t *testing.T) {
		_, err := ParseGroupID("not-a-uuid")
		require.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := ParseGroupID("")
		require.Error(t, err)
	})
}

func TestMustParseGroupID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id := MustParseGroupID(validUUID)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustParseGroupID("not-a-uuid")
		})
	})
}

func TestGroupID_Equals(t *testing.T) {
	t.Parallel()

	id1 := NewGroupID()
	id2 := NewGroupID()
	id3, _ := ParseGroupID(id1.String())

	assert.False(t, id1.Equals(id2))
	assert.True(t, id1.Equals(id3))
}

func TestGroupID_IsZero(t *testing.T) {
	t.Parallel()

	zeroID := GroupID{}
	nonZeroID := NewGroupID()

	assert.True(t, zeroID.IsZero())
	assert.False(t, nonZeroID.IsZero())
}
