package activity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestNewActivityID(t *testing.T) {
	t.Parallel()

	// Act
	id := activity.NewActivityID()

	// Assert
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
}

func TestParseActivityID_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	validUUID := uuid.New().String()

	// Act
	id, err := activity.ParseActivityID(validUUID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
	assert.False(t, id.IsZero())
}

func TestParseActivityID_InvalidUUID(t *testing.T) {
	t.Parallel()

	// Arrange
	invalidUUID := "not-a-uuid"

	// Act
	id, err := activity.ParseActivityID(invalidUUID)

	// Assert
	require.Error(t, err)
	assert.True(t, id.IsZero())
}

func TestActivityID_String(t *testing.T) {
	t.Parallel()

	// Arrange
	id := activity.NewActivityID()

	// Act
	str := id.String()

	// Assert
	assert.NotEmpty(t, str)
	_, err := uuid.Parse(str)
	require.NoError(t, err, "String() should return valid UUID format")
}

func TestActivityID_IsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       activity.ActivityID
		expected bool
	}{
		{
			name:     "new ID is not zero",
			id:       activity.NewActivityID(),
			expected: false,
		},
		{
			name:     "default ID is zero",
			id:       activity.ActivityID{},
			expected: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.id.IsZero()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestActivityID_Equals(t *testing.T) {
	t.Parallel()

	// Arrange
	id1 := activity.NewActivityID()
	id2 := activity.NewActivityID()
	id3, _ := activity.ParseActivityID(id1.String())

	// Assert
	assert.True(t, id1.Equals(id1), "ID should equal itself")
	assert.False(t, id1.Equals(id2), "Different IDs should not be equal")
	assert.True(t, id1.Equals(id3), "IDs with same UUID should be equal")
}
