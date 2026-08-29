package community

import (
	"testing"
)

func TestInvitationToken_Equals(t *testing.T) {
	t1, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t2 := t1

	if !t1.Equals(t2) {
		t.Error("Expected t1 to equal t2")
	}

	t3, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	if t1.Equals(t3) {
		t.Error("Expected t1 to not equal t3")
	}
}

func TestParseInvitationToken(t *testing.T) {
	t1, _ := NewInvitationToken()

	// Valid token
	t2, err := ParseInvitationToken(t1.String())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !t1.Equals(t2) {
		t.Error("Tokens should be equal")
	}

	// Invalid token (wrong length)
	_, err = ParseInvitationToken("invalid")
	if err == nil {
		t.Error("Expected error for invalid length")
	}

	// Invalid token (not hex)
	notHex := string(make([]byte, InvitationTokenLength*2))
	_, err = ParseInvitationToken(notHex)
	if err == nil {
		t.Error("Expected error for non-hex string")
	}
}

func TestInvitationToken_IsEmpty(t *testing.T) {
	t1 := InvitationToken{}
	if !t1.IsEmpty() {
		t.Error("Expected token to be empty")
	}

	t2, _ := NewInvitationToken()
	if t2.IsEmpty() {
		t.Error("Expected token to not be empty")
	}
}

func TestInvitationToken_String(t *testing.T) {
	t1, _ := NewInvitationToken()

	if t1.String() == "" {
		t.Error("String() should not return an empty string for a valid token")
	}
}
