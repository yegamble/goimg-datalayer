package community_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestGroupActivityID(t *testing.T) {
	t.Run("ParseGroupActivityID parses valid ID", func(t *testing.T) {
		idStr := uuid.New().String()
		id, err := community.ParseGroupActivityID(idStr)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if id.String() != idStr {
			t.Errorf("expected %s, got %s", idStr, id.String())
		}
	})

	t.Run("MustParseGroupActivityID parses valid ID", func(t *testing.T) {
		idStr := uuid.New().String()
		id := community.MustParseGroupActivityID(idStr)
		if id.String() != idStr {
			t.Errorf("expected %s, got %s", idStr, id.String())
		}
	})

	t.Run("MustParseGroupActivityID panics on invalid ID", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		community.MustParseGroupActivityID("invalid")
	})

	t.Run("Equals correctly identifies equal IDs", func(t *testing.T) {
		idStr := uuid.New().String()
		id1 := community.MustParseGroupActivityID(idStr)
		id2 := community.MustParseGroupActivityID(idStr)
		if !id1.Equals(id2) {
			t.Errorf("expected IDs to be equal")
		}
	})
}
