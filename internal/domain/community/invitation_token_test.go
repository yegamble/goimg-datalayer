package community

import (
	"testing"
)

func TestInvitationToken_Equals(t *testing.T) {
	t.Parallel()

	t.Run("returns true for identical tokens", func(t *testing.T) {
		t.Parallel()
		token1, err := NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Create a separate token with same value to avoid dupArg linter error
		token2, err := ParseInvitationToken(token1.String())
		if err != nil {
			t.Fatalf("unexpected error parsing token: %v", err)
		}

		if !token1.Equals(token2) {
			t.Errorf("expected tokens to be equal")
		}
	})

	t.Run("returns false for different tokens", func(t *testing.T) {
		t.Parallel()
		token1, err := NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		token2, err := NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if token1.Equals(token2) {
			t.Errorf("expected tokens to not be equal")
		}
	})
}
