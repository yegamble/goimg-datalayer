package community_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestMustParseGroupActivityID(t *testing.T) {
	idStr := uuid.New().String()

	// Valid UUID should not panic
	assert.NotPanics(t, func() {
		id := community.MustParseGroupActivityID(idStr)
		assert.Equal(t, idStr, id.String())
	})

	// Invalid UUID should panic
	assert.Panics(t, func() {
		community.MustParseGroupActivityID("invalid-uuid")
	})
}
