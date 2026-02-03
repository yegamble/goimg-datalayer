package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	goredis "github.com/redis/go-redis/v9"

	actqueries "github.com/yegamble/goimg-datalayer/internal/application/activity/queries"
	commcommands "github.com/yegamble/goimg-datalayer/internal/application/community/commands"
	commqueries "github.com/yegamble/goimg-datalayer/internal/application/community/queries"
	gallerycommands "github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
	galleryqueries "github.com/yegamble/goimg-datalayer/internal/application/gallery/queries"
	appcommands "github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	appqueries "github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	appservices "github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	modcommands "github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	modqueries "github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	appnotification "github.com/yegamble/goimg-datalayer/internal/application/notification"
	notifcommands "github.com/yegamble/goimg-datalayer/internal/application/notification/commands"
	notifqueries "github.com/yegamble/goimg-datalayer/internal/application/notification/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	domidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/adapters/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/email"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/nsfw"
	appstorage "github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/local"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

const (
	defaultPort             = "8080"
	serverReadHeaderTimeout = 5 * time.Second
	shutdownTimeout         = 5 * time.Second

	// Default configuration values.
	defaultSMTPPort      = 587
	defaultSMTPRateLimit = 100
	defaultSMTPTimeout   = 30 * time.Second
	defaultAccessTTL     = 15 * time.Minute
	defaultRefreshTTL    = 7 * 24 * time.Hour

	// Constants
	strTrue = "true"
)

