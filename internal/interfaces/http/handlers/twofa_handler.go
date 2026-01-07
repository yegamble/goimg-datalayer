package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// TwoFAHandler handles two-factor authentication HTTP endpoints.
// It delegates to application layer command/query handlers for business logic.
type TwoFAHandler struct {
	setupHandler            *commands.Setup2FAHandler
	verifyHandler           *commands.Verify2FAHandler
	disableHandler          *commands.Disable2FAHandler
	regenerateBackupHandler *commands.RegenerateBackupCodesHandler
	statusQuery             *queries.Get2FAStatusHandler
	logger                  zerolog.Logger
}

// NewTwoFAHandler creates a new TwoFAHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewTwoFAHandler(
	setupHandler *commands.Setup2FAHandler,
	verifyHandler *commands.Verify2FAHandler,
	disableHandler *commands.Disable2FAHandler,
	regenerateBackupHandler *commands.RegenerateBackupCodesHandler,
	statusQuery *queries.Get2FAStatusHandler,
	logger zerolog.Logger,
) *TwoFAHandler {
	return &TwoFAHandler{
		setupHandler:            setupHandler,
		verifyHandler:           verifyHandler,
		disableHandler:          disableHandler,
		regenerateBackupHandler: regenerateBackupHandler,
		statusQuery:             statusQuery,
		logger:                  logger,
	}
}

// Routes registers 2FA routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/auth/2fa
//
// All routes require JWT authentication.
//
// Usage:
//
//	r.Route("/api/v1/auth/2fa", func(r chi.Router) {
//	    r.Use(jwtAuthMiddleware)
//	    r.Mount("/", twofaHandler.Routes())
//	})
func (h *TwoFAHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All routes require authentication (enforced by parent router)
	r.Post("/setup", h.Setup)
	r.Post("/verify", h.Verify)
	r.Post("/disable", h.Disable)
	r.Get("/status", h.Status)
	r.Post("/backup-codes/regenerate", h.RegenerateBackupCodes)

	return r
}

// Setup handles POST /api/v1/auth/2fa/setup
// Initiates 2FA setup by generating a TOTP secret and backup codes.
//
// Request: None (user ID from JWT context)
// Response: 200 OK with Setup2FAResponseDTO (secret, QR URI, backup codes)
// Errors:
//   - 401: Not authenticated
//   - 409: 2FA already enabled
//   - 500: Internal server error
func (h *TwoFAHandler) Setup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (set by JWTAuth middleware)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in 2FA setup handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Delegate to command handler
	cmd := commands.Setup2FACommand{
		UserID: userCtx.UserID.String(),
	}

	response, err := h.setupHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "2FA setup")
		return
	}

	// 3. Setup initiated - return secret and backup codes
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Msg("2FA setup initiated")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode 2FA setup response")
	}
}

// Verify handles POST /api/v1/auth/2fa/verify
// Verifies the TOTP code and enables 2FA for the user.
//
// Request: Verify2FARequest JSON body (6-digit code)
// Response: 200 OK with success message
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Not authenticated
//   - 400: Invalid TOTP code
//   - 404: No pending 2FA setup found
//   - 409: 2FA already enabled
//   - 500: Internal server error
func (h *TwoFAHandler) Verify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in 2FA verify handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request
	var req Verify2FARequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid 2FA verify request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid verification data",
			validationErrors,
		)
		return
	}

	// 3. Delegate to command handler
	cmd := commands.Verify2FACommand{
		UserID: userCtx.UserID.String(),
		Code:   req.Code,
	}

	if err := h.verifyHandler.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "2FA verification")
		return
	}

	// 4. 2FA enabled successfully
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Msg("2FA enabled successfully")

	response := map[string]string{
		"message": "Two-factor authentication has been enabled successfully",
	}
	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode 2FA verify response")
	}
}

// Disable handles POST /api/v1/auth/2fa/disable
// Disables 2FA for the user after password verification.
//
// Request: Disable2FARequest JSON body (password required, TOTP code optional)
// Response: 200 OK with success message
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Not authenticated or invalid password
//   - 400: Invalid TOTP code (if provided)
//   - 404: 2FA not enabled
//   - 500: Internal server error
func (h *TwoFAHandler) Disable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in 2FA disable handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request
	var req Disable2FARequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid 2FA disable request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
			validationErrors,
		)
		return
	}

	// 3. Delegate to command handler
	cmd := commands.Disable2FACommand{
		UserID:   userCtx.UserID.String(),
		Password: req.Password,
		Code:     req.Code,
	}

	if err := h.disableHandler.Handle(ctx, cmd); err != nil {
		h.mapErrorAndRespond(w, r, err, "2FA disable")
		return
	}

	// 4. 2FA disabled successfully
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Msg("2FA disabled successfully")

	response := map[string]string{
		"message": "Two-factor authentication has been disabled",
	}
	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode 2FA disable response")
	}
}

// Status handles GET /api/v1/auth/2fa/status
// Returns the current 2FA status for the authenticated user.
//
// Request: None (user ID from JWT context)
// Response: 200 OK with TwoFactorStatusDTO
// Errors:
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *TwoFAHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in 2FA status handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Delegate to query handler
	query := queries.Get2FAStatusQuery{
		UserID: userCtx.UserID.String(),
	}

	status, err := h.statusQuery.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "2FA status")
		return
	}

	// 3. Return status
	if err := EncodeJSON(w, http.StatusOK, status); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode 2FA status response")
	}
}

// RegenerateBackupCodes handles POST /api/v1/auth/2fa/backup-codes/regenerate
// Generates new backup codes, invalidating existing ones.
//
// Request: RegenerateBackupCodesRequest JSON body (password required)
// Response: 200 OK with BackupCodesResponseDTO (new codes)
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Not authenticated or invalid password
//   - 404: 2FA not enabled
//   - 500: Internal server error
func (h *TwoFAHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in regenerate backup codes handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request
	var req RegenerateBackupCodesRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid regenerate backup codes request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid request data",
			validationErrors,
		)
		return
	}

	// 3. Delegate to command handler
	cmd := commands.RegenerateBackupCodesCommand{
		UserID:   userCtx.UserID.String(),
		Password: req.Password,
	}

	response, err := h.regenerateBackupHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "regenerate backup codes")
		return
	}

	// 4. Return new backup codes
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Int("code_count", response.RemainingCodes).
		Msg("backup codes regenerated")

	if err := EncodeJSON(w, http.StatusOK, response); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode backup codes response")
	}
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
// This centralizes error mapping logic for consistency across all 2FA endpoints.
func (h *TwoFAHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("2FA operation failed")

	// Map specific application errors to HTTP status codes
	switch {
	case errors.Is(err, appidentity.Err2FAAlreadyEnabled):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"Two-factor authentication is already enabled",
		)

	case errors.Is(err, appidentity.Err2FANotEnabled):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"Two-factor authentication is not enabled",
		)

	case errors.Is(err, appidentity.Err2FASetupPending):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Two-factor authentication setup is pending verification",
		)

	case errors.Is(err, appidentity.Err2FAInvalidCode):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid verification code",
		)

	case errors.Is(err, appidentity.Err2FARequired):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Two-factor authentication verification required",
		)

	case errors.Is(err, appidentity.ErrBackupCodeInvalid):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Invalid or already used backup code",
		)

	case errors.Is(err, appidentity.ErrBackupCodesExhausted):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"All backup codes have been used. Please regenerate new codes.",
		)

	case errors.Is(err, appidentity.ErrInvalidCredentials):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Invalid password",
		)

	case errors.Is(err, appidentity.ErrPasswordRequired):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Password confirmation is required for this operation",
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
