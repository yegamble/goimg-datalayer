package community

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMembershipID(t *testing.T) {
	t.Parallel()

	id := NewMembershipID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseMembershipID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id, err := ParseMembershipID(validUUID)
		require.NoError(t, err)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID", func(t *testing.T) {
		_, err := ParseMembershipID("not-a-uuid")
		require.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := ParseMembershipID("")
		require.Error(t, err)
	})
}

func TestMustParseMembershipID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New().String()

	t.Run("valid UUID", func(t *testing.T) {
		id := MustParseMembershipID(validUUID)
		assert.False(t, id.IsZero())
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("invalid UUID panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustParseMembershipID("not-a-uuid")
		})
	})
}

func TestMembershipID_Equals(t *testing.T) {
	t.Parallel()

	id1 := NewMembershipID()
	id2 := NewMembershipID()
	id3, _ := ParseMembershipID(id1.String())

	assert.False(t, id1.Equals(id2))
	assert.True(t, id1.Equals(id3))
}

func TestMembershipID_IsZero(t *testing.T) {
	t.Parallel()

	zeroID := MembershipID{}
	nonZeroID := NewMembershipID()

	assert.True(t, zeroID.IsZero())
	assert.False(t, nonZeroID.IsZero())
}
