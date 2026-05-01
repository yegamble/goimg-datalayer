package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	domainidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// OAuthHandler handles OAuth 2.0 authentication endpoints.
// It delegates to application layer command handlers for business logic.
type OAuthHandler struct {
	authenticateHandler *commands.AuthenticateWithOAuthHandler
	linkHandler         *commands.LinkOAuthAccountHandler
	unlinkHandler       *commands.UnlinkOAuthAccountHandler
	listAccountsHandler *queries.ListOAuthAccountsHandler
	providerFactory     commands.OAuthProviderFactory
	redisClient         *redis.Client
	logger              zerolog.Logger
}

// NewOAuthHandler creates a new OAuthHandler with the given dependencies.
// All dependencies are injected via constructor for testability.
func NewOAuthHandler(
	authenticateHandler *commands.AuthenticateWithOAuthHandler,
	linkHandler *commands.LinkOAuthAccountHandler,
	unlinkHandler *commands.UnlinkOAuthAccountHandler,
	listAccountsHandler *queries.ListOAuthAccountsHandler,
	providerFactory commands.OAuthProviderFactory,
	redisClient *redis.Client,
	logger zerolog.Logger,
) *OAuthHandler {
	return &OAuthHandler{
		authenticateHandler: authenticateHandler,
		linkHandler:         linkHandler,
		unlinkHandler:       unlinkHandler,
		listAccountsHandler: listAccountsHandler,
		providerFactory:     providerFactory,
		redisClient:         redisClient,
		logger:              logger,
	}
}

// Routes registers OAuth routes with the chi router.
// Returns a chi.Router that can be mounted under /api/v1/auth/oauth
//
// Public routes:
// - GET /{provider} - Initiate OAuth flow
// - GET /{provider}/callback - Handle OAuth callback
//
// Protected routes (JWT authentication required):
// - POST /link - Link OAuth account to existing user
// - DELETE /{provider} - Unlink OAuth account
// - GET /accounts - List linked OAuth accounts
//
// Usage:
//
//	r.Route("/api/v1/auth/oauth", func(r chi.Router) {
//	    r.Mount("/", oauthHandler.Routes(jwtAuthMiddleware))
//	})
func (h *OAuthHandler) Routes(jwtAuthMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no authentication required)
	r.Get("/{provider}", h.InitiateOAuth)
	r.Get("/{provider}/callback", h.HandleCallback)

	// Protected routes (JWT authentication required)
	r.Group(func(r chi.Router) {
		r.Use(jwtAuthMiddleware)
		r.Post("/link", h.LinkAccount)
		r.Delete("/{provider}", h.UnlinkAccount)
		r.Get("/accounts", h.ListAccounts)
	})

	return r
}

