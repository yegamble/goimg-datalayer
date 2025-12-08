package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
)

// GetUserSessionsQuery retrieves all active sessions for a user.
// This is a read-only operation with no side effects.
type GetUserSessionsQuery struct {
	UserID      uuid.UUID
	RequestorID uuid.UUID // Who is requesting (for authorization)
}

// GetUserSessionsHandler processes GetUserSessionsQuery requests.
// It retrieves all active sessions for a user from Redis.
type GetUserSessionsHandler struct {
	sessionStore identity.SessionStore
}

// NewGetUserSessionsHandler creates a new GetUserSessionsHandler with the given dependencies.
func NewGetUserSessionsHandler(sessionStore identity.SessionStore) *GetUserSessionsHandler {
	return &GetUserSessionsHandler{
		sessionStore: sessionStore,
	}
}

// Handle executes the GetUserSessionsQuery and returns the list of active sessions.
// Authorization: The requestor must be the user themselves or an admin.
// Returns:
//   - []dto.SessionDTO: List of active sessions with metadata
//   - error: Authorization errors or repository errors
func (h *GetUserSessionsHandler) Handle(ctx context.Context, q GetUserSessionsQuery) ([]dto.SessionDTO, error) {
	// Authorization: Verify requestor owns the user ID
	// In a real implementation, you would check if requestor is admin OR owns the user
	// For now, we require exact match (HTTP layer should enforce admin override)
	if q.RequestorID != q.UserID {
		return nil, fmt.Errorf("unauthorized: cannot view sessions for another user")
	}

	// Retrieve all active sessions from Redis
	sessions, err := h.sessionStore.GetUserSessions(ctx, q.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user sessions: %w", err)
	}

	// Convert to DTOs
	sessionDTOs := make([]dto.SessionDTO, 0, len(sessions))
	for _, session := range sessions {
		sessionDTOs = append(sessionDTOs, dto.SessionDTO{
			SessionID: session.ID.String(),
			IP:        session.IPAddress,
			UserAgent: session.UserAgent,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
			IsCurrent: false, // HTTP layer should set this based on current session
		})
	}

	return sessionDTOs, nil
}
