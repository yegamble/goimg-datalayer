package community_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

func TestInvitationToken_NewInvitationToken(t *testing.T) {
	token, err := community.NewInvitationToken()
	require.NoError(t, err)

	assert.False(t, token.IsEmpty())
	// Length should be 64 characters (32 bytes hex encoded)
	assert.Equal(t, 64, len(token.String()))
}

func TestInvitationToken_Equals(t *testing.T) {
	// Generate a token
	t1, err := community.NewInvitationToken()
	require.NoError(t, err)

	// Create a copy of the token using assignment as per memory rules
	t2 := t1

	// Tokens should be equal
	assert.True(t, t1.Equals(t2))

	// Generate a different token
	t3, err := community.NewInvitationToken()
	require.NoError(t, err)

	// Tokens should not be equal
	assert.False(t, t1.Equals(t3))
}