// InitiateOAuth handles GET /api/v1/auth/oauth/{provider}
// Generates OAuth authorization URL with CSRF state and redirects user to provider.
//
// Path Parameters:
// - provider: OAuth provider (google, github)
//
// Response: 302 redirect to OAuth provider authorization URL
// Errors:
//   - 400: Invalid provider
//   - 500: Failed to generate state or store in Redis
func (h *OAuthHandler) InitiateOAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate provider from path
	providerStr := chi.URLParam(r, "provider")
	provider, err := domainidentity.ParseOAuthProvider(providerStr)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("provider", providerStr).
			Msg("invalid OAuth provider")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("Invalid OAuth provider: %s", providerStr),
		)
		return
	}

	// 2. Generate cryptographically secure CSRF state
	state, err := h.generateCSRFState()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate CSRF state")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to initiate OAuth flow",
		)
		return
	}

	// 3. Store state in Redis with 10-minute TTL (CSRF protection)
	stateKey := fmt.Sprintf("goimg:oauth:state:%s", state)
	err = h.redisClient.Set(ctx, stateKey, provider.String(), 10*time.Minute).Err()
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("state", state).
			Msg("failed to store OAuth state in Redis")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to initiate OAuth flow",
		)
		return
	}

	// 4. Create OAuth provider instance and get authorization URL
	oauthProvider, err := h.providerFactory.CreateProvider(provider)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("provider", provider.String()).
			Msg("failed to create OAuth provider")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to initiate OAuth flow",
		)
		return
	}

	authURL := oauthProvider.GetAuthorizationURL(state)

	h.logger.Info().
		Str("provider", provider.String()).
		Str("state", state).
		Msg("OAuth flow initiated, redirecting to provider")

	// 5. Redirect user to OAuth provider authorization page
	// #nosec G710 // URL generated by trusted library using configured provider endpoints
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback handles GET /api/v1/auth/oauth/{provider}/callback
// Processes OAuth callback from provider, exchanges code for tokens, and authenticates user.
//
// Path Parameters:
// - provider: OAuth provider (google, github)
//
// Query Parameters:
// - code: Authorization code from OAuth provider
// - state: CSRF state parameter
//
// Response: 200 OK with AuthResponseDTO (user + tokens)
// Errors:
//   - 400: Missing code or state, invalid provider, state mismatch
//   - 401: OAuth authentication failed
//   - 500: Internal server error
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract provider from path
	providerStr := chi.URLParam(r, "provider")
	provider, err := domainidentity.ParseOAuthProvider(providerStr)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("provider", providerStr).
			Msg("invalid OAuth provider in callback")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("Invalid OAuth provider: %s", providerStr),
		)
		return
	}

	// 2. Extract query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		h.logger.Warn().
			Str("provider", provider.String()).
			Msg("OAuth callback missing authorization code")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing authorization code",
		)
		return
	}

	if state == "" {
		h.logger.Warn().
			Str("provider", provider.String()).
			Msg("OAuth callback missing state parameter")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"Missing state parameter",
		)
		return
	}

	// 3. Validate CSRF state from Redis
	stateKey := fmt.Sprintf("goimg:oauth:state:%s", state)
	storedProvider, err := h.redisClient.Get(ctx, stateKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			h.logger.Warn().
				Str("state", state).
				Str("provider", provider.String()).
				Msg("OAuth state not found or expired")
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid or expired state parameter",
			)
			return
		}

		h.logger.Error().
			Err(err).
			Str("state", state).
			Msg("failed to retrieve OAuth state from Redis")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to validate OAuth state",
		)
		return
	}

	// 4. Verify state matches expected provider
	if storedProvider != provider.String() {
		h.logger.Warn().
			Str("state", state).
			Str("expected_provider", storedProvider).
			Str("actual_provider", provider.String()).
			Msg("OAuth state provider mismatch")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"State parameter mismatch",
		)
		return
	}

	// 5. Delete state from Redis (one-time use)
	_ = h.redisClient.Del(ctx, stateKey).Err()

	// 6. Extract client metadata for security auditing
	ipAddress := GetClientIP(r)
	userAgent := GetUserAgent(r)

	// 7. Delegate to command handler to authenticate via OAuth
	cmd := commands.AuthenticateWithOAuthCommand{
		Provider:  provider,
		Code:      code,
		State:     state,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	authResponse, err := h.authenticateHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "OAuth authentication")
		return
	}

	// 8. OAuth authentication successful - return user and tokens
	h.logger.Info().
		Str("user_id", authResponse.User.ID).
		Str("email", authResponse.User.Email).
		Str("provider", provider.String()).
		Str("ip_address", ipAddress).
		Msg("OAuth authentication successful")

	if err := EncodeJSON(w, http.StatusOK, authResponse); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode OAuth authentication response")
	}
}

// LinkAccountRequest represents the request to link an OAuth account.
type LinkAccountRequest struct {
	Provider string `json:"provider" validate:"required,oneof=google github"`
	Code     string `json:"code" validate:"required"`
	State    string `json:"state" validate:"required"`
}

