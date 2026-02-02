package handlers

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MiddlewareConfig holds configuration for HTTP middleware.
type MiddlewareConfig struct {
	// JWTService for token validation
	JWTService middleware.JWTServiceInterface

	// TokenBlacklist for revoked token checking
	TokenBlacklist middleware.TokenBlacklistInterface

	// Logger for structured logging
	Logger zerolog.Logger

	// RateLimiterConfig for rate limiting middleware (optional)
	// If nil, rate limiting is disabled
	RateLimiterConfig *middleware.RateLimiterConfig
}

// NewRouter creates a new chi router with all routes and middleware configured.
// This is the main entry point for HTTP routing.
//
// Middleware order (CRITICAL for security):
//  1. RequestID - generates correlation ID
//  2. Metrics - Prometheus metrics collection
//  3. Logger - structured request/response logging
//  4. Recovery - panic recovery
//  5. SecurityHeaders - defense headers (CSP, X-Frame-Options, etc.)
//  6. CORS - cross-origin resource sharing
//
// Route groups:
//   - Health/Metrics routes: /health, /health/ready, /metrics (no authentication)
//   - Public routes: /api/v1/auth/* (no authentication)
//   - Protected routes: /api/v1/users/*, /api/v1/images/*, /api/v1/albums/* (JWT authentication required)
//   - 2FA routes: /api/v1/auth/2fa/* (JWT authentication required)
//   - OAuth routes: /api/v1/auth/oauth/* (public initiate/callback, protected link/unlink/list)
//   - Social routes: /api/v1/images/{id}/likes, /api/v1/images/{id}/comments (JWT authentication required)
//   - Follow routes: POST/DELETE /api/v1/users/{id}/follow (JWT required), GET /api/v1/users/{id}/followers|following (optional auth)
//   - Notification routes: GET /api/v1/notifications, GET /api/v1/notifications/count, POST /api/v1/notifications/read (JWT required)
//   - IPFS routes: POST/DELETE/GET /api/v1/images/{id}/ipfs (JWT required, pin/unpin owner only)
//   - Moderation routes: POST /api/v1/reports (JWT required), moderator/admin: /api/v1/moderation/reports/*, /api/v1/moderation/nsfw/*, admin only: /api/v1/users/{id}/ban, /api/v1/moderation/bans, /api/v1/moderation/featured/*
//   - Guest routes: POST /api/v1/guest/images/{id}/claim (JWT required, claim guest uploads)
//   - oEmbed routes: GET /api/v1/oembed (public, no auth required) - Sprint 16
//   - Preview routes: GET /images/{id}/preview (public HTML page with social meta tags) - Sprint 16
//   - Variant Config routes: /api/v1/variant-configs/* (JWT required) - Sprint 17
//   - Tag routes: GET /api/v1/tags/* (public, no auth required) - Sprint 19
//   - Group routes: /api/v1/groups/* (mixed public/protected, Sprint 20)
//
//nolint:funlen // Router setup with middleware and routes.
func NewRouter(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	imageHandler *ImageHandler,
	albumHandler *AlbumHandler,
	socialHandler *SocialHandler,
	exploreHandler *ExploreHandler,
	healthHandler *HealthHandler,
	twoFAHandler *TwoFAHandler,
	oauthHandler *OAuthHandler,
	followHandler *FollowHandler,
	activityHandler *ActivityHandler,
	notificationHandler *NotificationHandler,
	ipfsHandler *IPFSHandler,
	moderationHandler *ModerationHandler,
	guestHandler *GuestHandler,
	oembedHandler *OEmbedHandler,
	previewHandler *PreviewHandler,
	variantConfigHandler *VariantConfigHandler,
	tagHandler *TagHandler,
	featuredHandler *FeaturedHandler,
	groupHandler *GroupHandler,
	groupAlbumHandler *GroupAlbumHandler,
	metricsCollector *middleware.MetricsCollector,
	middlewareConfig MiddlewareConfig,
	isProd bool,
) chi.Router {
	r := chi.NewRouter()

	// Global middleware (applies to all routes)
	r.Use(middleware.RequestID)
	r.Use(middleware.MetricsMiddleware(metricsCollector))
	r.Use(middleware.Logger(middlewareConfig.Logger))
	r.Use(middleware.Recovery(middlewareConfig.Logger))

	// Security headers with production config
	securityCfg := middleware.DefaultSecurityHeadersConfig(isProd)
	r.Use(middleware.SecurityHeaders(securityCfg))

	// CORS with appropriate config
	var corsCfg middleware.CORSConfig
	if isProd {
		corsCfg = middleware.DefaultCORSConfig()
	} else {
		corsCfg = middleware.DevelopmentCORSConfig()
	}
	r.Use(middleware.CORS(corsCfg))

	// Timeout middleware (prevent long-running requests)
	r.Use(chimiddleware.Timeout(contextTimeout * time.Second))

	// Health check endpoints (no authentication required)
	if healthHandler != nil {
		// Liveness probe - checks if server is running
		r.Get("/health", healthHandler.Liveness)

		// Readiness probe - checks if all dependencies (DB, Redis) are healthy
		r.Get("/health/ready", healthHandler.Readiness)
	}

	// Prometheus metrics endpoint (no authentication required)
	// In production, consider adding basic auth or IP restriction
	r.Handle("/metrics", promhttp.Handler())

	// Image preview endpoint (Sprint 16 - no authentication required)
	// Public HTML page with Open Graph and Twitter Card meta tags
	// Mounted at /images/{id}/preview (not under /api/v1 for SEO-friendly URLs)
	if previewHandler != nil {
		r.Mount("/images", previewHandler.Routes())
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no authentication required)
		// Most auth routes are public, but guest session creation is rate-limited
		if authHandler != nil {
			r.Route("/auth", func(r chi.Router) {
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
				r.Post("/refresh", authHandler.Refresh)
				r.Post("/logout", authHandler.Logout)

				// Guest session creation with IP-based rate limiting (10/hour per IP)
				// Prevents abuse of anonymous upload feature
				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.GuestSessionRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/guest", authHandler.CreateGuestSession)
				} else {
					r.Post("/guest", authHandler.CreateGuestSession)
				}
			})
		}

		// OAuth routes (mixed public and protected)
		// Public: GET /{provider}, GET /{provider}/callback
		// Protected: POST /link, DELETE /{provider}, GET /accounts
		if oauthHandler != nil {
			// Create JWT middleware for OAuth protected routes
			oauthJWTMiddleware := middleware.JWTAuth(middleware.AuthConfig{
				JWTService:     middlewareConfig.JWTService,
				TokenBlacklist: middlewareConfig.TokenBlacklist,
				Logger:         middlewareConfig.Logger,
				Optional:       false,
			})
			r.Mount("/auth/oauth", oauthHandler.Routes(oauthJWTMiddleware))
		}

		// Public explore routes (no authentication required)
		// Allows anonymous users to discover public content
		if exploreHandler != nil {
			r.Mount("/explore", exploreHandler.Routes())
		}

		// oEmbed endpoint (Sprint 16 - no authentication required)
		// Enables external sites to embed images using oEmbed protocol
		if oembedHandler != nil {
			r.Mount("/oembed", oembedHandler.Routes())
		}

		// Tag discovery endpoints (Sprint 19 - no authentication required)
		// Public tag search, popular, and trending endpoints
		if tagHandler != nil {
			r.Mount("/tags", tagHandler.Routes())
		}

		// Public group endpoints (Sprint 20 - no authentication required)
		// List, search, and get group operations are public
		if groupHandler != nil {
			r.Mount("/groups", groupHandler.PublicRoutes())
		}

		// Image variant endpoint with optional authentication
		// Supports both authenticated and anonymous access (respects image visibility)
		if imageHandler != nil {
			r.Group(func(r chi.Router) {
				// Optional JWT authentication - extracts user context if token present
				optionalAuthCfg := middleware.AuthConfig{
					JWTService:     middlewareConfig.JWTService,
					TokenBlacklist: middlewareConfig.TokenBlacklist,
					Logger:         middlewareConfig.Logger,
					Optional:       true, // Authentication optional
				}
				r.Use(middleware.JWTAuth(optionalAuthCfg))

				// Variant endpoint returns binary image data
				r.Get("/images/{imageID}/variants/{size}", imageHandler.GetImageVariant)
			})
		}

		// Protected routes (JWT authentication required)
		r.Group(func(r chi.Router) {
			// JWT authentication middleware
			authCfg := middleware.AuthConfig{
				JWTService:     middlewareConfig.JWTService,
				TokenBlacklist: middlewareConfig.TokenBlacklist,
				Logger:         middlewareConfig.Logger,
				Optional:       false, // Authentication required
			}
			r.Use(middleware.JWTAuth(authCfg))

			// Mount user routes
			if userHandler != nil {
				r.Mount("/users", userHandler.Routes())
			}

			// Mount 2FA routes (requires authentication)
			// These are under /auth/2fa but protected unlike public auth routes
			// Rate limiting is applied to verification endpoints to prevent brute-force
			if twoFAHandler != nil {
				r.Route("/auth/2fa", func(r chi.Router) {
					// Apply 2FA rate limiting if configured (5 attempts/min)
					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.TwoFARateLimiter(*middlewareConfig.RateLimiterConfig)).Post("/verify", twoFAHandler.Verify)
						r.With(middleware.TwoFARateLimiter(*middlewareConfig.RateLimiterConfig)).Post("/disable", twoFAHandler.Disable)
					} else {
						r.Post("/verify", twoFAHandler.Verify)
						r.Post("/disable", twoFAHandler.Disable)
					}
					// Non-rate-limited endpoints
					r.Post("/setup", twoFAHandler.Setup)
					r.Get("/status", twoFAHandler.Status)
					r.Post("/backup-codes/regenerate", twoFAHandler.RegenerateBackupCodes)
				})
			}

			// Mount image routes
			// Note: Upload endpoint should have special rate limiting applied at handler level
			if imageHandler != nil {
				r.Mount("/images", imageHandler.Routes())
			}

			// Mount album routes
			if albumHandler != nil {
				r.Mount("/albums", albumHandler.Routes())
			}

			// Mount variant config routes (Sprint 17)
			// Custom variant configurations for image processing
			if variantConfigHandler != nil {
				r.Mount("/variant-configs", variantConfigHandler.Routes())
			}

			// Social interaction routes (likes and comments)
			// These are mounted under images and users paths
			if socialHandler != nil {
				r.Route("/images/{imageID}", func(r chi.Router) {
					// Like/unlike use singular /like path (action endpoints)
					r.Post("/like", socialHandler.LikeImage)
					r.Delete("/like", socialHandler.UnlikeImage)
					// Comments use plural /comments path (collection endpoints)
					r.Post("/comments", socialHandler.AddComment)
					r.Get("/comments", socialHandler.ListImageComments)

					// IPFS endpoints (Sprint 13)
					// POST /ipfs - Pin image to IPFS (owner only)
					// DELETE /ipfs - Unpin image from IPFS (owner only)
					// GET /ipfs - Get IPFS status (authenticated, owner or public images)
					if ipfsHandler != nil {
						r.Post("/ipfs", ipfsHandler.Pin)
						r.Delete("/ipfs", ipfsHandler.Unpin)
						r.Get("/ipfs", ipfsHandler.GetStatus)
					}
				})

				// Comment deletion endpoint (not under images path)
				r.Delete("/comments/{commentID}", socialHandler.DeleteComment)

				// User liked images endpoint
				r.Get("/users/{userID}/likes", socialHandler.GetUserLikedImages)
			}

			// Follow endpoints (authenticated routes)
			// POST /users/{id}/follow - Follow a user
			// DELETE /users/{id}/follow - Unfollow a user
			if followHandler != nil {
				r.Route("/users/{id}", func(r chi.Router) {
					r.Post("/follow", followHandler.FollowUser)
					r.Delete("/follow", followHandler.UnfollowUser)
				})
			}

			// Activity feed endpoint (authenticated route)
			// GET /feed - Get activity feed from followed users
			if activityHandler != nil {
				r.Get("/feed", activityHandler.GetFeed)
			}

			// Notification endpoints (authenticated routes)
			// GET /notifications - Get user's notifications
			// GET /notifications/count - Get unread notification count
			// POST /notifications/read - Mark notifications as read
			if notificationHandler != nil {
				r.Get("/notifications", notificationHandler.GetNotifications)
				r.Get("/notifications/count", notificationHandler.GetUnreadCount)
				r.Post("/notifications/read", notificationHandler.MarkAsRead)
			}

			// Moderation endpoints (Sprint 14)
			// User endpoints (authenticated users can report content)
			// POST /reports - Create abuse report (rate limited: 10/hour per user)
			if moderationHandler != nil {
				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.ReportRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/reports", moderationHandler.CreateReport)
				} else {
					r.Post("/reports", moderationHandler.CreateReport)
				}

				// Admin/moderator endpoints for report management
				// These require moderator or admin role
				r.Group(func(r chi.Router) {
					// Require moderator or admin role
					r.Use(middleware.RequireAnyRole(
						middlewareConfig.Logger,
						metricsCollector,
						"moderator",
						"admin",
					))

					// Report management
					r.Get("/moderation/reports", moderationHandler.ListPendingReports)
					r.Get("/moderation/reports/{reportID}", moderationHandler.GetReport)
					r.Post("/moderation/reports/{reportID}/review", moderationHandler.StartReview)
					r.Post("/moderation/reports/{reportID}/resolve", moderationHandler.ResolveReport)
					r.Post("/moderation/reports/{reportID}/dismiss", moderationHandler.DismissReport)

					// NSFW detection (Sprint 15)
					r.Post("/moderation/nsfw/scan", moderationHandler.ScanImageNSFW)
					r.Get("/moderation/nsfw/scans/{scanID}", moderationHandler.GetNSFWScan)
					r.Get("/moderation/nsfw/flagged", moderationHandler.ListNSFWFlagged)
					r.Get("/images/{imageID}/nsfw-scans", moderationHandler.ListNSFWScansByImage)
				})

				// Ban status endpoint (moderator/admin or self)
				// GET /users/{userID}/ban - Check ban status
				r.Get("/users/{userID}/ban", moderationHandler.GetUserBanStatus)

				// Admin-only ban management
				r.Group(func(r chi.Router) {
					// Require admin role only
					r.Use(middleware.RequireRole(
						middlewareConfig.Logger,
						metricsCollector,
						"admin",
					))

					// Ban operations
					r.Post("/users/{userID}/ban", moderationHandler.BanUser)
					r.Delete("/users/{userID}/ban", moderationHandler.UnbanUser)
					r.Get("/moderation/bans", moderationHandler.ListActiveBans)

					// Featured Picks management (Sprint 19)
					// POST /moderation/featured - Feature an image
					// DELETE /moderation/featured/{imageID} - Unfeature an image
					if featuredHandler != nil {
						r.Mount("/moderation/featured", featuredHandler.Routes())
					}
				})
			}

			// Guest endpoints (Sprint 14)
			// POST /guest/images/{imageID}/claim - Claim guest upload (registered users only)
			if guestHandler != nil {
				r.Mount("/guest", guestHandler.Routes())
			}

			// Protected group endpoints (Sprint 20)
			// Create, update, delete, join, leave, member management
			if groupHandler != nil {
				r.Route("/groups", func(r chi.Router) {
					// Group creation with rate limiting (5 groups/hour per user)
					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.GroupCreationRateLimiter(*middlewareConfig.RateLimiterConfig)).
							Post("/", groupHandler.CreateGroup)
					} else {
						r.Post("/", groupHandler.CreateGroup)
					}

					// Group update and delete (no rate limiting - these are updates to existing groups)
					r.Put("/{groupID}", groupHandler.UpdateGroup)
					r.Delete("/{groupID}", groupHandler.DeleteGroup)

					// Group join with rate limiting (10 joins/hour per user)
					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.GroupJoinRateLimiter(*middlewareConfig.RateLimiterConfig)).
							Post("/{groupID}/join", groupHandler.JoinGroup)
					} else {
						r.Post("/{groupID}/join", groupHandler.JoinGroup)
					}

					// Other membership operations (no rate limiting)
					r.Delete("/{groupID}/leave", groupHandler.LeaveGroup)
					r.Get("/{groupID}/members", groupHandler.ListMembers)

					// Member management routes (admin+ only)
					r.Put("/{groupID}/members/{userID}/role", groupHandler.UpdateMemberRole)
					r.Delete("/{groupID}/members/{userID}", groupHandler.RemoveMember)
					r.Post("/{groupID}/members/{userID}/ban", groupHandler.BanMember)

					// Group albums routes (Sprint 20)
					// Nested under /groups/{groupID}/albums
					if groupAlbumHandler != nil {
						r.Mount("/{groupID}/albums", groupAlbumHandler.Routes())
					}
				})
			}

			// User's groups endpoint
			// GET /me/groups - List current user's group memberships
			if groupHandler != nil {
				r.Get("/me/groups", groupHandler.GetUserGroups)
			}
		})

		// Public follow list endpoints (optional authentication)
		// These allow anonymous access to view follower/following lists
		if followHandler != nil {
			r.Group(func(r chi.Router) {
				// Optional JWT authentication - extracts user context if token present
				optionalAuthCfg := middleware.AuthConfig{
					JWTService:     middlewareConfig.JWTService,
					TokenBlacklist: middlewareConfig.TokenBlacklist,
					Logger:         middlewareConfig.Logger,
					Optional:       true,
				}
				r.Use(middleware.JWTAuth(optionalAuthCfg))

				// GET /users/{id}/followers - List user's followers
				// GET /users/{id}/following - List users this user follows
				r.Get("/users/{id}/followers", followHandler.GetFollowers)
				r.Get("/users/{id}/following", followHandler.GetFollowing)
			})
		}
	})

	return r
}
