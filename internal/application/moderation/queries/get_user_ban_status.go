package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

// GetUserBanStatusQuery checks if a user is currently banned.
// This is used for authorization checks and user status displays.
type GetUserBanStatusQuery struct {
	UserID string // User to check ban status for
}

// Implement Query interface
func (GetUserBanStatusQuery) isQuery() {}

// UserBanStatusResult represents a user's current ban status.
type UserBanStatusResult struct {
	UserID      string     `json:"user_id"`
	IsBanned    bool       `json:"is_banned"`
	BanID       *string    `json:"ban_id,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
	BannedBy    *string    `json:"banned_by,omitempty"`
	IsPermanent *bool      `json:"is_permanent,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

// GetUserBanStatusHandler processes GetUserBanStatusQuery requests.
// It checks if a user has an active ban and returns the details.
type GetUserBanStatusHandler struct {
	bans moderation.BanRepository
}

// NewGetUserBanStatusHandler creates a new GetUserBanStatusHandler with the given dependencies.
func NewGetUserBanStatusHandler(bans moderation.BanRepository) *GetUserBanStatusHandler {
	return &GetUserBanStatusHandler{
		bans: bans,
	}
}

// Handle executes the GetUserBanStatusQuery and returns the user's ban status.
//
// Process flow:
//  1. Parse and validate user ID
//  2. Check if user is banned (quick check)
//  3. If banned, retrieve the ban details
//  4. Return status with details
//
// Returns:
//   - *UserBanStatusResult: The user's ban status with details if banned
//   - Validation errors for invalid user ID
func (h *GetUserBanStatusHandler) Handle(ctx context.Context, q GetUserBanStatusQuery) (*UserBanStatusResult, error) {
	// 1. Parse and validate user ID
	userID, err := identity.ParseUserID(q.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	// 2. Check if user is banned (quick check)
	isBanned, err := h.bans.IsUserBanned(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check user ban status: %w", err)
	}

	// Base result
	result := &UserBanStatusResult{
		UserID:   userID.String(),
		IsBanned: isBanned,
	}

	// 3. If banned, retrieve the ban details
	if isBanned {
		ban, err := h.bans.FindByUserID(ctx, userID)
		if err != nil {
			// User is banned but we can't find the ban - log this inconsistency
			// Return isBanned = true but without details
			return result, nil
		}

		// Only return details if the ban is actually active
		if ban.IsActive() {
			banIDStr := ban.ID().String()
			reasonStr := ban.Reason()
			bannedByStr := ban.BannedBy().String()
			isPermanent := ban.IsPermanent()
			createdAt := ban.CreatedAt()

			result.BanID = &banIDStr
			result.Reason = &reasonStr
			result.BannedBy = &bannedByStr
			result.IsPermanent = &isPermanent
			result.ExpiresAt = ban.ExpiresAt()
			result.CreatedAt = &createdAt
		} else {
			// Ban exists but is not active (expired or revoked)
			result.IsBanned = false
		}
	}

	return result, nil
}
