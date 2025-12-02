package identity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestNewUserID(t *testing.T) {
	t.Parallel()

	id := identity.NewUserID()

	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
	assert.Len(t, id.String(), 36) // UUID string length
}

func TestParseUserID(t *testing.T) {
	t.Parallel()

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
			name:    "invalid format",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "nil UUID is valid",
			input:   "00000000-0000-0000-0000-000000000000",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := identity.ParseUserID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
			}
		})
	}
}

func TestUserID_String(t *testing.T) {
	t.Parallel()

	expectedUUID := "550e8400-e29b-41d4-a716-446655440000"
	id, err := identity.ParseUserID(expectedUUID)
	require.NoError(t, err)

	assert.Equal(t, expectedUUID, id.String())
}

func TestUserID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("zero value is zero", func(t *testing.T) {
		t.Parallel()

		var id identity.UserID
		assert.True(t, id.IsZero())
	})

	t.Run("nil UUID is zero", func(t *testing.T) {
		t.Parallel()

		id, _ := identity.ParseUserID("00000000-0000-0000-0000-000000000000")
		assert.True(t, id.IsZero())
	})

	t.Run("generated ID is not zero", func(t *testing.T) {
		t.Parallel()

		id := identity.NewUserID()
		assert.False(t, id.IsZero())
	})
}

func TestUserID_Equals(t *testing.T) {
	t.Parallel()

	t.Run("same IDs are equal", func(t *testing.T) {
		t.Parallel()

		id1, err := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440000")
		require.NoError(t, err)
		id2, err := identity.ParseUserID("550e8400-e29b-41d4-a716-446655440000")
		require.NoError(t, err)

		assert.True(t, id1.Equals(id2))
		assert.True(t, id2.Equals(id1))
	})

	t.Run("different IDs are not equal", func(t *testing.T) {
		t.Parallel()

		id1 := identity.NewUserID()
		id2 := identity.NewUserID()

		assert.False(t, id1.Equals(id2))
	})

	t.Run("zero values are equal", func(t *testing.T) {
		t.Parallel()

		var id1, id2 identity.UserID
		assert.True(t, id1.Equals(id2))
	})
}
