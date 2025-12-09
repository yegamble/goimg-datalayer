package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// validate is the global validator instance for request validation.
var validate = validator.New()

// DecodeJSON decodes JSON request body into the provided struct and validates it.
// Returns an error if JSON decoding or validation fails.
//
// Usage:
//
//	var req RegisterRequest
//	if err := DecodeJSON(r, &req); err != nil {
//	    middleware.WriteError(w, r, http.StatusBadRequest, "Invalid Request", err.Error())
//	    return
//	}
func DecodeJSON[T any](r *http.Request, v *T) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	// Validate the decoded struct using go-playground/validator
	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// EncodeJSON encodes the provided value as JSON and writes it to the response.
// Sets the Content-Type header to application/json automatically.
//
// Usage:
//
//	user := dto.UserDTO{ID: "123", Email: "user@example.com"}
//	if err := EncodeJSON(w, http.StatusOK, user); err != nil {
//	    logger.Error().Err(err).Msg("failed to encode response")
//	}
func EncodeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

// GetPathParam extracts a path parameter from the chi router context.
// Returns empty string if the parameter is not found.
//
// Usage:
//
//	userID := GetPathParam(r, "id")
//	if userID == "" {
//	    middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Missing user ID")
//	    return
//	}
func GetPathParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// GetPathParamUUID extracts a path parameter and parses it as a UUID.
// Returns an error if the parameter is missing or not a valid UUID.
//
// Usage:
//
//	userID, err := GetPathParamUUID(r, "id")
//	if err != nil {
//	    middleware.WriteError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID format")
//	    return
//	}
func GetPathParamUUID(r *http.Request, name string) (uuid.UUID, error) {
	param := chi.URLParam(r, name)
	if param == "" {
		return uuid.Nil, fmt.Errorf("missing path parameter: %s", name)
	}

	id, err := uuid.Parse(param)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid uuid in path parameter %s: %w", name, err)
	}

	return id, nil
}

// UserContext represents the authenticated user context extracted from JWT claims.
// This is set by the JWTAuth middleware and available in all protected routes.
type UserContext struct {
	UserID    uuid.UUID
	Email     string
	Role      string
	SessionID uuid.UUID
}

// GetUserFromContext extracts the authenticated user information from the request context.
// This function should only be called in routes protected by JWTAuth middleware.
// Returns an error if the user context is not found (middleware not applied).
//
// Usage:
//
//	userCtx, err := GetUserFromContext(r.Context())
//	if err != nil {
//	    middleware.WriteError(w, r, http.StatusUnauthorized, "Unauthorized", "User context not found")
//	    return
//	}
func GetUserFromContext(ctx context.Context) (*UserContext, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user id not found in context")
	}

	email, ok := middleware.GetUserEmail(ctx)
	if !ok {
		return nil, fmt.Errorf("user email not found in context")
	}

	role, ok := middleware.GetUserRole(ctx)
	if !ok {
		return nil, fmt.Errorf("user role not found in context")
	}

	sessionID, ok := middleware.GetSessionID(ctx)
	if !ok {
		return nil, fmt.Errorf("session id not found in context")
	}

	return &UserContext{
		UserID:    userID,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
	}, nil
}

// MustGetUserFromContext extracts the user context or panics if not found.
// This is safe to use in routes where JWTAuth middleware is guaranteed to run.
// Only use this in protected routes where authentication is enforced.
//
// Usage:
//
//	userCtx := MustGetUserFromContext(r.Context())
//	// Use userCtx.UserID, userCtx.Email, etc.
func MustGetUserFromContext(ctx context.Context) *UserContext {
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("user context not found: %v - did you forget JWTAuth middleware?", err))
	}
	return userCtx
}

// GetClientIP extracts the client IP address from the request.
// Respects X-Forwarded-For header if behind a proxy (first IP in the list).
// Falls back to RemoteAddr if X-Forwarded-For is not present.
//
// Security note: Only trust X-Forwarded-For if behind a trusted proxy/load balancer.
// In production, configure this based on your infrastructure.
func GetClientIP(r *http.Request) string {
	// Try X-Forwarded-For first (standard for proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs (client, proxy1, proxy2, ...)
		// First IP is typically the original client
		return forwarded
	}

	// Try X-Real-IP (some proxies use this instead)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to RemoteAddr (direct connection)
	return r.RemoteAddr
}

// GetUserAgent extracts the User-Agent header from the request.
// Returns "unknown" if the header is not present.
func GetUserAgent(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return "unknown"
	}
	return ua
}

// ValidateOwnership checks if the authenticated user owns the resource.
// Returns true if userID matches the authenticated user's ID or if the user is an admin.
//
// Usage:
//
//	if !ValidateOwnership(userCtx, resourceOwnerID) {
//	    middleware.WriteError(w, r, http.StatusForbidden, "Forbidden",
//	        "You do not have permission to access this resource")
//	    return
//	}
func ValidateOwnership(userCtx *UserContext, resourceOwnerID uuid.UUID) bool {
	// Owner can always access
	if userCtx.UserID == resourceOwnerID {
		return true
	}

	// Admins can access any resource
	if userCtx.Role == "admin" {
		return true
	}

	return false
}

// FormatValidationErrors formats go-playground/validator errors into a human-readable map.
// This can be used in the extensions field of RFC 7807 Problem Details.
//
// Usage:
//
//	if err := validate.Struct(req); err != nil {
//	    validationErrors := FormatValidationErrors(err)
//	    middleware.WriteErrorWithExtensions(w, r, http.StatusBadRequest, "Validation Failed",
//	        "Invalid request data", validationErrors)
//	    return
//	}
func FormatValidationErrors(err error) map[string]interface{} {
	validationErrors := make(map[string]interface{})

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			validationErrors[fe.Field()] = map[string]string{
				"tag":   fe.Tag(),
				"value": fe.Param(),
				"error": fe.Error(),
			}
		}
	} else {
		// Non-validator error
		validationErrors["error"] = err.Error()
	}

	return validationErrors
}
