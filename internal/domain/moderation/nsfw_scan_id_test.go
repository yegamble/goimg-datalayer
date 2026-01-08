package moderation_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewNSFWScanID(t *testing.T) {
	id := moderation.NewNSFWScanID()

	assert.False(t, id.IsZero(), "new ID should not be zero")
	assert.NotEmpty(t, id.String(), "new ID should have a string representation")
}

func TestNSFWScanIDFromUUID(t *testing.T) {
	originalUUID := uuid.New()
	id := moderation.NSFWScanIDFromUUID(originalUUID)

	assert.Equal(t, originalUUID, id.UUID())
	assert.Equal(t, originalUUID.String(), id.String())
}

func TestParseNSFWScanID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := moderation.ParseNSFWScanID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
			}
		})
	}
}

func TestMustParseNSFWScanID(t *testing.T) {
	t.Run("valid UUID does not panic", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		assert.NotPanics(t, func() {
			id := moderation.MustParseNSFWScanID(validUUID)
			assert.Equal(t, validUUID, id.String())
		})
	})

	t.Run("invalid UUID panics", func(t *testing.T) {
		assert.Panics(t, func() {
			moderation.MustParseNSFWScanID("not-a-uuid")
		})
	})
}

func TestNSFWScanID_IsZero(t *testing.T) {
	t.Run("zero value is zero", func(t *testing.T) {
		var id moderation.NSFWScanID
		assert.True(t, id.IsZero())
	})

	t.Run("nil UUID is zero", func(t *testing.T) {
		id := moderation.NSFWScanIDFromUUID(uuid.Nil)
		assert.True(t, id.IsZero())
	})

	t.Run("new ID is not zero", func(t *testing.T) {
		id := moderation.NewNSFWScanID()
		assert.False(t, id.IsZero())
	})
}

func TestNSFWScanID_Equals(t *testing.T) {
	id1 := moderation.NewNSFWScanID()
	id2 := moderation.NewNSFWScanID()

	t.Run("same ID equals itself", func(t *testing.T) {
		//nolint:gocritic // Testing reflexivity - id should equal itself
		assert.True(t, id1.Equals(id1))
	})

	t.Run("different IDs are not equal", func(t *testing.T) {
		assert.False(t, id1.Equals(id2))
	})

	t.Run("parsed ID equals original", func(t *testing.T) {
		parsed, err := moderation.ParseNSFWScanID(id1.String())
		require.NoError(t, err)
		assert.True(t, id1.Equals(parsed))
	})
}
