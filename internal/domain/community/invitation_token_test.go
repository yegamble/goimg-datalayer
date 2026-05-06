package community

import (
	"strings"
	"testing"
)

func TestNewInvitationToken(t *testing.T) {
	token, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token.IsEmpty() {
		t.Fatal("expected token to not be empty")
	}

	if len(token.String()) != InvitationTokenLength*2 {
		t.Fatalf("expected token length %d, got %d", InvitationTokenLength*2, len(token.String()))
	}
}

func TestInvitationToken_Equals(t *testing.T) {
	token1, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	token2, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token1.Equals(token2) {
		t.Fatal("expected tokens to be different")
	}

	// Create a copy of token1 to avoid gocritic linter "dupArg: suspicious method call with the same argument and receiver"
	token1Copy := token1
	if !token1.Equals(token1Copy) {
		t.Fatal("expected identical tokens to be equal")
	}
}

func TestInvitationToken_ParseInvitationToken(t *testing.T) {
	// Valid token
	token, err := NewInvitationToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	parsedToken, err := ParseInvitationToken(token.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !token.Equals(parsedToken) {
		t.Fatal("expected parsed token to equal original token")
	}

	// Invalid length
	_, err = ParseInvitationToken("short")
	if err == nil {
		t.Fatal("expected error for short token")
	}

	// Invalid hex characters
	invalidHex := strings.Repeat("z", InvitationTokenLength*2)
	_, err = ParseInvitationToken(invalidHex)
	if err == nil {
		t.Fatal("expected error for invalid hex token")
	}
}
