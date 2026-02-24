package handlers

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type MiddlewareConfig struct {
	JWTService middleware.JWTServiceInterface

	TokenBlacklist middleware.TokenBlacklistInterface

	Logger zerolog.Logger

	RateLimiterConfig *middleware.RateLimiterConfig
}

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
	groupImageHandler *GroupImageHandler,
	metricsCollector *middleware.MetricsCollector,
	middlewareConfig MiddlewareConfig,
	isProd bool,
) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.MetricsMiddleware(metricsCollector))
	r.Use(middleware.Logger(middlewareConfig.Logger))
	r.Use(middleware.Recovery(middlewareConfig.Logger))

	r.Use(middleware.BodyLimit(middleware.BodyLimitConfig{LimitBytes: 10 * 1024 * 1024}))

	securityCfg := middleware.DefaultSecurityHeadersConfig(isProd)
	r.Use(middleware.SecurityHeaders(securityCfg))

	var corsCfg middleware.CORSConfig
	if isProd {
		corsCfg = middleware.DefaultCORSConfig()
	} else {
		corsCfg = middleware.DevelopmentCORSConfig()
	}
	r.Use(middleware.CORS(corsCfg))

	r.Use(chimiddleware.Timeout(contextTimeout * time.Second))

	if healthHandler != nil {
		r.Get("/health", healthHandler.Liveness)

		r.Get("/health/ready", healthHandler.Readiness)
	}

	r.Handle("/metrics", promhttp.Handler())

	if previewHandler != nil {
		r.Mount("/images", previewHandler.Routes())
	}

	r.Route("/api/v1", func(r chi.Router) {
		if healthHandler != nil {
			r.Get("/health", healthHandler.Liveness)
			r.Get("/health/live", healthHandler.Liveness)
			r.Get("/health/ready", healthHandler.Readiness)
		}

		r.Handle("/metrics", promhttp.Handler())

		if authHandler != nil {
			r.Route("/auth", func(r chi.Router) {
				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.LoginRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/login", authHandler.Login)
					r.With(middleware.LoginRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/register", authHandler.Register)
				} else {
					r.Post("/login", authHandler.Login)
					r.Post("/register", authHandler.Register)
				}

				r.Post("/refresh", authHandler.Refresh)

				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.LoginRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/forgot-password", authHandler.ForgotPassword)
					r.With(middleware.LoginRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/reset-password", authHandler.ResetPassword)
					r.With(middleware.LoginRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/verify-email", authHandler.VerifyEmail)
				} else {
					r.Post("/forgot-password", authHandler.ForgotPassword)
					r.Post("/reset-password", authHandler.ResetPassword)
					r.Post("/verify-email", authHandler.VerifyEmail)
				}

				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.GuestSessionRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/guest", authHandler.CreateGuestSession)
				} else {
					r.Post("/guest", authHandler.CreateGuestSession)
				}
			})
		}

		if oauthHandler != nil {
			oauthJWTMiddleware := middleware.JWTAuth(middleware.AuthConfig{
				JWTService:     middlewareConfig.JWTService,
				TokenBlacklist: middlewareConfig.TokenBlacklist,
				Logger:         middlewareConfig.Logger,
				Optional:       false,
			})
			r.Mount("/auth/oauth", oauthHandler.Routes(oauthJWTMiddleware))
		}

		if exploreHandler != nil {
			r.Mount("/explore", exploreHandler.Routes())
		}

		if oembedHandler != nil {
			r.Mount("/oembed", oembedHandler.Routes())
		}

		if tagHandler != nil {
			r.Mount("/tags", tagHandler.Routes())
		}

		if imageHandler != nil {
			r.Group(func(r chi.Router) {
				optionalAuthCfg := middleware.AuthConfig{
					JWTService:     middlewareConfig.JWTService,
					TokenBlacklist: middlewareConfig.TokenBlacklist,
					Logger:         middlewareConfig.Logger,
					Optional:       true,
				}
				r.Use(middleware.JWTAuth(optionalAuthCfg))

				r.Get("/images/{imageID}/variants/{size}", imageHandler.GetImageVariant)
				r.Get("/images/{imageID}/qr", imageHandler.GetImageQRCode)
			})
		}

		if groupHandler != nil {
			r.Route("/groups", func(r chi.Router) {
				r.Get("/", groupHandler.ListPublicGroups)
				r.Get("/search", groupHandler.SearchGroups)
				r.Get("/{groupID}", groupHandler.GetGroup)
				r.Get("/by-slug/{slug}", groupHandler.GetGroupBySlug)

				r.Group(func(r chi.Router) {
					authCfg := middleware.AuthConfig{
						JWTService:     middlewareConfig.JWTService,
						TokenBlacklist: middlewareConfig.TokenBlacklist,
						Logger:         middlewareConfig.Logger,
						Optional:       false,
					}
					r.Use(middleware.JWTAuth(authCfg))

					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.GroupCreationRateLimiter(*middlewareConfig.RateLimiterConfig)).
							Post("/", groupHandler.CreateGroup)
					} else {
						r.Post("/", groupHandler.CreateGroup)
					}

					r.Put("/{groupID}", groupHandler.UpdateGroup)
					r.Delete("/{groupID}", groupHandler.DeleteGroup)

					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.GroupJoinRateLimiter(*middlewareConfig.RateLimiterConfig)).
							Post("/{groupID}/join", groupHandler.JoinGroup)
					} else {
						r.Post("/{groupID}/join", groupHandler.JoinGroup)
					}

					r.Delete("/{groupID}/leave", groupHandler.LeaveGroup)
					r.Get("/{groupID}/members", groupHandler.ListMembers)

					r.Put("/{groupID}/members/{userID}/role", groupHandler.UpdateMemberRole)
					r.Delete("/{groupID}/members/{userID}", groupHandler.RemoveMember)
					r.Post("/{groupID}/members/{userID}/ban", groupHandler.BanMember)

					r.Post("/{groupID}/invitations", groupHandler.CreateInvitation)
					r.Get("/{groupID}/invitations", groupHandler.ListInvitations)
					r.Post("/invitations/{token}/accept", groupHandler.AcceptInvitation)
					r.Post("/invitations/{token}/decline", groupHandler.DeclineInvitation)

					if groupAlbumHandler != nil {
						r.Mount("/{groupID}/albums", groupAlbumHandler.Routes())
					}

					if groupImageHandler != nil {
						r.Mount("/{groupID}/images", groupImageHandler.Routes())
					}
				})
			})
		}

		r.Group(func(r chi.Router) {
			authCfg := middleware.AuthConfig{
				JWTService:     middlewareConfig.JWTService,
				TokenBlacklist: middlewareConfig.TokenBlacklist,
				Logger:         middlewareConfig.Logger,
				Optional:       false,
			}
			r.Use(middleware.JWTAuth(authCfg))

			if authHandler != nil {
				r.Post("/auth/logout", authHandler.Logout)
				r.Post("/auth/resend-verification", authHandler.ResendVerification)
			}

			if userHandler != nil || socialHandler != nil || followHandler != nil {
				r.Route("/users", func(r chi.Router) {
					if userHandler != nil {
						r.Get("/me", userHandler.GetCurrentUser)
						r.Get("/{id}", userHandler.GetUser)
						r.Put("/{id}", userHandler.UpdateUser)
						r.Delete("/{id}", userHandler.DeleteUser)
						r.Get("/{id}/sessions", userHandler.GetUserSessions)
					}

					if socialHandler != nil {
						r.Get("/{userID}/likes", socialHandler.GetUserLikedImages)
					}

					if followHandler != nil {
						r.Post("/{id}/follow", followHandler.FollowUser)
						r.Delete("/{id}/follow", followHandler.UnfollowUser)
					}
				})
			}

			if twoFAHandler != nil {
				r.Route("/auth/2fa", func(r chi.Router) {
					if middlewareConfig.RateLimiterConfig != nil {
						r.With(middleware.TwoFARateLimiter(*middlewareConfig.RateLimiterConfig)).Post("/verify", twoFAHandler.Verify)
						r.With(middleware.TwoFARateLimiter(*middlewareConfig.RateLimiterConfig)).Post("/disable", twoFAHandler.Disable)
					} else {
						r.Post("/verify", twoFAHandler.Verify)
						r.Post("/disable", twoFAHandler.Disable)
					}
					r.Post("/setup", twoFAHandler.Setup)
					r.Get("/status", twoFAHandler.Status)
					r.Post("/backup-codes/regenerate", twoFAHandler.RegenerateBackupCodes)
				})
			}

			// Note: Upload endpoint should have special rate limiting applied at handler level
			if imageHandler != nil {
				r.Mount("/images", imageHandler.Routes())
			}

			if albumHandler != nil {
				r.Mount("/albums", albumHandler.Routes())
			}

			if variantConfigHandler != nil {
				r.Mount("/variant-configs", variantConfigHandler.Routes())
			}

			if socialHandler != nil {
				r.Route("/images/{imageID}", func(r chi.Router) {
					r.Post("/like", socialHandler.LikeImage)
					r.Delete("/like", socialHandler.UnlikeImage)
					r.Post("/comments", socialHandler.AddComment)
					r.Get("/comments", socialHandler.ListImageComments)

					if ipfsHandler != nil {
						r.Post("/ipfs", ipfsHandler.Pin)
						r.Delete("/ipfs", ipfsHandler.Unpin)
						r.Get("/ipfs", ipfsHandler.GetStatus)
					}
				})

				r.Delete("/comments/{commentID}", socialHandler.DeleteComment)
			}

			if activityHandler != nil {
				r.Get("/feed", activityHandler.GetFeed)
			}

			if notificationHandler != nil {
				r.Get("/notifications", notificationHandler.GetNotifications)
				r.Get("/notifications/count", notificationHandler.GetUnreadCount)
				r.Post("/notifications/read", notificationHandler.MarkAsRead)
			}

			if moderationHandler != nil {
				if middlewareConfig.RateLimiterConfig != nil {
					r.With(middleware.ReportRateLimiter(*middlewareConfig.RateLimiterConfig)).
						Post("/reports", moderationHandler.CreateReport)
				} else {
					r.Post("/reports", moderationHandler.CreateReport)
				}

				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireAnyRole(
						middlewareConfig.Logger,
						metricsCollector,
						"moderator",
						"admin",
					))

					r.Get("/moderation/reports", moderationHandler.ListPendingReports)
					r.Get("/moderation/reports/{reportID}", moderationHandler.GetReport)
					r.Post("/moderation/reports/{reportID}/review", moderationHandler.StartReview)
					r.Post("/moderation/reports/{reportID}/resolve", moderationHandler.ResolveReport)
					r.Post("/moderation/reports/{reportID}/dismiss", moderationHandler.DismissReport)

					r.Post("/moderation/nsfw/scan", moderationHandler.ScanImageNSFW)
					r.Get("/moderation/nsfw/scans/{scanID}", moderationHandler.GetNSFWScan)
					r.Get("/moderation/nsfw/flagged", moderationHandler.ListNSFWFlagged)
					r.Get("/images/{imageID}/nsfw-scans", moderationHandler.ListNSFWScansByImage)
				})

				r.Get("/users/{userID}/ban", moderationHandler.GetUserBanStatus)

				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole(
						middlewareConfig.Logger,
						metricsCollector,
						"admin",
					))

					r.Post("/users/{userID}/ban", moderationHandler.BanUser)
					r.Delete("/users/{userID}/ban", moderationHandler.UnbanUser)
					r.Get("/moderation/bans", moderationHandler.ListActiveBans)

					if featuredHandler != nil {
						r.Mount("/moderation/featured", featuredHandler.Routes())
					}
				})
			}

			if guestHandler != nil {
				r.Mount("/guest", guestHandler.Routes())
			}

			if groupHandler != nil {
				r.Get("/me/groups", groupHandler.GetUserGroups)
			}
		})

		if followHandler != nil {
			r.Group(func(r chi.Router) {
				optionalAuthCfg := middleware.AuthConfig{
					JWTService:     middlewareConfig.JWTService,
					TokenBlacklist: middlewareConfig.TokenBlacklist,
					Logger:         middlewareConfig.Logger,
					Optional:       true,
				}
				r.Use(middleware.JWTAuth(optionalAuthCfg))

				r.Get("/users/{id}/followers", followHandler.GetFollowers)
				r.Get("/users/{id}/following", followHandler.GetFollowing)
			})
		}
	})

	return r
}
