package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type AuthHandler struct {
	registerHandler              *commands.RegisterUserHandler
	loginHandler                 *commands.LoginHandler
	refreshHandler               *commands.RefreshTokenHandler
	logoutHandler                *commands.LogoutHandler
	guestSessionHandler          *commands.CreateGuestSessionHandler
	forgotPasswordHandler        *commands.RequestPasswordResetHandler
	resetPasswordHandler         *commands.ResetPasswordHandler
	sendVerificationEmailHandler *commands.SendVerificationEmailHandler
	verifyEmailHandler           *commands.VerifyEmailHandler
	logger                       zerolog.Logger
}

func NewAuthHandler(
	registerHandler *commands.RegisterUserHandler,
	loginHandler *commands.LoginHandler,
	refreshHandler *commands.RefreshTokenHandler,
	logoutHandler *commands.LogoutHandler,
	guestSessionHandler *commands.CreateGuestSessionHandler,
	forgotPasswordHandler *commands.RequestPasswordResetHandler,
	resetPasswordHandler *commands.ResetPasswordHandler,
	sendVerificationEmailHandler *commands.SendVerificationEmailHandler,
	verifyEmailHandler *commands.VerifyEmailHandler,
	logger zerolog.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerHandler:              registerHandler,
		loginHandler:                 loginHandler,
		refreshHandler:               refreshHandler,
		logoutHandler:                logoutHandler,
		guestSessionHandler:          guestSessionHandler,
		forgotPasswordHandler:        forgotPasswordHandler,
		resetPasswordHandler:         resetPasswordHandler,
		sendVerificationEmailHandler: sendVerificationEmailHandler,
		verifyEmailHandler:           verifyEmailHandler,
		logger:                       logger,
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/guest", h.CreateGuestSession)

	// Note: Logout requires authentication to identify the user and session
	r.Post("/logout", h.Logout)

	return r
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid register request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid registration data",
			validationErrors,
		)
		return
	}

	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	cmd := commands.RegisterUserCommand{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	userDTO, err := h.registerHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "registration")
		return
	}

	// Note: The RegisterUserHandler should also generate tokens and return AuthResponseDTO
	h.logger.Info().
		Str("user_id", userDTO.ID).
		Str("email", userDTO.Email).
		Str("username", userDTO.Username).
		Str("ip_address", ipAddress).
		Msg("user registered successfully")

	if err := EncodeJSON(w, http.StatusCreated, userDTO); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode register response")
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid login request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid login data",
			validationErrors,
		)
		return
	}

	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	cmd := commands.LoginCommand{
		Identifier: req.Email,
		Password:   req.Password,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	authResponse, err := h.loginHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "login")
		return
	}

	h.logger.Info().
		Str("user_id", authResponse.User.ID).
		Str("email", authResponse.User.Email).
		Str("ip_address", ipAddress).
		Msg("user logged in successfully")

	if err := EncodeJSON(w, http.StatusOK, authResponse); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode login response")
	}
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RefreshRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid refresh request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid refresh token data",
			validationErrors,
		)
		return
	}

	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	cmd := commands.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	tokenPair, err := h.refreshHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "token refresh")
		return
	}

	h.logger.Info().
		Str("ip_address", ipAddress).
		Msg("token refreshed successfully")

	if err := EncodeJSON(w, http.StatusOK, tokenPair); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode refresh response")
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in logout handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.logger.Debug().Err(err).Msg("invalid logout request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid request body",
		)
		return
	}

	authHeader := r.Header.Get("Authorization")
	accessToken := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}

	cmd := commands.LogoutCommand{
		UserID:       userCtx.UserID.String(),
		SessionID:    userCtx.SessionID.String(),
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		LogoutAll:    req.LogoutAll,
	}

	if err := h.logoutHandler.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "logout")
		return
	}

	logType := "single session"
	if req.LogoutAll {
		logType = "all sessions"
	}

	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Str("session_id", userCtx.SessionID.String()).
		Str("logout_type", logType).
		Msg("user logged out successfully")

	w.WriteHeader(http.StatusNoContent)
}

