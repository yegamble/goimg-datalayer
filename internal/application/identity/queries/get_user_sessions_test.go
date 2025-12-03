package queries_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
)

// MockSessionStore is a mock implementation of SessionStore for testing.
type MockSessionStore struct {
	getUserSessionsFunc func(ctx context.Context, userID uuid.UUID) ([]*identity.Session, error)
}

func (m *MockSessionStore) Create(ctx context.Context, session *identity.Session) error {
	panic("not implemented")
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID uuid.UUID) (*identity.Session, error) {
	panic("not implemented")
}

func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*identity.Session, error) {
	if m.getUserSessionsFunc != nil {
		return m.getUserSessionsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockSessionStore) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSessionStore) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	panic("not implemented")
}

func (m *MockSessionStore) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	panic("not implemented")
}

func TestGetUserSessionsHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	mockStore := &MockSessionStore{}
	handler := queries.NewGetUserSessionsHandler(mockStore)

	userID := uuid.New()
	now := time.Now().UTC()

	sessions := []*identity.Session{
		{
			ID:        uuid.New(),
			UserID:    userID,
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			CreatedAt: now.Add(-1 * time.Hour),
			ExpiresAt: now.Add(23 * time.Hour),
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			IPAddress: "10.0.0.1",
			UserAgent: "Chrome/120.0",
			CreatedAt: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(22 * time.Hour),
		},
	}

	mockStore.getUserSessionsFunc = func(ctx context.Context, uid uuid.UUID) ([]*identity.Session, error) {
		assert.Equal(t, userID, uid)
		return sessions, nil
	}

	query := queries.GetUserSessionsQuery{
		UserID:      userID,
		RequestorID: userID, // Same user requesting their own sessions
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, sessions[0].ID.String(), result[0].SessionID)
	assert.Equal(t, "192.168.1.1", result[0].IP)
	assert.Equal(t, "Mozilla/5.0", result[0].UserAgent)
}

func TestGetUserSessionsHandler_Handle_Unauthorized(t *testing.T) {
	t.Parallel()

	// Arrange
	mockStore := &MockSessionStore{}
	handler := queries.NewGetUserSessionsHandler(mockStore)

	userID := uuid.New()
	otherUserID := uuid.New()

	query := queries.GetUserSessionsQuery{
		UserID:      userID,
		RequestorID: otherUserID, // Different user trying to access
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGetUserSessionsHandler_Handle_NoSessions(t *testing.T) {
	t.Parallel()

	// Arrange
	mockStore := &MockSessionStore{}
	handler := queries.NewGetUserSessionsHandler(mockStore)

	userID := uuid.New()

	mockStore.getUserSessionsFunc = func(ctx context.Context, uid uuid.UUID) ([]*identity.Session, error) {
		return []*identity.Session{}, nil
	}

	query := queries.GetUserSessionsQuery{
		UserID:      userID,
		RequestorID: userID,
	}

	// Act
	result, err := handler.Handle(context.Background(), query)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result)
}
