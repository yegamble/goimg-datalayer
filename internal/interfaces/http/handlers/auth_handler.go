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

// AuthHandler handles authentication-related HTTP endpoints.
// It delegates to application layer command handlers for business logic.
type AuthHandler struct {
	registerHandler *commands.RegisterUserHandler
	loginHandler    *commands.LoginHandler
	refreshHandler  *commands.RefreshTokenHandler
	logoutHandler   *commands.LogoutHandler
	logger          zerolog.Logger
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewAuthHandler(
	registerHandler *commands.RegisterUserHandler,
	loginHandler *commands.LoginHandler,
	refreshHandler *commands.RefreshTokenHandler,
	logoutHandler *commands.LogoutHandler,
	logger zerolog.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerHandler: registerHandler,
		loginHandler:    loginHandler,
		refreshHandler:  refreshHandler,
		logoutHandler:   logoutHandler,
		logger:          logger,
	}
}

// Routes registers authentication routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/auth
//
// Usage:
//
//	r.Mount("/api/v1/auth", authHandler.Routes())
//
//nolint:ireturn // Returning chi.Router interface is chi's standard pattern for sub-routers
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no authentication required)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)

	// Protected route (JWT authentication required)
	// Note: Logout requires authentication to identify the user and session
	r.Post("/logout", h.Logout)

	return r
}

// Register handles POST /api/v1/auth/register
// Creates a new user account and returns user data with authentication tokens.
//
// Request: RegisterRequest JSON body
// Response: 201 Created with AuthResponseDTO
// Errors:
//   - 400: Invalid request body or validation failure
//   - 409: Email or username already exists
//   - 500: Internal server error
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode and validate request
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

	// 2. Extract client metadata for security auditing
	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	// 3. Delegate to command handler
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

	// 4. Registration successful - return user DTO
	// Note: The RegisterUserHandler should also generate tokens and return AuthResponseDTO
	// For now, returning just the UserDTO as per the current implementation
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

// Login handles POST /api/v1/auth/login
// Authenticates a user and returns authentication tokens.
//
// Request: LoginRequest JSON body
// Response: 200 OK with AuthResponseDTO (user + tokens)
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Invalid credentials
//   - 403: Account suspended or locked
//   - 500: Internal server error
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode and validate request
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

	// 2. Extract client metadata for security auditing
	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	// 3. Delegate to command handler
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

	// 4. Login successful - return auth response with tokens
	h.logger.Info().
		Str("user_id", authResponse.User.ID).
		Str("email", authResponse.User.Email).
		Str("ip_address", ipAddress).
		Msg("user logged in successfully")

	if err := EncodeJSON(w, http.StatusOK, authResponse); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode login response")
	}
}

// Refresh handles POST /api/v1/auth/refresh
// Exchanges a refresh token for a new access token (token rotation).
//
// Request: RefreshRequest JSON body
// Response: 200 OK with TokenPairDTO (new access + refresh tokens)
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Invalid, expired, or revoked refresh token
//   - 403: Token replay detected (security incident)
//   - 500: Internal server error
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Decode and validate request
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

	// 2. Extract client metadata for anomaly detection
	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	// 3. Delegate to command handler
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

	// 4. Refresh successful - return new token pair
	h.logger.Info().
		Str("ip_address", ipAddress).
		Msg("token refreshed successfully")

	if err := EncodeJSON(w, http.StatusOK, tokenPair); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode refresh response")
	}
}

// Logout handles POST /api/v1/auth/logout
// Invalidates the current session and optionally all user sessions.
//
// This endpoint requires JWT authentication (protected route).
// The access token is extracted from the Authorization header by middleware.
//
// Request: LogoutRequest JSON body (optional refresh_token and logout_all flag)
// Response: 204 No Content
// Errors:
//   - 400: Invalid request body
//   - 401: Missing or invalid authentication token
//   - 500: Internal server error
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (set by JWTAuth middleware)
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

	// 2. Decode logout request (both fields are optional)
	var req LogoutRequest
	// Use custom decoding without validation since both fields are optional
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.logger.Debug().Err(err).Msg("invalid logout request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid request body",
		)
		return
	}

	// 3. Extract access token from Authorization header
	authHeader := r.Header.Get("Authorization")
	accessToken := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}

	// 4. Delegate to command handler
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

	// 5. Logout successful - return 204 No Content
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

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
// This centralizes error mapping logic for consistency across all auth endpoints.
//
//nolint:funlen,cyclop // Comprehensive error mapping for all authentication error types.
func (h *AuthHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("authentication operation failed")

	// Map specific application errors to HTTP status codes
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
		// Unknown error - return generic 500 without exposing internal details
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"An unexpected error occurred. Please try again later.",
		)
	}
}
