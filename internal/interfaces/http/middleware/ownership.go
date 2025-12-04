package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ResourceType defines the type of resource being accessed.
type ResourceType string

const (
	// ResourceTypeImage represents an image resource.
	ResourceTypeImage ResourceType = "image"

	// ResourceTypeAlbum represents an album resource.
	ResourceTypeAlbum ResourceType = "album"

	// ResourceTypeComment represents a comment resource.
	ResourceTypeComment ResourceType = "comment"
)

// OwnershipChecker defines the interface for checking resource ownership.
// This is implemented by repository interfaces in the infrastructure layer.
type OwnershipChecker interface {
	// CheckOwnership verifies if the given user owns the specified resource.
	// Returns true if the user owns the resource, false otherwise.
	CheckOwnership(ctx context.Context, userID, resourceID uuid.UUID) (bool, error)

	// ExistsByID checks if a resource exists.
	// Returns true if the resource exists, false otherwise.
	ExistsByID(ctx context.Context, resourceID uuid.UUID) (bool, error)
}

// OwnershipConfig holds configuration for ownership validation middleware.
type OwnershipConfig struct {
	// ResourceType specifies the type of resource being protected.
	ResourceType ResourceType

	// Checker is the interface for verifying resource ownership.
	Checker OwnershipChecker

	// URLParam is the URL parameter name containing the resource ID.
	// Example: "imageID", "albumID", "commentID"
	URLParam string

	// Logger is used to log ownership check events.
	Logger zerolog.Logger

	// AllowAdmins determines if admin users bypass ownership checks.
	// Default: true
	AllowAdmins bool

	// AllowModerators determines if moderator users bypass ownership checks.
	// Default: false (moderators still need ownership)
	AllowModerators bool
}

