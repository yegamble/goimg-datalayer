package shared_test

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"testing"
)

func TestNotificationType_Methods(t *testing.T) {
	nt := shared.NotificationTypeNewFollower
	if !nt.IsValid() {
		t.Fatalf("expected IsValid to return true")
	}
	if nt.IsAdminOnly() {
		t.Fatalf("expected IsAdminOnly to return false")
	}
	if nt.RequiresEmail() {
		t.Fatalf("expected RequiresEmail to return false")
	}
	if nt.String() != string(nt) {
		t.Fatalf("expected String to return correct value")
	}
}
