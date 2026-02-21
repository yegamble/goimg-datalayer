package identity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

func TestUser_EmailVerification(t *testing.T) {
	t.Parallel()

	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	passwordHash, _ := identity.NewPasswordHash("SecurePass@123!")

	t.Run("new user has email not verified", func(t *testing.T) {
		t.Parallel()
		user, err := identity.NewUser(email, username, passwordHash)
		require.NoError(t, err)
		assert.False(t, user.EmailVerified())
		assert.Nil(t, user.EmailVerifiedAt())
	})

	t.Run("VerifyEmail sets verified true and timestamp", func(t *testing.T) {
		t.Parallel()
		user, err := identity.NewUser(email, username, passwordHash)
		require.NoError(t, err)

		user.VerifyEmail()

		assert.True(t, user.EmailVerified())
		assert.NotNil(t, user.EmailVerifiedAt())
	})

	t.Run("VerifyEmail is idempotent", func(t *testing.T) {
		t.Parallel()
		user, err := identity.NewUser(email, username, passwordHash)
		require.NoError(t, err)

		user.VerifyEmail()
		firstVerifiedAt := user.EmailVerifiedAt()

		user.VerifyEmail()
		assert.Equal(t, firstVerifiedAt, user.EmailVerifiedAt())
	})
}
