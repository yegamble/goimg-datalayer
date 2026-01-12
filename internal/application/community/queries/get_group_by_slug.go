package queries

import (
	"context"
	"fmt"

	"github.com/yegamble/goimg-datalayer/internal/domain/community"
)

// GetGroupBySlugQuery retrieves a group by its unique slug.
// Slugs are URL-friendly identifiers.
type GetGroupBySlugQuery struct {
	Slug community.GroupSlug
}

// Implement Query interface.
func (GetGroupBySlugQuery) isQuery() {}

// GetGroupBySlugHandler processes GetGroupBySlugQuery requests.
type GetGroupBySlugHandler struct {
	groupRepo community.GroupRepository
}

// NewGetGroupBySlugHandler creates a new GetGroupBySlugHandler with the given dependencies.
func NewGetGroupBySlugHandler(groupRepo community.GroupRepository) *GetGroupBySlugHandler {
	return &GetGroupBySlugHandler{
		groupRepo: groupRepo,
	}
}

// Handle executes the GetGroupBySlugQuery and returns the group data.
//
// Returns:
//   - *community.Group: The group aggregate
//   - error: ErrGroupNotFound if the group does not exist
//
func (h *GetGroupBySlugHandler) Handle(ctx context.Context, q GetGroupBySlugQuery) (*community.Group, error) {
	group, err := h.groupRepo.FindBySlug(ctx, q.Slug)
	if err != nil {
		return nil, fmt.Errorf("find group by slug: %w", err)
	}

	return group, nil
}
