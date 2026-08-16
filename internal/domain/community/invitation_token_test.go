package community

import (
	"testing"
)

func TestInvitationToken_Equals(t *testing.T) {
	// Generate two tokens
	t1, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("failed to generate first token: %v", err)
	}

	t2, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("failed to generate second token: %v", err)
	}

	// Create a different instance of the first token with same value
	t1Copy, err := ParseInvitationToken(t1.String())
	if err != nil {
		t.Fatalf("failed to parse first token: %v", err)
	}

	t.Run("should be equal to itself", func(t *testing.T) {
		// Use t1Copy to avoid dupArg linter error
		if !t1.Equals(t1Copy) {
			t.Errorf("token %v should equal itself %v", t1, t1Copy)
		}
	})

	t.Run("should not equal another random token", func(t *testing.T) {
		if t1.Equals(t2) {
			t.Errorf("token %v should not equal %v", t1, t2)
		}
	})

	t.Run("empty tokens should be equal", func(t *testing.T) {
		e1 := InvitationToken{}
		e2 := InvitationToken{}
		if !e1.Equals(e2) {
			t.Error("empty tokens should be equal")
		}
	})

	t.Run("empty token should not equal filled token", func(t *testing.T) {
		e := InvitationToken{}
		if e.Equals(t1) {
			t.Error("empty token should not equal filled token")
		}
		if t1.Equals(e) {
			t.Error("filled token should not equal empty token")
		}
	})
}
