package community_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestNewInvitationToken(t *testing.T) {
	t.Parallel()

	token, err := community.NewInvitationToken()
	require.NoError(t, err)
	assert.False(t, token.IsEmpty())
	assert.Len(t, token.String(), community.InvitationTokenLength*2)
}

func TestParseInvitationToken(t *testing.T) {
	t.Parallel()

	t.Run("Valid Token", func(t *testing.T) {
		t.Parallel()
		tokenStr := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		token, err := community.ParseInvitationToken(tokenStr)
		require.NoError(t, err)
		assert.Equal(t, tokenStr, token.String())
	})

	t.Run("Invalid Length", func(t *testing.T) {
		t.Parallel()
		tokenStr := "123"
		_, err := community.ParseInvitationToken(tokenStr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be 64 characters")
	})

	t.Run("Invalid Hex", func(t *testing.T) {
		t.Parallel()
		tokenStr := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
		_, err := community.ParseInvitationToken(tokenStr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be hexadecimal")
	})
}

func TestInvitationToken_Equals(t *testing.T) {
	t.Parallel()

	token1, _ := community.NewInvitationToken()
	token2, _ := community.NewInvitationToken()

	assert.True(t, token1.Equals(token1))
	assert.False(t, token1.Equals(token2))
}
