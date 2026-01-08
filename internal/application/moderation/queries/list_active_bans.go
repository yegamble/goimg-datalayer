package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// ListActiveBansQuery retrieves all currently active bans.
// This is used for the admin view to see all banned users.
type ListActiveBansQuery struct {
	// No pagination yet - we can add it later if needed
}

// Implement Query interface
func (ListActiveBansQuery) isQuery() {}

// BanDTO represents a ban data transfer object for HTTP responses.
type BanDTO struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	BannedBy    string     `json:"banned_by"`
	Reason      string     `json:"reason"`
	IsPermanent bool       `json:"is_permanent"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   *string    `json:"revoked_by,omitempty"`
}

// ListActiveBansResult represents all currently active bans.
type ListActiveBansResult struct {
	Bans       []BanDTO `json:"bans"`
	TotalCount int      `json:"total_count"`
}

// ListActiveBansHandler processes ListActiveBansQuery requests.
// It retrieves all active bans for admin viewing.
type ListActiveBansHandler struct {
	bans   moderation.BanRepository
	logger *zerolog.Logger
}

// NewListActiveBansHandler creates a new ListActiveBansHandler with the given dependencies.
func NewListActiveBansHandler(
	bans moderation.BanRepository,
	logger *zerolog.Logger,
) *ListActiveBansHandler {
	return &ListActiveBansHandler{
		bans:   bans,
		logger: logger,
	}
}

// Handle executes the ListActiveBansQuery and returns all active bans.
//
// Process flow:
//  1. Load all active bans from repository
//  2. Convert to DTOs
//  3. Return with count
//
// Returns:
//   - *ListActiveBansResult: All active bans
//   - Repository errors
func (h *ListActiveBansHandler) Handle(ctx context.Context, q ListActiveBansQuery) (*ListActiveBansResult, error) {
	// 1. Load all active bans from repository
	bans, err := h.bans.FindActiveBans(ctx)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("failed to list active bans")
		return nil, fmt.Errorf("find active bans: %w", err)
	}

	// 2. Convert to DTOs
	banDTOs := make([]BanDTO, 0, len(bans))
	for _, ban := range bans {
		banDTOs = append(banDTOs, *banToDTO(ban))
	}

	// 3. Build result
	result := &ListActiveBansResult{
		Bans:       banDTOs,
		TotalCount: len(banDTOs),
	}

	h.logger.Debug().
		Int("total_count", len(banDTOs)).
		Msg("active bans listed successfully")

	return result, nil
}

// banToDTO converts a domain Ban to a BanDTO.
func banToDTO(b *moderation.Ban) *BanDTO {
	dto := &BanDTO{
		ID:          b.ID().String(),
		UserID:      b.UserID().String(),
		BannedBy:    b.BannedBy().String(),
		Reason:      b.Reason(),
		IsPermanent: b.IsPermanent(),
		IsActive:    b.IsActive(),
		ExpiresAt:   b.ExpiresAt(),
		CreatedAt:   b.CreatedAt(),
		RevokedAt:   b.RevokedAt(),
	}

	if b.RevokedBy() != nil {
		revokedByStr := b.RevokedBy().String()
		dto.RevokedBy = &revokedByStr
	}

	return dto
}