//nolint:funlen,cyclop // Comprehensive error mapping for all authentication error types.
func (h *AuthHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("authentication operation failed")

	switch {
	case errors.Is(err, appidentity.ErrEmailAlreadyExists):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Email address is already registered",
		)

	case errors.Is(err, appidentity.ErrUsernameAlreadyExists):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Username is already taken",
		)

	case errors.Is(err, appidentity.ErrInvalidCredentials):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Invalid email or password",
		)

	case errors.Is(err, appidentity.ErrAccountSuspended):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Account has been suspended. Please contact support.",
		)

	case errors.Is(err, appidentity.ErrAccountLocked):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Account temporarily locked due to multiple failed login attempts. Please try again later.",
		)

	case errors.Is(err, appidentity.ErrAccountDeleted):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Account has been deleted",
		)

	case errors.Is(err, appidentity.ErrInvalidToken):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Invalid or expired token",
		)

	case errors.Is(err, appidentity.ErrTokenExpired):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Token has expired",
		)

	case errors.Is(err, appidentity.ErrTokenRevoked):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Token has been revoked. Please log in again.",
		)

	case errors.Is(err, appidentity.ErrTokenReplayDetected):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Security Alert",
			"Token replay detected. All sessions have been revoked for security. Please log in again.",
		)

	case errors.Is(err, appidentity.ErrSessionNotFound):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Session not found or expired",
		)

	case errors.Is(err, identity.ErrPasswordCompromised):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Password Compromised",
			"This password has been found in a data breach and cannot be used. Please choose a different, stronger password.",
		)

	case errors.Is(err, appidentity.ErrPasswordResetTokenInvalid),
		errors.Is(err, appidentity.ErrPasswordResetTokenUsed),
		errors.Is(err, identity.ErrTokenNotFound):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid or expired password reset token",
		)

	case errors.Is(err, identity.ErrEmailInvalid),
		errors.Is(err, identity.ErrEmailEmpty),
		errors.Is(err, identity.ErrEmailTooLong),
		errors.Is(err, identity.ErrUsernameInvalid),
		errors.Is(err, identity.ErrUsernameEmpty),
		errors.Is(err, identity.ErrUsernameTooShort),
		errors.Is(err, identity.ErrUsernameTooLong),
		errors.Is(err, identity.ErrPasswordEmpty),
		errors.Is(err, identity.ErrPasswordTooShort),
		errors.Is(err, identity.ErrPasswordTooLong),
		errors.Is(err, identity.ErrPasswordWeak):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			err.Error(),
		)

	default:
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
	}
}

func (h *AuthHandler) CreateGuestSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	h.logger.Debug().
		Str("ip_address", ipAddress).
		Str("user_agent", userAgent).
		Msg("guest session creation request")

	cmd := commands.CreateGuestSessionCommand{
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	authResponse, err := h.guestSessionHandler.Handle(ctx, cmd)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("ip_address", ipAddress).
			Msg("guest session creation failed")

		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to create guest session. Please try again later.",
		)
		return
	}

	h.logger.Info().
		Str("guest_user_id", authResponse.User.ID).
		Str("ip_address", ipAddress).
		Msg("guest session created successfully")

	if err := EncodeJSON(w, http.StatusCreated, authResponse); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode guest session response")
	}
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ForgotPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid forgot-password request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
			validationErrors,
		)
		return
	}

	if err := h.forgotPasswordHandler.Handle(ctx, commands.RequestPasswordResetCommand{Email: req.Email}); err != nil {
		h.logger.Error().Err(err).Msg("forgot-password handler failed")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResetPasswordRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid reset-password request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
			validationErrors,
		)
		return
	}

	if err := h.resetPasswordHandler.Handle(ctx, commands.ResetPasswordCommand{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		h.mapErrorAndRespond(w, r, err, "reset-password")
		return
	}

	h.logger.Info().Msg("password reset successfully")
	w.WriteHeader(http.StatusNoContent)
}
