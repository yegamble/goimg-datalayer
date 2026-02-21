package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

var validate = validator.New()

func DecodeJSON[T any](r *http.Request, v *T) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

func EncodeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

func GetPathParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

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

type UserContext struct {
	UserID        uuid.UUID
	Email         string
	Role          string
	SessionID     uuid.UUID
	TwoFAVerified bool
	EmailVerified bool
}

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

	twofaVerified, _ := middleware.Get2FAVerified(ctx)
	emailVerified, _ := middleware.GetEmailVerified(ctx)

	return &UserContext{
		UserID:        userID,
		Email:         email,
		Role:          role,
		SessionID:     sessionID,
		TwoFAVerified: twofaVerified,
		EmailVerified: emailVerified,
	}, nil
}

func MustGetUserFromContext(ctx context.Context) *UserContext {
	userCtx, err := GetUserFromContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("user context not found: %v - did you forget JWTAuth middleware?", err))
	}
	return userCtx
}

func GetClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		first := forwarded
		if idx := strings.IndexByte(forwarded, ','); idx >= 0 {
			first = forwarded[:idx]
		}
		return normalizeClientIP(first)
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return normalizeClientIP(realIP)
	}

	return normalizeClientIP(r.RemoteAddr)
}

func normalizeClientIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}

	return strings.Trim(value, "[]")
}

func GetUserAgent(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return "unknown"
	}
	return ua
}

func ValidateOwnership(userCtx *UserContext, resourceOwnerID uuid.UUID) bool {
	if userCtx.UserID == resourceOwnerID {
		return true
	}

	if userCtx.Role == "admin" {
		return true
	}

	return false
}

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
		validationErrors["error"] = err.Error()
	}

	return validationErrors
}
