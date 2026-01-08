package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// CreateGuestSessionCommand represents the intent to create a guest user session.
// Guest users can upload images without registration. They have limited access
// and their accounts auto-expire after 30 days.
type CreateGuestSessionCommand struct {
	IPAddress string // Required for guest identification and rate limiting
	UserAgent string // Browser/client info for security tracking
}

// CreateGuestSessionHandler processes guest session creation commands.
// It creates a temporary guest user account and issues JWT tokens for immediate use.
type CreateGuestSessionHandler struct {
	users          identity.UserRepository
	eventPublisher appidentity.EventPublisher
	jwtService     appidentity.JWTService   // For generating access tokens
	sessionStore   appidentity.SessionStore // For storing session metadata
	logger         *zerolog.Logger
}

// NewCreateGuestSessionHandler creates a new CreateGuestSessionHandler.
// All dependencies are injected for testability.
func NewCreateGuestSessionHandler(
	users identity.UserRepository,
	eventPublisher appidentity.EventPublisher,
	jwtService appidentity.JWTService,
	sessionStore appidentity.SessionStore,
	logger *zerolog.Logger,
) *CreateGuestSessionHandler {
	return &CreateGuestSessionHandler{
		users:          users,
		eventPublisher: eventPublisher,
		jwtService:     jwtService,
		sessionStore:   sessionStore,
		logger:         logger,
	}
}

// Handle executes the guest session creation use case.
//
// Process flow:
//  1. Validate IP address
//  2. Create guest user via domain factory (NewGuestUser)
//  3. Persist guest user
//  4. Publish domain events
//  5. Generate JWT access token
//  6. Create session in session store
//  7. Return auth response with token
//
// Returns:
//   - AuthResponseDTO with guest user info and access token
//   - Error if IP is invalid or persistence fails
func (h *CreateGuestSessionHandler) Handle(
	ctx context.Context,
	cmd CreateGuestSessionCommand,
) (*dto.AuthResponseDTO, error) {
	// 1. Validate IP address
	if cmd.IPAddress == "" {
		return nil, fmt.Errorf("ip address is required for guest sessions")
	}

	h.logger.Debug().
		Str("ip_address", cmd.IPAddress).
		Str("user_agent", cmd.UserAgent).
		Msg("creating guest session")

	// 2. Create guest user via domain factory
	guestUser, err := identity.NewGuestUser(cmd.IPAddress)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("ip_address", cmd.IPAddress).
			Msg("failed to create guest user")
		return nil, fmt.Errorf("create guest user: %w", err)
	}

	// 3. Persist guest user
	if err := h.users.Save(ctx, guestUser); err != nil {
		h.logger.Error().
			Err(err).
			Str("guest_user_id", guestUser.ID().String()).
			Msg("failed to save guest user")
		return nil, fmt.Errorf("save guest user: %w", err)
	}

	// 4. Publish domain events after successful save
	for _, event := range guestUser.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			// Log but don't fail the operation
			h.logger.Error().
				Err(err).
				Str("event_type", event.EventType()).
				Str("guest_user_id", guestUser.ID().String()).
				Msg("failed to publish guest user event")
		}
	}
	guestUser.ClearEvents()

	// 5. Generate JWT access token
	// Guest sessions use a shorter expiration (24 hours)
	sessionID := identity.NewUserID() // Generate unique session ID

	// Get user ID as UUID for TokenClaims
	userUUID := guestUser.ID().UUID()

	// Create token claims using the correct structure
	claims := &appidentity.TokenClaims{
		UserID:    userUUID,
		Email:     guestUser.Email().String(),
		Role:      guestUser.Role().String(),
		SessionID: sessionID.UUID(),
		TokenType: "access",
	}

	accessToken, err := h.jwtService.GenerateAccessToken(claims)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("guest_user_id", guestUser.ID().String()).
			Msg("failed to generate access token for guest")
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 6. Create session in session store
	expiresAt, err := h.jwtService.GetTokenExpiration(accessToken)
	if err != nil {
		return nil, fmt.Errorf("get token expiration: %w", err)
	}

	session := &appidentity.Session{
		ID:        sessionID.UUID(),
		UserID:    userUUID,
		IPAddress: cmd.IPAddress,
		UserAgent: cmd.UserAgent,
		CreatedAt: guestUser.CreatedAt(),
		ExpiresAt: expiresAt,
	}

	if err := h.sessionStore.Create(ctx, session); err != nil {
		// Non-critical, log but continue
		h.logger.Warn().
			Err(err).
			Str("guest_user_id", guestUser.ID().String()).
			Msg("failed to store guest session")
	}

	h.logger.Info().
		Str("guest_user_id", guestUser.ID().String()).
		Str("ip_address", cmd.IPAddress).
		Msg("guest session created successfully")

	// 7. Return auth response (guest users only get access token, no refresh token)
	userDTO := dto.FromDomain(guestUser)
	tokens := dto.NewTokenPairDTO(accessToken, "", expiresAt) // No refresh token for guests

	response := &dto.AuthResponseDTO{
		User:    userDTO,
		Tokens:  tokens,
		IsGuest: true,
	}

	return response, nil
}
