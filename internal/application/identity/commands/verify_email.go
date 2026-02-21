package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

type VerifyEmailCommand struct {
	Token string
}

type VerifyEmailHandler struct {
	tokenRepo identity.TokenRepository
	users     identity.UserRepository
	logger    zerolog.Logger
}

func NewVerifyEmailHandler(
	tokenRepo identity.TokenRepository,
	users identity.UserRepository,
	logger zerolog.Logger,
) *VerifyEmailHandler {
	return &VerifyEmailHandler{
		tokenRepo: tokenRepo,
		users:     users,
		logger:    logger,
	}
}

func (h *VerifyEmailHandler) Handle(ctx context.Context, cmd VerifyEmailCommand) error {
	evToken, err := h.tokenRepo.FindValidEmailVerificationToken(ctx, cmd.Token)
	if err != nil {
		return fmt.Errorf("find email verification token: %w", err)
	}

	user, err := h.users.FindByID(ctx, evToken.UserID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	user.VerifyEmail()

	if err := h.users.Save(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	if err := h.tokenRepo.MarkEmailVerificationTokenUsed(ctx, cmd.Token); err != nil {
		h.logger.Warn().
			Err(err).
			Str("user_id", user.ID().String()).
			Msg("failed to mark email verification token as used")
	}

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Msg("email verified successfully")

	return nil
}
