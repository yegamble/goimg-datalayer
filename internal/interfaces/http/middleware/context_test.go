package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name:     "returns request ID when present",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), RequestIDKey, "test-request-id-123") },
			expected: "test-request-id-123",
		},
		{
			name:     "returns empty string when not present",
			setupCtx: context.Background,
			expected: "",
		},
		{
			name:     "returns empty string when wrong type",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), RequestIDKey, 12345) },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := GetRequestID(tt.setupCtx())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetRequestID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	requestID := "test-request-id"

	result := SetRequestID(ctx, requestID)

	retrieved := GetRequestID(result)
	assert.Equal(t, requestID, retrieved)
}

func TestGetUserID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		expectedID uuid.UUID
		expectedOK bool
	}{
		{
			name:       "returns user ID when present",
			setupCtx:   func() context.Context { return context.WithValue(context.Background(), UserIDKey, userID) },
			expectedID: userID,
			expectedOK: true,
		},
		{
			name:       "returns nil UUID and false when not present",
			setupCtx:   context.Background,
			expectedID: uuid.Nil,
			expectedOK: false,
		},
		{
			name:       "returns nil UUID and false when wrong type",
			setupCtx:   func() context.Context { return context.WithValue(context.Background(), UserIDKey, "not-a-uuid") },
			expectedID: uuid.Nil,
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := GetUserID(tt.setupCtx())
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestGetUserIDString(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		expectedID string
		expectedOK bool
	}{
		{
			name:       "returns user ID string when present",
			setupCtx:   func() context.Context { return context.WithValue(context.Background(), UserIDKey, userID) },
			expectedID: userID.String(),
			expectedOK: true,
		},
		{
			name:       "returns empty string and false when not present",
			setupCtx:   context.Background,
			expectedID: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := GetUserIDString(tt.setupCtx())
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestGetUserEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
		ok       bool
	}{
		{
			name:     "returns email when present",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), UserEmailKey, "test@example.com") },
			expected: "test@example.com",
			ok:       true,
		},
		{
			name:     "returns empty and false when not present",
			setupCtx: context.Background,
			expected: "",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			email, ok := GetUserEmail(tt.setupCtx())
			assert.Equal(t, tt.expected, email)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestGetUserRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
		ok       bool
	}{
		{
			name:     "returns role when present",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), UserRoleKey, "admin") },
			expected: "admin",
			ok:       true,
		},
		{
			name:     "returns empty and false when not present",
			setupCtx: context.Background,
			expected: "",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			role, ok := GetUserRole(tt.setupCtx())
			assert.Equal(t, tt.expected, role)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestGetSessionID(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		expectedID uuid.UUID
		expectedOK bool
	}{
		{
			name:       "returns session ID when present",
			setupCtx:   func() context.Context { return context.WithValue(context.Background(), SessionIDKey, sessionID) },
			expectedID: sessionID,
			expectedOK: true,
		},
		{
			name:       "returns nil UUID and false when not present",
			setupCtx:   context.Background,
			expectedID: uuid.Nil,
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := GetSessionID(tt.setupCtx())
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestGetSessionIDString(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		expectedID string
		expectedOK bool
	}{
		{
			name:       "returns session ID string when present",
			setupCtx:   func() context.Context { return context.WithValue(context.Background(), SessionIDKey, sessionID) },
			expectedID: sessionID.String(),
			expectedOK: true,
		},
		{
			name:       "returns empty string and false when not present",
			setupCtx:   context.Background,
			expectedID: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := GetSessionIDString(tt.setupCtx())
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestGet2FAVerified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected bool
		ok       bool
	}{
		{
			name:     "returns true when verified",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), TwoFAVerifiedKey, true) },
			expected: true,
			ok:       true,
		},
		{
			name:     "returns false when not verified",
			setupCtx: func() context.Context { return context.WithValue(context.Background(), TwoFAVerifiedKey, false) },
			expected: false,
			ok:       true,
		},
		{
			name:     "returns false and false when not present",
			setupCtx: context.Background,
			expected: false,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			verified, ok := Get2FAVerified(tt.setupCtx())
			assert.Equal(t, tt.expected, verified)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestSetUserContext(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	email := "test@example.com"
	role := "admin"
	twofaVerified := true

	ctx := SetUserContext(context.Background(), userID, email, role, sessionID, twofaVerified)

	// Verify all values are set correctly
	retrievedUserID, ok := GetUserID(ctx)
	require.True(t, ok)
	assert.Equal(t, userID, retrievedUserID)

	retrievedEmail, ok := GetUserEmail(ctx)
	require.True(t, ok)
	assert.Equal(t, email, retrievedEmail)

	retrievedRole, ok := GetUserRole(ctx)
	require.True(t, ok)
	assert.Equal(t, role, retrievedRole)

	retrievedSessionID, ok := GetSessionID(ctx)
	require.True(t, ok)
	assert.Equal(t, sessionID, retrievedSessionID)

	retrieved2FA, ok := Get2FAVerified(ctx)
	require.True(t, ok)
	assert.Equal(t, twofaVerified, retrieved2FA)
}

func TestMustGetUserID(t *testing.T) {
	t.Parallel()

	t.Run("returns user ID when present", func(t *testing.T) {
		t.Parallel()
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), UserIDKey, userID)

		result := MustGetUserID(ctx)
		assert.Equal(t, userID, result)
	})

	t.Run("panics when not present", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetUserID(ctx)
		})
	})
}

func TestMustGetUserIDString(t *testing.T) {
	t.Parallel()

	t.Run("returns user ID string when present", func(t *testing.T) {
		t.Parallel()
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), UserIDKey, userID)

		result := MustGetUserIDString(ctx)
		assert.Equal(t, userID.String(), result)
	})

	t.Run("panics when not present", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetUserIDString(ctx)
		})
	})
}

func TestMustGetUserEmail(t *testing.T) {
	t.Parallel()

	t.Run("returns email when present", func(t *testing.T) {
		t.Parallel()
		email := "test@example.com"
		ctx := context.WithValue(context.Background(), UserEmailKey, email)

		result := MustGetUserEmail(ctx)
		assert.Equal(t, email, result)
	})

	t.Run("panics when not present", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetUserEmail(ctx)
		})
	})
}

func TestMustGetUserRole(t *testing.T) {
	t.Parallel()

	t.Run("returns role when present", func(t *testing.T) {
		t.Parallel()
		role := "admin"
		ctx := context.WithValue(context.Background(), UserRoleKey, role)

		result := MustGetUserRole(ctx)
		assert.Equal(t, role, result)
	})

	t.Run("panics when not present", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetUserRole(ctx)
		})
	})
}

func TestMustGetSessionID(t *testing.T) {
	t.Parallel()

	t.Run("returns session ID when present", func(t *testing.T) {
		t.Parallel()
		sessionID := uuid.New()
		ctx := context.WithValue(context.Background(), SessionIDKey, sessionID)

		result := MustGetSessionID(ctx)
		assert.Equal(t, sessionID, result)
	})

	t.Run("panics when not present", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetSessionID(ctx)
		})
	})
}
