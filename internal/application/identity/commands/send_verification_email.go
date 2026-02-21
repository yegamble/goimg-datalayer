package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const emailVerificationTokenExpiry = 24 * time.Hour

type SendVerificationEmailCommand struct {
	UserID string
}

type SendVerificationEmailHandler struct {
	users       identity.UserRepository
	tokenRepo   identity.TokenRepository
	emailSender appidentity.EmailSender
	logger      zerolog.Logger
}

func NewSendVerificationEmailHandler(
	users identity.UserRepository,
	tokenRepo identity.TokenRepository,
	emailSender appidentity.EmailSender,
	logger zerolog.Logger,
) *SendVerificationEmailHandler {
	return &SendVerificationEmailHandler{
		users:       users,
		tokenRepo:   tokenRepo,
		emailSender: emailSender,
		logger:      logger,
	}
}

func (h *SendVerificationEmailHandler) Handle(ctx context.Context, cmd SendVerificationEmailCommand) error {
	userID, err := identity.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if user.EmailVerified() {
		h.logger.Debug().
			Str("user_id", cmd.UserID).
			Msg("email already verified, skipping verification email")
		return nil
	}

	expiresAt := time.Now().UTC().Add(emailVerificationTokenExpiry)
	token, err := h.tokenRepo.CreateEmailVerificationToken(ctx, user.ID(), expiresAt)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}

	if err := h.emailSender.SendVerificationEmail(ctx, user.Email().String(), token); err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", cmd.UserID).
			Str("email", user.Email().String()).
			Msg("failed to send verification email")
		return nil
	}

	h.logger.Info().
		Str("user_id", cmd.UserID).
		Msg("verification email sent")

	return nil
}