// LinkAccount handles POST /api/v1/auth/oauth/link
// Links an OAuth account to the authenticated user.
//
// This endpoint requires JWT authentication.
//
// Request: LinkAccountRequest JSON body
// Response: 200 OK with OAuthAccountDTO
// Errors:
//   - 400: Invalid request body or validation failure
//   - 401: Not authenticated
//   - 409: OAuth account already linked to another user
//   - 500: Internal server error
func (h *OAuthHandler) LinkAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context (set by JWTAuth middleware)
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in OAuth link handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Decode and validate request
	var req LinkAccountRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.logger.Debug().Err(err).Msg("invalid OAuth link request")
		validationErrors := FormatValidationErrors(err)
		middleware.WriteErrorWithExtensions(w, r,
			http.StatusBadRequest,
			"Validation Failed",
			"Invalid OAuth link data",
			validationErrors,
		)
		return
	}

	// 3. Parse provider
	provider, err := domainidentity.ParseOAuthProvider(req.Provider)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("provider", req.Provider).
			Msg("invalid OAuth provider in link request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("Invalid OAuth provider: %s", req.Provider),
		)
		return
	}

	// 4. Validate CSRF state (same as callback)
	stateKey := fmt.Sprintf("goimg:oauth:state:%s", req.State)
	storedProvider, err := h.redisClient.Get(ctx, stateKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			h.logger.Warn().
				Str("state", req.State).
				Str("user_id", userCtx.UserID.String()).
				Msg("OAuth state not found or expired for link")
			middleware.WriteError(w, r,
				http.StatusBadRequest,
				"Bad Request",
				"Invalid or expired state parameter",
			)
			return
		}

		h.logger.Error().
			Err(err).
			Str("state", req.State).
			Msg("failed to retrieve OAuth state from Redis for link")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to validate OAuth state",
		)
		return
	}

	if storedProvider != provider.String() {
		h.logger.Warn().
			Str("state", req.State).
			Str("expected_provider", storedProvider).
			Str("actual_provider", provider.String()).
			Msg("OAuth state provider mismatch for link")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			"State parameter mismatch",
		)
		return
	}

	// 5. Delete state from Redis (one-time use)
	_ = h.redisClient.Del(ctx, stateKey).Err()

	// 6. Parse user ID from context
	userID, err := domainidentity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userCtx.UserID.String()).
			Msg("failed to parse user ID in OAuth link handler")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user ID",
		)
		return
	}

	// 7. Delegate to command handler
	cmd := commands.LinkOAuthAccountCommand{
		UserID:   userID,
		Provider: provider,
		Code:     req.Code,
		State:    req.State,
	}

	oauthAccount, err := h.linkHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "OAuth account linking")
		return
	}

	// 8. OAuth account linked successfully
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Str("provider", provider.String()).
		Msg("OAuth account linked successfully")

	if err := EncodeJSON(w, http.StatusOK, oauthAccount); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode OAuth link response")
	}
}

// UnlinkAccount handles DELETE /api/v1/auth/oauth/{provider}
// Unlinks an OAuth account from the authenticated user.
//
// This endpoint requires JWT authentication.
//
// Path Parameters:
// - provider: OAuth provider to unlink (google, github)
//
// Response: 200 OK with success message
// Errors:
//   - 400: Invalid provider
//   - 401: Not authenticated
//   - 404: OAuth provider not linked
//   - 400: Cannot unlink last authentication method
//   - 500: Internal server error
func (h *OAuthHandler) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in OAuth unlink handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Extract and validate provider from path
	providerStr := chi.URLParam(r, "provider")
	provider, err := domainidentity.ParseOAuthProvider(providerStr)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("provider", providerStr).
			Msg("invalid OAuth provider in unlink request")
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
			fmt.Sprintf("Invalid OAuth provider: %s", providerStr),
		)
		return
	}

	// 3. Parse user ID from context
	userID, err := domainidentity.ParseUserID(userCtx.UserID.String())
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", userCtx.UserID.String()).
			Msg("failed to parse user ID in OAuth unlink handler")
		middleware.WriteError(w, r,
			http.StatusInternalServerError,
			"Internal Server Error",
			"Invalid user ID",
		)
		return
	}

	// 4. Delegate to command handler
	cmd := commands.UnlinkOAuthAccountCommand{
		UserID:   userID,
		Provider: provider,
	}

	message, err := h.unlinkHandler.Handle(ctx, cmd)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "OAuth account unlinking")
		return
	}

	// 5. OAuth account unlinked successfully
	h.logger.Info().
		Str("user_id", userCtx.UserID.String()).
		Str("provider", provider.String()).
		Msg("OAuth account unlinked successfully")

	if err := EncodeJSON(w, http.StatusOK, message); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode OAuth unlink response")
	}
}

