package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationToken_NewInvitationToken(t *testing.T) {
	token, err := NewInvitationToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token.String())
	assert.Equal(t, 64, len(token.String()))
	assert.False(t, token.IsEmpty())
}

func TestInvitationToken_ParseInvitationToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		tokenStr := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		token, err := ParseInvitationToken(tokenStr)
		require.NoError(t, err)
		assert.Equal(t, tokenStr, token.String())
	})

	t.Run("invalid length", func(t *testing.T) {
		_, err := ParseInvitationToken("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be 64 characters")
	})

	t.Run("invalid hex", func(t *testing.T) {
		invalidHex := "g3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		_, err := ParseInvitationToken(invalidHex)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be hexadecimal")
	})
}

func TestInvitationToken_Equals(t *testing.T) {
	token1, err := NewInvitationToken()
	require.NoError(t, err)

	token2, err := ParseInvitationToken(token1.String())
	require.NoError(t, err)

	token3, err := NewInvitationToken()
	require.NoError(t, err)

	assert.True(t, token1.Equals(token2))
	assert.False(t, token1.Equals(token3))
}

func TestInvitationToken_RandReadError(t *testing.T) {
	// Not practically testable as crypto/rand.Read reading from /dev/urandom almost never fails
	// Could test by replacing the rand.Read variable if it was exported, but we'll accept
	// the missing coverage for that specific error path.
}
