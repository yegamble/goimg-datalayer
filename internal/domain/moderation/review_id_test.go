package moderation_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestNewReviewID(t *testing.T) {
	t.Parallel()

	id := moderation.NewReviewID()

	assert.False(t, id.IsZero(), "new ReviewID should not be zero")
	assert.NotEmpty(t, id.String(), "new ReviewID should have string representation")

	// Should be valid UUID
	_, err := uuid.Parse(id.String())
	assert.NoError(t, err, "ReviewID string should be valid UUID")
}

func TestParseReviewID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid uuid",
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
			name:    "partial uuid",
			input:   "550e8400-e29b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := moderation.ParseReviewID(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
				assert.False(t, id.IsZero())
			}
		})
	}
}

func TestMustParseReviewID(t *testing.T) {
	t.Parallel()

	t.Run("valid uuid", func(t *testing.T) {
		t.Parallel()

		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		id := moderation.MustParseReviewID(validUUID)

		assert.Equal(t, validUUID, id.String())
		assert.False(t, id.IsZero())
	})

	t.Run("invalid uuid panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			moderation.MustParseReviewID("not-a-uuid")
		})
	})
}

func TestReviewID_String(t *testing.T) {
	t.Parallel()

	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	id := moderation.MustParseReviewID(validUUID)

	assert.Equal(t, validUUID, id.String())
}

func TestReviewID_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("new id is not zero", func(t *testing.T) {
		t.Parallel()

		id := moderation.NewReviewID()
		assert.False(t, id.IsZero())
	})

	t.Run("default id is zero", func(t *testing.T) {
		t.Parallel()

		var id moderation.ReviewID
		assert.True(t, id.IsZero())
	})
}

func TestReviewID_Equals(t *testing.T) {
	t.Parallel()

	t.Run("same id equals", func(t *testing.T) {
		t.Parallel()

		id1 := moderation.MustParseReviewID("550e8400-e29b-41d4-a716-446655440000")
		id2 := moderation.MustParseReviewID("550e8400-e29b-41d4-a716-446655440000")

		assert.True(t, id1.Equals(id2))
		assert.True(t, id2.Equals(id1))
	})

	t.Run("different ids not equal", func(t *testing.T) {
		t.Parallel()

		id1 := moderation.NewReviewID()
		id2 := moderation.NewReviewID()

		assert.False(t, id1.Equals(id2))
		assert.False(t, id2.Equals(id1))
	})

	t.Run("zero ids equal", func(t *testing.T) {
		t.Parallel()

		var id1, id2 moderation.ReviewID

		assert.True(t, id1.Equals(id2))
	})
}