// ListAccounts handles GET /api/v1/auth/oauth/accounts
// Returns all OAuth accounts linked to the authenticated user.
//
// This endpoint requires JWT authentication.
//
// Response: 200 OK with OAuthAccountListDTO
// Errors:
//   - 401: Not authenticated
//   - 500: Internal server error
func (h *OAuthHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract user context
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("user context not found in OAuth list accounts handler")
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"Authentication required",
		)
		return
	}

	// 2. Delegate to query handler
	query := queries.ListOAuthAccountsQuery{
		UserID: userCtx.UserID.String(),
	}

	accountList, err := h.listAccountsHandler.Handle(ctx, query)
	if err != nil {
		h.mapErrorAndRespond(w, r, err, "list OAuth accounts")
		return
	}

	// 3. Return account list
	if err := EncodeJSON(w, http.StatusOK, accountList); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode OAuth accounts list response")
	}
}

// generateCSRFState generates a cryptographically secure random state for CSRF protection.
// Returns a base64-encoded 32-byte random string.
func (h *OAuthHandler) generateCSRFState() (string, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(stateBytes), nil
}

// mapErrorAndRespond maps application/domain errors to HTTP responses using RFC 7807 Problem Details.
// This centralizes error mapping logic for consistency across all OAuth endpoints.
func (h *OAuthHandler) mapErrorAndRespond(w http.ResponseWriter, r *http.Request, err error, operation string) {
	h.logger.Error().
		Err(err).
		Str("operation", operation).
		Msg("OAuth operation failed")

	// Map specific application/domain errors to HTTP status codes
	switch {
	case errors.Is(err, domainidentity.ErrOAuthAccountNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"OAuth account not found",
		)

	case errors.Is(err, domainidentity.ErrOAuthAccountExists):
		middleware.WriteError(w, r,
			http.StatusConflict,
			"Conflict",
			"OAuth account is already linked to another user",
		)

	case errors.Is(err, domainidentity.ErrOAuthProviderNotLinked):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"OAuth provider is not linked to your account",
		)

	case errors.Is(err, domainidentity.ErrUserNotFound):
		middleware.WriteError(w, r,
			http.StatusNotFound,
			"Not Found",
			"User not found",
		)

	case errors.Is(err, appidentity.ErrAccountSuspended):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Account has been suspended. Please contact support.",
		)

	case errors.Is(err, appidentity.ErrAccountDeleted):
		middleware.WriteError(w, r,
			http.StatusForbidden,
			"Forbidden",
			"Account has been deleted",
		)

	case errors.Is(err, appidentity.ErrInvalidCredentials):
		middleware.WriteError(w, r,
			http.StatusUnauthorized,
			"Unauthorized",
			"OAuth authentication failed",
		)

	// Check for error messages indicating last auth method
	case err != nil && (err.Error() == "cannot unlink last authentication method" ||
		err.Error() == "user account is not active"):
		middleware.WriteError(w, r,
			http.StatusBadRequest,
			"Bad Request",
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
