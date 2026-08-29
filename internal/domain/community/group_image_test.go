package community_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestGroupImage(t *testing.T) {
	groupID := community.NewGroupID()
	imageID := gallery.NewImageID()
	userID := identity.NewUserID()

	t.Run("NewGroupImage creates a valid pending group image", func(t *testing.T) {
		gi, err := community.NewGroupImage(groupID, imageID, userID, true)
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		if !gi.IsPending() {
			t.Errorf("expected new group image to be pending")
		}
	})

	t.Run("Approve correctly approves the image", func(t *testing.T) {
		gi, _ := community.NewGroupImage(groupID, imageID, userID, true)
		approverID := identity.NewUserID()

		err := gi.Approve(approverID)
		if err != nil {
			t.Fatalf("expected to approve, got error: %v", err)
		}
		if !gi.IsApproved() {
			t.Errorf("expected group image to be approved")
		}
	})

	t.Run("Reject correctly rejects the image", func(t *testing.T) {
		gi, _ := community.NewGroupImage(groupID, imageID, userID, true)
		rejecterID := identity.NewUserID()

		err := gi.Reject(rejecterID)
		if err != nil {
			t.Fatalf("expected to reject, got error: %v", err)
		}
		if !gi.IsRejected() {
			t.Errorf("expected group image to be rejected")
		}
	})
}
