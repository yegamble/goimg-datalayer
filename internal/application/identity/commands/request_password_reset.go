package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const passwordResetTokenExpiry = time.Hour

type RequestPasswordResetCommand struct {
	Email string
}

type RequestPasswordResetHandler struct {
	users       identity.UserRepository
	tokenRepo   identity.TokenRepository
	emailSender appidentity.EmailSender
	logger      *zerolog.Logger
}

func NewRequestPasswordResetHandler(
	users identity.UserRepository,
	tokenRepo identity.TokenRepository,
	emailSender appidentity.EmailSender,
	logger *zerolog.Logger,
) *RequestPasswordResetHandler {
	return &RequestPasswordResetHandler{
		users:       users,
		tokenRepo:   tokenRepo,
		emailSender: emailSender,
		logger:      logger,
	}
}

func (h *RequestPasswordResetHandler) Handle(ctx context.Context, cmd RequestPasswordResetCommand) error {
	email, err := identity.NewEmail(cmd.Email)
	if err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	user, err := h.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			h.logger.Debug().
				Str("email", cmd.Email).
				Msg("password reset requested for non-existent email")
			return nil
		}
		return fmt.Errorf("find user by email: %w", err)
	}

	if err := h.tokenRepo.InvalidateAllPasswordResetTokens(ctx, user.ID()); err != nil {
		return fmt.Errorf("invalidate existing reset tokens: %w", err)
	}

	expiresAt := time.Now().UTC().Add(passwordResetTokenExpiry)
	token, err := h.tokenRepo.CreatePasswordResetToken(ctx, user.ID(), expiresAt)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}

	if err := h.emailSender.SendPasswordResetEmail(ctx, email.String(), token); err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("email", email.String()).
			Msg("failed to send password reset email")
		return nil
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Msg("password reset email sent")

	return nil
}
