package handlers

import (
	"net/http"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyEmailRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid verify-email request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(
			w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
			validationErrors,
		)
		return
	}

	if err := h.verifyEmailHandler.Handle(ctx, commands.VerifyEmailCommand{Token: req.Token}); err != nil {
		h.mapErrorAndRespond(w, r, err, "verify-email")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in resend-verification handler")
		middleware.WriteError(
			w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	if err := h.sendVerificationEmailHandler.Handle(ctx, commands.SendVerificationEmailCommand{
		UserID: userCtx.UserID.String(),
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to send verification email")
		middleware.WriteError(
			w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to send verification email. Please try again later.",
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}