// RequireOwnership creates middleware that enforces resource ownership.
// This middleware must be placed AFTER JWTAuth middleware.
//
// It performs the following checks:
// 1. Extract user ID from JWT context
// 2. Extract resource ID from URL parameter
// 3. Check if resource exists
// 4. Check if user owns the resource (or has admin/moderator privileges)
// 5. Return 403 Forbidden if not authorized
//
// Usage:
//
//	ownershipCfg := middleware.OwnershipConfig{
//	    ResourceType: middleware.ResourceTypeImage,
//	    Checker:      imageRepository,
//	    URLParam:     "imageID",
//	    Logger:       logger,
//	    AllowAdmins:  true,
//	}
//
//	r.With(middleware.RequireOwnership(ownershipCfg)).
//	    Delete("/api/v1/images/{imageID}", handlers.Image.Delete)
func RequireOwnership(cfg OwnershipConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := GetRequestID(ctx)

			// Step 1: Extract user ID from context (set by JWT middleware)
			userID, ok := GetUserID(ctx)
			if !ok {
				cfg.Logger.Error().
					Str("event", "ownership_check_no_user").
					Str("resource_type", string(cfg.ResourceType)).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("ownership check called without user context")

				WriteError(w, r,
					http.StatusUnauthorized,
					"Unauthorized",
					"User authentication required",
				)
				return
			}

			// Step 2: Extract resource ID from URL parameter
			resourceIDStr := chi.URLParam(r, cfg.URLParam)
			if resourceIDStr == "" {
				cfg.Logger.Error().
					Str("event", "ownership_check_missing_param").
					Str("resource_type", string(cfg.ResourceType)).
					Str("url_param", cfg.URLParam).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("missing resource ID in URL")

				WriteError(w, r,
					http.StatusBadRequest,
					"Bad Request",
					fmt.Sprintf("Missing %s in URL", cfg.URLParam),
				)
				return
			}

			resourceID, err := uuid.Parse(resourceIDStr)
			if err != nil {
				cfg.Logger.Warn().
					Err(err).
					Str("event", "ownership_check_invalid_id").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceIDStr).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("invalid resource ID format")

				WriteError(w, r,
					http.StatusBadRequest,
					"Bad Request",
					fmt.Sprintf("Invalid %s format", cfg.ResourceType),
				)
				return
			}

			// Step 3: Check if resource exists
			exists, err := cfg.Checker.ExistsByID(ctx, resourceID)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("event", "ownership_check_exists_failed").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("request_id", requestID).
					Msg("failed to check resource existence")

				WriteError(w, r,
					http.StatusInternalServerError,
					"Internal Server Error",
					"Failed to verify resource",
				)
				return
			}

			if !exists {
				cfg.Logger.Warn().
					Str("event", "ownership_check_not_found").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("user_id", userID.String()).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("resource not found")

				WriteError(w, r,
					http.StatusNotFound,
					"Not Found",
					fmt.Sprintf("%s not found", cfg.ResourceType),
				)
				return
			}

			// Step 4: Check role-based bypass
			role, _ := GetUserRole(ctx)

			if cfg.AllowAdmins && role == "admin" {
				cfg.Logger.Debug().
					Str("event", "ownership_check_admin_bypass").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("user_id", userID.String()).
					Str("request_id", requestID).
					Msg("admin user bypassing ownership check")

				next.ServeHTTP(w, r)
				return
			}

			if cfg.AllowModerators && role == "moderator" {
				cfg.Logger.Debug().
					Str("event", "ownership_check_moderator_bypass").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("user_id", userID.String()).
					Str("request_id", requestID).
					Msg("moderator user bypassing ownership check")

				next.ServeHTTP(w, r)
				return
			}

			// Step 5: Check ownership
			isOwner, err := cfg.Checker.CheckOwnership(ctx, userID, resourceID)
			if err != nil {
				cfg.Logger.Error().
					Err(err).
					Str("event", "ownership_check_failed").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("user_id", userID.String()).
					Str("request_id", requestID).
					Msg("failed to check ownership")

				WriteError(w, r,
					http.StatusInternalServerError,
					"Internal Server Error",
					"Failed to verify ownership",
				)
				return
			}

			if !isOwner {
				cfg.Logger.Warn().
					Str("event", "ownership_check_denied").
					Str("resource_type", string(cfg.ResourceType)).
					Str("resource_id", resourceID.String()).
					Str("user_id", userID.String()).
					Str("path", r.URL.Path).
					Str("request_id", requestID).
					Msg("ownership check failed - access denied")

				WriteError(w, r,
					http.StatusForbidden,
					"Forbidden",
					fmt.Sprintf("You do not have permission to access this %s", cfg.ResourceType),
				)
				return
			}

			// Step 6: Ownership verified - allow access
			cfg.Logger.Debug().
				Str("event", "ownership_check_success").
				Str("resource_type", string(cfg.ResourceType)).
				Str("resource_id", resourceID.String()).
				Str("user_id", userID.String()).
				Str("request_id", requestID).
				Msg("ownership verified")

			next.ServeHTTP(w, r)
		})
	}
}

// RequireImageOwnership is a convenience function for image ownership checks.
func RequireImageOwnership(checker OwnershipChecker, logger zerolog.Logger) func(http.Handler) http.Handler {
	cfg := OwnershipConfig{
		ResourceType:    ResourceTypeImage,
		Checker:         checker,
		URLParam:        "imageID",
		Logger:          logger,
		AllowAdmins:     true,
		AllowModerators: false,
	}
	return RequireOwnership(cfg)
}

// RequireAlbumOwnership is a convenience function for album ownership checks.
func RequireAlbumOwnership(checker OwnershipChecker, logger zerolog.Logger) func(http.Handler) http.Handler {
	cfg := OwnershipConfig{
		ResourceType:    ResourceTypeAlbum,
		Checker:         checker,
		URLParam:        "albumID",
		Logger:          logger,
		AllowAdmins:     true,
		AllowModerators: false,
	}
	return RequireOwnership(cfg)
}

// RequireCommentOwnership is a convenience function for comment ownership checks.
func RequireCommentOwnership(checker OwnershipChecker, logger zerolog.Logger) func(http.Handler) http.Handler {
	cfg := OwnershipConfig{
		ResourceType:    ResourceTypeComment,
		Checker:         checker,
		URLParam:        "commentID",
		Logger:          logger,
		AllowAdmins:     true,
		AllowModerators: true, // Moderators can delete comments
	}
	return RequireOwnership(cfg)
}
