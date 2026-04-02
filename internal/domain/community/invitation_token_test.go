package community_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestInvitationToken_Equals(t *testing.T) {
	t.Parallel()

	t.Run("matching tokens", func(t *testing.T) {
		t.Parallel()
		token1, err := community.NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error creating token: %v", err)
		}

		token2 := token1 // Avoid gocritic dupArg rule

		if !token1.Equals(token2) {
			t.Errorf("expected matching tokens to be equal")
		}
	})

	t.Run("non-matching tokens", func(t *testing.T) {
		t.Parallel()
		token1, err := community.NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error creating token: %v", err)
		}

		token2, err := community.NewInvitationToken()
		if err != nil {
			t.Fatalf("unexpected error creating token: %v", err)
		}

		if token1.Equals(token2) {
			t.Errorf("expected non-matching tokens to not be equal")
		}
	})

	t.Run("empty tokens", func(t *testing.T) {
		t.Parallel()
		token1 := community.InvitationToken{}
		token2 := community.InvitationToken{}

		if !token1.Equals(token2) {
			t.Errorf("expected empty tokens to be equal")
		}
	})
}
