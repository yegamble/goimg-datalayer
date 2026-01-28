package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/commands"
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
	jwtpkg "github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// Mocks

type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) NextID() moderation.ReportID {
	args := m.Called()
	return args.Get(0).(moderation.ReportID)
}

func (m *MockReportRepository) FindByID(ctx context.Context, id moderation.ReportID) (*moderation.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Report), args.Error(1)
}

func (m *MockReportRepository) FindPending(ctx context.Context, pagination shared.Pagination) ([]*moderation.Report, int64, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).([]*moderation.Report), args.Get(1).(int64), args.Error(2)
}

func (m *MockReportRepository) FindByImage(ctx context.Context, imageID gallery.ImageID) ([]*moderation.Report, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).([]*moderation.Report), args.Error(1)
}

func (m *MockReportRepository) FindByReporter(ctx context.Context, reporterID identity.UserID, pagination shared.Pagination) ([]*moderation.Report, int64, error) {
	args := m.Called(ctx, reporterID, pagination)
	return args.Get(0).([]*moderation.Report), args.Get(1).(int64), args.Error(2)
}

func (m *MockReportRepository) Save(ctx context.Context, report *moderation.Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

type MockImageRepository struct {
	mock.Mock
}

func (m *MockImageRepository) FindByID(ctx context.Context, id gallery.ImageID) (*gallery.Image, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gallery.Image), args.Error(1)
}

func (m *MockImageRepository) NextID() gallery.ImageID { return gallery.ImageID{} }
func (m *MockImageRepository) FindByOwner(ctx context.Context, ownerID identity.UserID, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindPublic(ctx context.Context, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindByTag(ctx context.Context, tag gallery.Tag, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) FindByStatus(ctx context.Context, status gallery.ImageStatus, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) Search(ctx context.Context, params gallery.SearchParams) ([]*gallery.Image, int64, error) {
	return nil, 0, nil
}
func (m *MockImageRepository) Save(ctx context.Context, image *gallery.Image) error { return nil }
func (m *MockImageRepository) Delete(ctx context.Context, id gallery.ImageID) error { return nil }
func (m *MockImageRepository) ExistsByID(ctx context.Context, id gallery.ImageID) (bool, error) {
	return false, nil
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event shared.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateToken(user *identity.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*jwtpkg.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.Claims), args.Error(1)
}

func (m *MockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(user *identity.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateRefreshToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

type MockTokenBlacklist struct {
	mock.Mock
}

func (m *MockTokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

func TestModerationRateLimitIntegration(t *testing.T) {
	// 1. Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 2. Setup Config
	logger := zerolog.Nop()
	rlConfig := middleware.RateLimiterConfig{
		RedisClient: redisClient,
		Logger:      logger,
		GlobalLimit: 100,
		AuthLimit:   300,
		LoginLimit:  5,
		WindowSize:  time.Hour, // Use Hour to match ReportRateLimiter logic
	}

	// 3. Setup Mocks
	mockReportRepo := new(MockReportRepository)
	mockImageRepo := new(MockImageRepository)
	mockEventPub := new(MockEventPublisher)
	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	metrics := middleware.NewMetricsCollector()

	// 4. Create Handler
	createReportHandler := commands.NewCreateReportHandler(
		mockReportRepo,
		mockImageRepo,
		mockEventPub,
		&logger,
	)

	moderationHandler := handlers.NewModerationHandler(
		createReportHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, // other handlers nil
		logger,
	)

	// Other minimal handlers required by router
	imageHandler := handlers.NewImageHandler(nil, nil, nil, nil, nil, nil, nil, nil, logger)
	albumHandler := handlers.NewAlbumHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	healthHandler := handlers.NewHealthHandler(nil, nil, nil, nil, logger)

	// 5. Create Router
	mwConfig := handlers.MiddlewareConfig{
		JWTService:        mockJWT,
		TokenBlacklist:    mockBlacklist,
		Logger:            logger,
		RateLimiterConfig: &rlConfig,
	}

	router := handlers.NewRouter(
		nil, nil, imageHandler, albumHandler, nil, nil, healthHandler, nil, nil, nil, nil, nil, nil,
		moderationHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, // other handlers
		metrics,
		mwConfig,
		false, // isProd
	)

	// 6. Test
	userID := uuid.New()
	token := "valid-token"
	tokenID := "token-id"

	// Mock JWT validation
	claims := &jwtpkg.Claims{
		UserID:    userID.String(),
		Role:      "user",
		TokenType: jwtpkg.TokenTypeAccess,
		SessionID: uuid.New().String(),
	}
	// Need to mock these for each call or use Mock.On(...).Return(...).Maybe() / Times()
	mockJWT.On("ExtractTokenID", token).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", token).Return(claims, nil)

	// 10 allowed requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/v1/reports", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Expect 400 Bad Request (Validation Failed due to empty body)
		// NOT 429
		assert.Equal(t, http.StatusBadRequest, w.Code, "request %d should be allowed but fail validation", i+1)

		// Verify Rate Limit Headers
		// X-RateLimit-Limit should be 10 for report endpoint
		assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	}

	// 11th request - blocked
	req := httptest.NewRequest("POST", "/api/v1/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "11th request should be blocked")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}
