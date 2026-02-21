package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

type ResetPasswordHandler struct {
	tokenRepo    identity.TokenRepository
	users        identity.UserRepository
	sessionStore services.SessionStore
	logger       zerolog.Logger
}

func NewResetPasswordHandler(
	tokenRepo identity.TokenRepository,
	users identity.UserRepository,
	sessionStore services.SessionStore,
	logger zerolog.Logger,
) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		tokenRepo:    tokenRepo,
		users:        users,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd ResetPasswordCommand) error {
	prt, err := h.tokenRepo.FindValidPasswordResetToken(ctx, cmd.Token)
	if err != nil {
		return fmt.Errorf("find password reset token: %w", err)
	}

	newHash, err := identity.NewPasswordHash(cmd.NewPassword)
	if err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}

	user, err := h.users.FindByID(ctx, prt.UserID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if err := user.ChangePassword(newHash); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	if err := h.users.Save(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	if err := h.tokenRepo.MarkPasswordResetTokenUsed(ctx, cmd.Token); err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to mark password reset token as used")
	}

	if err := h.sessionStore.RevokeAll(ctx, user.ID().String()); err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to revoke sessions after password reset")
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Msg("password reset successfully")

	return nil
}