func main() {
	// 1. Setup Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().Msg("Starting goimg-datalayer API server...")

	// 2. Load Configuration
	dbConfig := postgres.ConfigFromEnv()
	redisConfig := redis.DefaultConfig()
	if pwd := os.Getenv("REDIS_PASSWORD"); pwd != "" {
		redisConfig.Password = pwd
	}
	if useTLS := os.Getenv("REDIS_USE_TLS"); useTLS == strTrue {
		redisConfig.UseTLS = true
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Warn().
			Msg("ENCRYPTION_KEY not set. Generating a random key for this session (2FA secrets will be invalid after restart)")
		_, keyBase64, err := security.GenerateKey()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate encryption key")
		}
		encryptionKey = keyBase64
	}

	// 3. Initialize Infrastructure
	// Email
	emailConfig := email.Config{
		Host:        getEnv("SMTP_HOST", "localhost"),
		Port:        getEnvInt("SMTP_PORT", defaultSMTPPort),
		Username:    getEnv("SMTP_USERNAME", ""),
		Password:    getEnv("SMTP_PASSWORD", ""),
		FromAddress: getEnv("SMTP_FROM_ADDRESS", "noreply@goimg.local"),
		FromName:    getEnv("SMTP_FROM_NAME", "goimg Gallery"),
		UseTLS:      getEnv("SMTP_USE_TLS", strTrue) == strTrue,
		RateLimit:   getEnvInt("SMTP_RATE_LIMIT", defaultSMTPRateLimit),
		Enabled:     getEnv("SMTP_ENABLED", "false") == strTrue,
	}
	// Add timeout if needed, using default from Config struct if zero
	if emailConfig.Timeout == 0 {
		emailConfig.Timeout = defaultSMTPTimeout
	}

	smtpSender, err := email.NewSMTPSender(emailConfig, log.Logger)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize SMTP sender (email notifications disabled)")
		// Create disabled sender if validation fails, to allow app startup
		emailConfig.Enabled = false
		smtpSender, _ = email.NewSMTPSender(emailConfig, log.Logger)
	}

	// Database
	db, err := postgres.NewDB(dbConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database (continuing in degraded mode)")
	} else {
		log.Info().Msg("Connected to database")
		defer func() {
			if err := postgres.Close(db); err != nil {
				log.Error().Err(err).Msg("Failed to close database connection")
			}
		}()
	}

	// Redis
	redisClientWrapper, err := redis.NewClient(redisConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis (continuing in degraded mode)")
	} else {
		log.Info().Msg("Connected to Redis")
		defer func() {
			if err := redisClientWrapper.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close Redis client")
			}
		}()
	}

	var rdbClient *goredis.Client
	if redisClientWrapper != nil {
		rdbClient = redisClientWrapper.UnderlyingClient()
	}

	// Storage
	storageConfig := local.Config{
		BasePath: os.Getenv("STORAGE_BASE_PATH"),
		BaseURL:  os.Getenv("STORAGE_BASE_URL"),
	}
	if storageConfig.BasePath == "" {
		storageConfig.BasePath = "./uploads"
	}
	localStorage, err := local.New(storageConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize local storage")
	}
	storage := &storageAdapter{Store: localStorage}

	// Security - TOTP
	encryptor, err := security.NewSecretEncryptorFromBase64(encryptionKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize secret encryptor")
		return // Exit early as security is compromised without encryptor
	}

	totpConfig := security.DefaultTOTPConfig()
	totpService, err := security.NewTOTPService(totpConfig, encryptor)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize TOTP service")
		return
	}

	// 4. Initialize Repositories
	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	imageRepo := postgres.NewImageRepository(db)
	albumRepo := postgres.NewAlbumRepository(db)
	albumImageRepo := postgres.NewAlbumImageRepository(db)
	commentRepo := postgres.NewCommentRepository(db)
	likeRepo := postgres.NewLikeRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	banRepo := postgres.NewBanRepository(db)
	nsfwRepo := &noOpNSFWScanRepository{} // Using no-op until implementation exists
	groupRepo := postgres.NewGroupRepository(db)
	groupMemberRepo := postgres.NewGroupMembershipRepository(db)
	groupInvitationRepo := postgres.NewGroupInvitationRepository(db)
	groupAlbumRepo := postgres.NewGroupAlbumRepository(db)
	groupAlbumImageRepo := postgres.NewGroupAlbumImageRepository(db)
	groupActivityRepo := postgres.NewGroupActivityRepository(db)
	groupImageRepo := postgres.NewGroupImageRepository(db)
	totpRepo := postgres.NewTOTPRepository(db)
	backupRepo := postgres.NewBackupCodeRepository(db)
	oauthRepo := postgres.NewOAuthAccountRepository(db)
	followRepo := postgres.NewFollowRepository(db)
	activityRepo := postgres.NewActivityRepository(db)
	notifRepo := postgres.NewNotificationRepository(db)
	variantRepo := postgres.NewVariantConfigRepository(db)
	tagRepo := postgres.NewTagRepository(db)
	featuredRepo := postgres.NewFeaturedPickRepository(db)

	// 5. Initialize Services
	ipfsService := &noOpIPFSService{}
	notificationService := appnotification.NewNotificationService(notifRepo, userRepo, smtpSender, log.Logger)

	oauthFactory := security.NewOAuthProviderFactory(encryptor)
	oauthProviderFactory := &oauthProviderFactoryAdapter{
		factory: oauthFactory,
		configs: map[domidentity.OAuthProvider]security.OAuthProviderConfig{
			domidentity.OAuthProviderGoogle: {
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
				Scopes:       []string{"openid", "profile", "email"},
			},
			domidentity.OAuthProviderGitHub: {
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", ""),
				Scopes:       []string{"user:email", "read:user"},
			},
		},
	}

	jwtConfig := jwt.Config{
		Issuer:         "goimg-api",
		AccessTTL:      defaultAccessTTL,
		RefreshTTL:     defaultRefreshTTL,
		PrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
		PublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
	}
	if jwtConfig.PrivateKeyPath == "" {
		jwtConfig.PrivateKeyPath = "certs/private.pem"
	}
	if jwtConfig.PublicKeyPath == "" {
		jwtConfig.PublicKeyPath = "certs/public.pem"
	}

	if _, err := os.Stat(jwtConfig.PrivateKeyPath); os.IsNotExist(err) {
		log.Warn().Msg("JWT keys not found, auth will fail")
	}

	jwtServiceImpl, err := jwt.NewService(jwtConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize JWT service")
	}

	refreshTokenServiceImpl := jwt.NewRefreshTokenService(rdbClient, jwtConfig.RefreshTTL)
	tokenBlacklistImpl := jwt.NewTokenBlacklist(rdbClient)

	idEventPub := &identityEventPublisher{}
	galEventPub := &galleryEventPublisher{}
	modEventPub := &moderationEventPublisher{}
	commEventPub := &communityEventPublisher{}
	jobEnqueuer := &noOpJobEnqueuer{}
	nsfwService := &noOpNSFWService{}

	// Adapters for Application Layer
	var jwtServiceApp appservices.JWTService
	if jwtServiceImpl != nil {
		jwtServiceApp = &identity.JWTServiceAdapter{Service: jwtServiceImpl}
	}

	refreshTokenServiceApp := &identity.RefreshTokenServiceAdapter{Service: refreshTokenServiceImpl}
	sessionStoreApp := &identity.SessionStoreAdapter{Repo: sessionRepo}
	tokenBlacklistApp := &identity.TokenBlacklistAdapter{Service: tokenBlacklistImpl}

	var jwtServiceAppIdentity *identity.JWTServiceAdapterIdentity
	if jwtServiceImpl != nil {
		jwtServiceAppIdentity = &identity.JWTServiceAdapterIdentity{Service: jwtServiceImpl}
	}
	sessionStoreAppIdentity := &identity.SessionStoreAdapterIdentity{Repo: sessionRepo}

	// 6. Initialize Application Handlers

	// Identity Context
	registerUserHandler := appcommands.NewRegisterUserHandler(userRepo, idEventPub, nil, &log.Logger)

	loginHandler := appcommands.NewLoginHandler(
		userRepo,
		jwtServiceApp,
		refreshTokenServiceApp,
		sessionStoreApp,
		nil, // Metrics
		&log.Logger,
	)

	refreshTokenHandler := appcommands.NewRefreshTokenHandler(
		userRepo,
		jwtServiceApp,
		refreshTokenServiceApp,
		sessionStoreApp,
		&log.Logger,
	)

	logoutHandler := appcommands.NewLogoutHandler(
		userRepo,
		jwtServiceApp,
		refreshTokenServiceApp,
		sessionStoreApp,
		tokenBlacklistApp,
		&log.Logger,
	)

	createGuestHandler := appcommands.NewCreateGuestSessionHandler(
		userRepo,
		idEventPub,
		jwtServiceAppIdentity,
		sessionStoreAppIdentity,
		&log.Logger,
	)

	getUserHandler := appqueries.NewGetUserHandler(userRepo)
	updateUserHandler := appcommands.NewUpdateUserHandler(userRepo)
	deleteUserHandler := appcommands.NewDeleteUserHandler(userRepo, sessionStoreAppIdentity)
	getSessionsHandler := appqueries.NewGetUserSessionsHandler(sessionStoreAppIdentity)

	// 2FA Handlers
	setup2FAHandler := appcommands.NewSetup2FAHandler(userRepo, totpRepo, backupRepo, totpService, &log.Logger)
	verify2FAHandler := appcommands.NewVerify2FAHandler(userRepo, totpRepo, totpService, &log.Logger)
	disable2FAHandler := appcommands.NewDisable2FAHandler(userRepo, totpRepo, backupRepo, totpService, &log.Logger)
	regenerateBackupHandler := appcommands.NewRegenerateBackupCodesHandler(userRepo, totpRepo, backupRepo, &log.Logger)
	verifyLoginHandler := appcommands.NewVerify2FALoginHandler(userRepo, totpRepo, jwtServiceApp, totpService, &log.Logger)
	get2FAStatusHandler := appqueries.NewGet2FAStatusHandler(totpRepo, backupRepo, &log.Logger)

	// OAuth Handlers
	authOAuthHandler := appcommands.NewAuthenticateWithOAuthHandler(
		userRepo, oauthRepo, oauthProviderFactory, jwtServiceApp,
		refreshTokenServiceApp, sessionStoreApp, &log.Logger,
	)
	linkOAuthHandler := appcommands.NewLinkOAuthAccountHandler(
		userRepo, oauthRepo, oauthProviderFactory, &log.Logger,
	)
	unlinkOAuthHandler := appcommands.NewUnlinkOAuthAccountHandler(userRepo, oauthRepo, &log.Logger)
	listOAuthAccountsHandler := appqueries.NewListOAuthAccountsHandler(oauthRepo, &log.Logger)

	// Follow Handlers
	followUserHandler := appcommands.NewFollowUserHandler(
		followRepo, userRepo, notificationService, &log.Logger,
	)
	unfollowUserHandler := appcommands.NewUnfollowUserHandler(followRepo, &log.Logger)
	getFollowersHandler := appqueries.NewGetFollowersHandler(followRepo, userRepo)
	getFollowingHandler := appqueries.NewGetFollowingHandler(followRepo, userRepo)

	// Activity Handler
	getActivityFeedHandler := actqueries.NewGetActivityFeedHandler(activityRepo, userRepo)

	// Notification Handlers
	getNotificationsHandler := notifqueries.NewGetNotificationsHandler(notifRepo)
	getUnreadCountHandler := notifqueries.NewGetUnreadCountHandler(notifRepo)
	markNotificationsReadHandler := notifcommands.NewMarkNotificationsReadHandler(notifRepo, log.Logger)

	// Guest Handlers
	claimGuestImageHandler := gallerycommands.NewClaimGuestImageHandler(
		imageRepo, userRepo, galEventPub, &log.Logger,
	)

	// Gallery Context
	uploadImageHandler := gallerycommands.NewUploadImageHandler(
		imageRepo, storage, jobEnqueuer, galEventPub, &log.Logger,
	)
	updateImageHandler := gallerycommands.NewUpdateImageHandler(imageRepo, galEventPub, &log.Logger)
	deleteImageHandler := gallerycommands.NewDeleteImageHandler(
		imageRepo, jobEnqueuer, galEventPub, &log.Logger,
	)

	getImageHandler := galleryqueries.NewGetImageHandler(imageRepo, &log.Logger)
	listImagesHandler := galleryqueries.NewListImagesHandler(imageRepo, &log.Logger)
	searchImagesHandler := galleryqueries.NewSearchImagesHandler(imageRepo)

	// IPFS Handlers
	pinImageHandler := gallerycommands.NewPinImageToIPFSHandler(
		imageRepo, storage, ipfsService, galEventPub, &log.Logger,
	)
	unpinImageHandler := gallerycommands.NewUnpinImageFromIPFSHandler(
		imageRepo, ipfsService, galEventPub, &log.Logger,
	)
	getIPFSStatusHandler := galleryqueries.NewGetImageIPFSStatusHandler(imageRepo, ipfsService, &log.Logger)

	// Variant Config Handlers
	createVarHandler := gallerycommands.NewCreateVariantConfigHandler(
		variantRepo, userRepo, galEventPub, &log.Logger,
	)
	updateVarHandler := gallerycommands.NewUpdateVariantConfigHandler(variantRepo, galEventPub, &log.Logger)
	deleteVarHandler := gallerycommands.NewDeleteVariantConfigHandler(variantRepo, galEventPub, &log.Logger)
	getVarHandler := galleryqueries.NewGetVariantConfigHandler(variantRepo)
	listVarHandler := galleryqueries.NewListVariantConfigsHandler(variantRepo)
	listPresetHandler := galleryqueries.NewListVariantConfigPresetsHandler(variantRepo)

	// Tag Handlers
	searchTagsHandler := galleryqueries.NewSearchTagsHandler(tagRepo, &log.Logger)
	listPopularTagsHandler := galleryqueries.NewListPopularTagsHandler(tagRepo, &log.Logger)
	listTrendingTagsHandler := galleryqueries.NewListTrendingTagsHandler(tagRepo, &log.Logger)

	// Featured Handlers
	featureImageHandler := gallerycommands.NewFeatureImageHandler(imageRepo, featuredRepo, log.Logger)
	unfeatureImageHandler := gallerycommands.NewUnfeatureImageHandler(featuredRepo, log.Logger)
	listFeaturedHandler := galleryqueries.NewListFeaturedImagesHandler(featuredRepo, imageRepo, log.Logger)

	createAlbumHandler := gallerycommands.NewCreateAlbumHandler(albumRepo, userRepo, galEventPub, &log.Logger)
	updateAlbumHandler := gallerycommands.NewUpdateAlbumHandler(albumRepo, galEventPub, &log.Logger)
	deleteAlbumHandler := gallerycommands.NewDeleteAlbumHandler(albumRepo, galEventPub, &log.Logger)
	addImageToAlbumHandler := gallerycommands.NewAddImageToAlbumHandler(
		albumRepo, imageRepo, albumImageRepo, galEventPub, &log.Logger,
	)
	removeImageFromAlbumHandler := gallerycommands.NewRemoveImageFromAlbumHandler(
		albumRepo, albumImageRepo, galEventPub, &log.Logger,
	)

	getAlbumHandler := galleryqueries.NewGetAlbumHandler(albumRepo)
	listAlbumsHandler := galleryqueries.NewListAlbumsHandler(albumRepo)
	listAlbumImagesHandler := galleryqueries.NewListAlbumImagesHandler(albumRepo, albumImageRepo)
	getAlbumBreadcrumbHandler := galleryqueries.NewGetAlbumBreadcrumbHandler(albumRepo)
	getAlbumChildrenHandler := galleryqueries.NewGetAlbumChildrenHandler(albumRepo)

	// Social Context
	likeImageHandler := gallerycommands.NewLikeImageHandler(
		imageRepo, likeRepo, userRepo, galEventPub, &log.Logger,
	)
	unlikeImageHandler := gallerycommands.NewUnlikeImageHandler(
		imageRepo, likeRepo, userRepo, galEventPub, &log.Logger,
	)
	addCommentHandler := gallerycommands.NewAddCommentHandler(
		imageRepo, commentRepo, userRepo, galEventPub, &log.Logger,
	)
	deleteCommentHandler := gallerycommands.NewDeleteCommentHandler(
		imageRepo, commentRepo, userRepo, galEventPub, &log.Logger,
	)

	listImageCommentsHandler := galleryqueries.NewListImageCommentsHandler(commentRepo)
	getUserLikedImagesHandler := galleryqueries.NewGetUserLikedImagesHandler(likeRepo, imageRepo)

	// Moderation Context
	createReportHandler := modcommands.NewCreateReportHandler(reportRepo, imageRepo, modEventPub, &log.Logger)
	startReviewHandler := modcommands.NewStartReviewHandler(reportRepo, modEventPub, &log.Logger)
	resolveReportHandler := modcommands.NewResolveReportHandler(reportRepo, modEventPub, &log.Logger)
	dismissReportHandler := modcommands.NewDismissReportHandler(reportRepo, modEventPub, &log.Logger)
	banUserHandler := modcommands.NewBanUserHandler(banRepo, modEventPub, &log.Logger)
	unbanUserHandler := modcommands.NewUnbanUserHandler(banRepo, modEventPub, &log.Logger)
	scanNSFWHandler := modcommands.NewScanImageNSFWHandler(
		nsfwRepo, imageRepo, nsfwService, modEventPub, &log.Logger,
	)

	getReportHandler := modqueries.NewGetReportHandler(reportRepo)
	listPendingReportsHandler := modqueries.NewListPendingReportsHandler(reportRepo, &log.Logger)
	getUserBanStatusHandler := modqueries.NewGetUserBanStatusHandler(banRepo)
	listActiveBansHandler := modqueries.NewListActiveBansHandler(banRepo, &log.Logger)

	// Group Context
	createGroupHandler := commcommands.NewCreateGroupHandler(
		groupRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	updateGroupHandler := commcommands.NewUpdateGroupHandler(
		groupRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	deleteGroupHandler := commcommands.NewDeleteGroupHandler(groupRepo, commEventPub, &log.Logger)
	joinGroupHandler := commcommands.NewJoinGroupHandler(
		groupRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	leaveGroupHandler := commcommands.NewLeaveGroupHandler(
		groupRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	updateMemberRoleHandler := commcommands.NewUpdateMemberRoleHandler(
		groupMemberRepo, commEventPub, &log.Logger,
	)
	removeMemberHandler := commcommands.NewRemoveMemberHandler(
		groupRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	banMemberHandler := commcommands.NewBanMemberHandler(
		groupRepo, groupMemberRepo, groupActivityRepo, commEventPub, &log.Logger,
	)
	inviteToGroupHandler := commcommands.NewInviteToGroupHandler(
		groupRepo, groupMemberRepo, groupInvitationRepo, commEventPub, &log.Logger,
	)
	acceptInvitationHandler := commcommands.NewAcceptInvitationHandler(
		groupRepo, groupMemberRepo, groupInvitationRepo, commEventPub, &log.Logger,
	)
	declineInvitationHandler := commcommands.NewDeclineInvitationHandler(
		groupInvitationRepo, commEventPub, &log.Logger,
	)

	getGroupHandler := commqueries.NewGetGroupHandler(groupRepo)
	getGroupBySlugHandler := commqueries.NewGetGroupBySlugHandler(groupRepo)
	listPublicGroupsHandler := commqueries.NewListPublicGroupsHandler(groupRepo)
	searchGroupsHandler := commqueries.NewSearchGroupsHandler(groupRepo)
	listGroupMembersHandler := commqueries.NewListGroupMembersHandler(groupMemberRepo)
	listUserGroupsHandler := commqueries.NewListUserGroupsHandler(groupMemberRepo)
	listGroupInvitationsHandler := commqueries.NewListGroupInvitationsHandler(groupInvitationRepo)

	// Group Album Handlers
	createGroupAlbumHandler := commcommands.NewCreateGroupAlbumHandler(
		groupRepo, groupMemberRepo, groupAlbumRepo, commEventPub, &log.Logger,
	)
	updateGroupAlbumHandler := commcommands.NewUpdateGroupAlbumHandler(
		groupAlbumRepo, groupMemberRepo, commEventPub, &log.Logger,
	)
	deleteGroupAlbumHandler := commcommands.NewDeleteGroupAlbumHandler(
		groupAlbumRepo, groupMemberRepo, &log.Logger,
	)
	addImageToGroupAlbumHandler := commcommands.NewAddImageToGroupAlbumHandler(
		groupAlbumRepo, groupAlbumImageRepo, groupImageRepo, groupMemberRepo, &log.Logger,
	)
	removeImageFromGroupAlbumHandler := commcommands.NewRemoveImageFromGroupAlbumHandler(
		groupAlbumRepo, groupAlbumImageRepo, groupMemberRepo, &log.Logger,
	)

	getGroupAlbumHandler := commqueries.NewGetGroupAlbumHandler(groupRepo, groupAlbumRepo, groupMemberRepo)
	listGroupAlbumsHandler := commqueries.NewListGroupAlbumsHandler(groupRepo, groupAlbumRepo, groupMemberRepo)

	// Group Image Handlers
	shareImageHandler := commcommands.NewShareImageToGroupHandler(
		groupRepo, groupMemberRepo, groupImageRepo, groupActivityRepo, commEventPub, &log.Logger,
	)
	approveImageHandler := commcommands.NewApproveGroupImageHandler(
		groupImageRepo, groupMemberRepo, groupActivityRepo, commEventPub, &log.Logger,
	)
	rejectImageHandler := commcommands.NewRejectGroupImageHandler(
		groupImageRepo, groupMemberRepo, groupActivityRepo, commEventPub, &log.Logger,
	)

	listPendingImagesHandler := commqueries.NewListPendingGroupImagesHandler(groupImageRepo, groupMemberRepo)
	listApprovedImagesHandler := commqueries.NewListApprovedGroupImagesHandler(groupImageRepo)

	// 7. Initialize HTTP Handlers
	healthHandler := handlers.NewHealthHandler(
		db,
		redisClientWrapper,
		storage,
		nil, // clamav
		log.Logger,
	)

	authHandler := handlers.NewAuthHandler(
		registerUserHandler,
		loginHandler,
		refreshTokenHandler,
		logoutHandler,
		createGuestHandler,
		log.Logger,
	)

	userHandler := handlers.NewUserHandler(
		getUserHandler,
		updateUserHandler,
		deleteUserHandler,
		getSessionsHandler,
		log.Logger,
	)

	twoFAHandler := handlers.NewTwoFAHandler(
		setup2FAHandler,
		verify2FAHandler,
		disable2FAHandler,
		regenerateBackupHandler,
		verifyLoginHandler,
		get2FAStatusHandler,
		log.Logger,
	)

	oauthHandler := handlers.NewOAuthHandler(
		authOAuthHandler,
		linkOAuthHandler,
		unlinkOAuthHandler,
		listOAuthAccountsHandler,
		oauthProviderFactory,
		rdbClient,
		log.Logger,
	)

	followHandler := handlers.NewFollowHandler(
		followUserHandler,
		unfollowUserHandler,
		getFollowersHandler,
		getFollowingHandler,
		log.Logger,
	)

	activityHandler := handlers.NewActivityHandler(getActivityFeedHandler, log.Logger)

	notificationHandler := handlers.NewNotificationHandler(
		getNotificationsHandler,
		getUnreadCountHandler,
		markNotificationsReadHandler,
		log.Logger,
	)

	guestHandler := handlers.NewGuestHandler(claimGuestImageHandler, log.Logger)

	ipfsHandler := handlers.NewIPFSHandler(
		pinImageHandler,
		unpinImageHandler,
		getIPFSStatusHandler,
		log.Logger,
	)

	variantConfigHandler := handlers.NewVariantConfigHandler(
		createVarHandler,
		updateVarHandler,
		deleteVarHandler,
		getVarHandler,
		listVarHandler,
		listPresetHandler,
		log.Logger,
	)

	tagHandler := handlers.NewTagHandler(
		listPopularTagsHandler,
		listTrendingTagsHandler,
		searchTagsHandler,
		listImagesHandler,
		log.Logger,
	)

	featuredHandler := handlers.NewFeaturedHandler(
		featureImageHandler,
		unfeatureImageHandler,
		log.Logger,
	)

	exploreHandler := handlers.NewExploreHandler(
		listImagesHandler,
		listFeaturedHandler,
		log.Logger,
	)

	oembedHandler := handlers.NewOEmbedHandler(
		getImageHandler,
		os.Getenv("BASE_URL"),
		log.Logger,
	)

	previewHandler := handlers.NewPreviewHandler(
		getImageHandler,
		os.Getenv("BASE_URL"),
		log.Logger,
	)

	imageHandler := handlers.NewImageHandler(
		uploadImageHandler,
		updateImageHandler,
		deleteImageHandler,
		nil, // generateCustomVariant
		getImageHandler,
		listImagesHandler,
		searchImagesHandler,
		storage,
		log.Logger,
	)

	albumHandler := handlers.NewAlbumHandler(
		createAlbumHandler,
		updateAlbumHandler,
		deleteAlbumHandler,
		addImageToAlbumHandler,
		removeImageFromAlbumHandler,
		getAlbumHandler,
		listAlbumsHandler,
		listAlbumImagesHandler,
		getAlbumBreadcrumbHandler,
		getAlbumChildrenHandler,
		log.Logger,
	)

	socialHandler := handlers.NewSocialHandler(
		likeImageHandler,
		unlikeImageHandler,
		addCommentHandler,
		deleteCommentHandler,
		listImageCommentsHandler,
		getUserLikedImagesHandler,
		log.Logger,
	)

	moderationHandler := handlers.NewModerationHandler(
		createReportHandler,
		startReviewHandler,
		resolveReportHandler,
		dismissReportHandler,
		banUserHandler,
		unbanUserHandler,
		scanNSFWHandler,
		getReportHandler,
		listPendingReportsHandler,
		getUserBanStatusHandler,
		listActiveBansHandler,
		nil, // getNSFWScanHandler
		nil, // listNSFWFlaggedHandler
		nil, // listNSFWScansByImageHandler
		log.Logger,
	)

	groupHandler := handlers.NewGroupHandler(
		createGroupHandler,
		updateGroupHandler,
		deleteGroupHandler,
		joinGroupHandler,
		leaveGroupHandler,
		updateMemberRoleHandler,
		removeMemberHandler,
		banMemberHandler,
		inviteToGroupHandler,
		acceptInvitationHandler,
		declineInvitationHandler,
		getGroupHandler,
		getGroupBySlugHandler,
		listPublicGroupsHandler,
		searchGroupsHandler,
		listGroupMembersHandler,
		listUserGroupsHandler,
		listGroupInvitationsHandler,
		log.Logger,
	)

	groupAlbumHandler := handlers.NewGroupAlbumHandler(
		createGroupAlbumHandler,
		updateGroupAlbumHandler,
		deleteGroupAlbumHandler,
		getGroupAlbumHandler,
		listGroupAlbumsHandler,
		addImageToGroupAlbumHandler,
		removeImageFromGroupAlbumHandler,
		log.Logger,
	)

	groupImageHandler := handlers.NewGroupImageHandler(
		shareImageHandler,
		approveImageHandler,
		rejectImageHandler,
		listPendingImagesHandler,
		listApprovedImagesHandler,
		log.Logger,
	)

	middlewareConfig := handlers.MiddlewareConfig{
		Logger:         log.Logger,
		JWTService:     jwtServiceImpl,
		TokenBlacklist: tokenBlacklistImpl,
	}

	router := handlers.NewRouter(
		authHandler,
		userHandler,
		imageHandler,
		albumHandler,
		socialHandler,
		exploreHandler,
		healthHandler,
		twoFAHandler,
		oauthHandler,
		followHandler,
		activityHandler,
		notificationHandler,
		ipfsHandler,
		moderationHandler,
		guestHandler,
		oembedHandler,
		previewHandler,
		variantConfigHandler,
		tagHandler,
		featuredHandler,
		groupHandler,
		groupAlbumHandler,
		groupImageHandler,
		middleware.NewMetricsCollector(),
		middlewareConfig,
		false,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server startup failed")
		}
	}()

	log.Info().Msgf("Server listening on port %s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}

// --- Adapters ---

type identityEventPublisher struct{}

func (p *identityEventPublisher) Publish(_ context.Context, _ interface{}) error {
	return nil
}

type galleryEventPublisher struct{}

func (p *galleryEventPublisher) Publish(_ context.Context, _ shared.DomainEvent) error {
	return nil
}

type moderationEventPublisher struct{}

func (p *moderationEventPublisher) Publish(_ context.Context, _ shared.DomainEvent) error {
	return nil
}

type communityEventPublisher struct{}

func (p *communityEventPublisher) Publish(_ context.Context, _ shared.DomainEvent) error {
	return nil
}

type noOpJobEnqueuer struct{}

func (j *noOpJobEnqueuer) EnqueueImageCleanup(_ context.Context, _ string, _ string, _ []string) error {
	return nil
}
func (j *noOpJobEnqueuer) EnqueueImageProcessing(_ context.Context, _ string) error {
	return nil
}
func (j *noOpJobEnqueuer) EnqueueImageScan(_ context.Context, _ string) error {
	return nil
}

type noOpNSFWService struct{}

func (s *noOpNSFWService) Scan(_ context.Context, _ string) (*nsfw.ScanResult, error) {
	// Return a safe mock result
	return &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.0,
		Provider: moderation.ProviderSightEngine, // Assuming standard provider or mock
	}, nil
}
func (s *noOpNSFWService) ScanBytes(_ context.Context, _ []byte, _ string) (*nsfw.ScanResult, error) {
	return &nsfw.ScanResult{
		Category: moderation.CategorySafe,
		Score:    0.0,
		Provider: moderation.ProviderSightEngine,
	}, nil
}
func (s *noOpNSFWService) IsAvailable(_ context.Context) bool {
	return true
}
func (s *noOpNSFWService) Provider() moderation.NSFWProvider {
	return moderation.ProviderSightEngine // Return a valid provider enum
}

// Stub for missing NSFW repository.
// Implementing moderation.NSFWScanRepository interface.
type noOpNSFWScanRepository struct{}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) NextID() moderation.NSFWScanID { return moderation.NSFWScanID{} }

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindByID(_ context.Context, _ moderation.NSFWScanID) (*moderation.NSFWScan, error) {
	return nil, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindByImageID(_ context.Context, _ gallery.ImageID) (*moderation.NSFWScan, error) {
	return nil, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindByImageIDAll(
	_ context.Context, _ gallery.ImageID,
) ([]*moderation.NSFWScan, error) {
	return nil, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindPending(
	_ context.Context, _ shared.Pagination,
) ([]*moderation.NSFWScan, int64, error) {
	return nil, 0, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindByStatus(
	_ context.Context, _ moderation.NSFWScanStatus, _ shared.Pagination,
) ([]*moderation.NSFWScan, int64, error) {
	return nil, 0, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) FindNSFWImages(
	_ context.Context, _ shared.Pagination,
) ([]*moderation.NSFWScan, int64, error) {
	return nil, 0, nil
}

//nolint:nilnil // Stub implementation
func (r *noOpNSFWScanRepository) HasActiveScan(_ context.Context, _ gallery.ImageID) (bool, error) {
	return false, nil
}

func (r *noOpNSFWScanRepository) Save(_ context.Context, _ *moderation.NSFWScan) error {
	return nil
}

type noOpIPFSService struct{}

func (s *noOpIPFSService) Add(_ context.Context, _ []byte) (string, error) {
	return "QmFakeCID", nil
}
func (s *noOpIPFSService) Pin(_ context.Context, _ string) error              { return nil }
func (s *noOpIPFSService) Unpin(_ context.Context, _ string) error            { return nil }
func (s *noOpIPFSService) IsPinned(_ context.Context, _ string) (bool, error) { return false, nil }
func (s *noOpIPFSService) GatewayURL(cid string) string                       { return "https://ipfs.io/ipfs/" + cid }

type storageAdapter struct {
	Store *local.Storage
}

func (s *storageAdapter) Put(
	ctx context.Context, key string, data io.Reader, size int64, opts appstorage.PutOptions,
) error {
	localOpts := local.PutOptions{
		ContentType:  opts.ContentType,
		CacheControl: opts.CacheControl,
		Metadata:     opts.Metadata,
	}
	if err := s.Store.Put(ctx, key, data, size, localOpts); err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (s *storageAdapter) PutBytes(ctx context.Context, key string, data []byte, opts appstorage.PutOptions) error {
	localOpts := local.PutOptions{
		ContentType:  opts.ContentType,
		CacheControl: opts.CacheControl,
		Metadata:     opts.Metadata,
	}
	if err := s.Store.PutBytes(ctx, key, data, localOpts); err != nil {
		return fmt.Errorf("failed to put bytes: %w", err)
	}
	return nil
}

func (s *storageAdapter) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := s.Store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return reader, nil
}

func (s *storageAdapter) GetBytes(ctx context.Context, key string) ([]byte, error) {
	data, err := s.Store.GetBytes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bytes: %w", err)
	}
	return data, nil
}

func (s *storageAdapter) Delete(ctx context.Context, key string) error {
	if err := s.Store.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (s *storageAdapter) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := s.Store.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

func (s *storageAdapter) URL(key string) string {
	return s.Store.URL(key)
}

func (s *storageAdapter) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	url, err := s.Store.PresignedURL(ctx, key, duration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url, nil
}

func (s *storageAdapter) Stat(ctx context.Context, key string) (*appstorage.ObjectInfo, error) {
	info, err := s.Store.Stat(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return &appstorage.ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
		ETag:         info.ETag,
	}, nil
}

func (s *storageAdapter) Provider() string {
	return s.Store.Provider()
}

// Helper functions for env vars.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// oauthProviderFactoryAdapter adapts security.OAuthProviderFactory to appcommands.OAuthProviderFactory.
type oauthProviderFactoryAdapter struct {
	factory *security.OAuthProviderFactory
	configs map[domidentity.OAuthProvider]security.OAuthProviderConfig
}

// CreateProvider creates a new OAuth provider.
//
//nolint:ireturn // Adapter requires returning interface
func (a *oauthProviderFactoryAdapter) CreateProvider(
	pType domidentity.OAuthProvider,
) (appcommands.OAuthProvider, error) {
	cfg, ok := a.configs[pType]
	if !ok {
		// Default config or error
		cfg = security.OAuthProviderConfig{}
	}
	provider, err := a.factory.CreateProvider(pType, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}
	return provider, nil
}

// Encryptor returns the token encryptor.
//
//nolint:ireturn // Adapter requires returning interface
func (a *oauthProviderFactoryAdapter) Encryptor() appcommands.TokenEncryptor {
	return a.factory.Encryptor()
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
