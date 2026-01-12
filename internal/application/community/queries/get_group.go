// Package queries implements read operations for the community bounded context.
package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

// GetGroupQuery retrieves a group by its unique ID.
// This is a read-only operation with no side effects.
type GetGroupQuery struct {
	GroupID community.GroupID
}

// Implement Query interface.
func (GetGroupQuery) isQuery() {}

// GetGroupHandler processes GetGroupQuery requests.
type GetGroupHandler struct {
	groupRepo community.GroupRepository
}

// NewGetGroupHandler creates a new GetGroupHandler with the given dependencies.
func NewGetGroupHandler(groupRepo community.GroupRepository) *GetGroupHandler {
	return &GetGroupHandler{
		groupRepo: groupRepo,
	}
}

// Handle executes the GetGroupQuery and returns the group data.
//
// Returns:
//   - *community.Group: The group aggregate
//   - error: ErrGroupNotFound if the group does not exist
//
func (h *GetGroupHandler) Handle(ctx context.Context, q GetGroupQuery) (*community.Group, error) {
	group, err := h.groupRepo.FindByID(ctx, q.GroupID)
	if err != nil {
		return nil, fmt.Errorf("find group by id: %w", err)
	}

	return group, nil
}
