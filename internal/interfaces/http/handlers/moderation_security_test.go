package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	jwtpkg "github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
)

// Define local mocks to avoid conflicts with other test files in the same package
type SecTestMockBanRepository struct {
	mock.Mock
}

func (m *SecTestMockBanRepository) NextID() moderation.BanID {
	args := m.Called()
	return args.Get(0).(moderation.BanID)
}

func (m *SecTestMockBanRepository) FindByID(ctx context.Context, id moderation.BanID) (*moderation.Ban, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Ban), args.Error(1)
}

func (m *SecTestMockBanRepository) FindByUserID(ctx context.Context, userID identity.UserID) (*moderation.Ban, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*moderation.Ban), args.Error(1)
}

func (m *SecTestMockBanRepository) FindActiveBans(ctx context.Context) ([]*moderation.Ban, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Ban), args.Error(1)
}

func (m *SecTestMockBanRepository) FindExpiredBans(ctx context.Context) ([]*moderation.Ban, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*moderation.Ban), args.Error(1)
}

func (m *SecTestMockBanRepository) IsUserBanned(ctx context.Context, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *SecTestMockBanRepository) Save(ctx context.Context, ban *moderation.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

type SecTestMockJWTService struct {
	mock.Mock
}

func (m *SecTestMockJWTService) GenerateToken(user *identity.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *SecTestMockJWTService) ValidateToken(tokenString string) (*jwtpkg.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.Claims), args.Error(1)
}

func (m *SecTestMockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

func (m *SecTestMockJWTService) GenerateRefreshToken(user *identity.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *SecTestMockJWTService) ValidateRefreshToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

type SecTestMockTokenBlacklist struct {
	mock.Mock
}

func (m *SecTestMockTokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

func TestGetUserBanStatus_Authorization(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	mockBanRepo := new(SecTestMockBanRepository)
	mockJWT := new(SecTestMockJWTService)
	mockBlacklist := new(SecTestMockTokenBlacklist)
	metrics := getTestMetricsCollector() // Provided by test_utils_test.go

	// Create GetUserBanStatusHandler
	getBanStatusHandler := queries.NewGetUserBanStatusHandler(mockBanRepo)

	// Create ModerationHandler with ONLY getBanStatusHandler populated (others nil)
	moderationHandler := handlers.NewModerationHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		getBanStatusHandler, // Inject the handler we want to test
		nil, nil, nil, nil,
		logger,
	)

	// Setup minimal router requirements
	mwConfig := handlers.MiddlewareConfig{
		JWTService:        mockJWT,
		TokenBlacklist:    mockBlacklist,
		Logger:            logger,
		RateLimiterConfig: nil, // No rate limiting for this test
	}

	router := handlers.NewRouter(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		moderationHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		metrics,
		mwConfig,
		false,
	)

	// Test Case: Regular user accessing another user's ban status
	requestingUserID := uuid.New()
	targetUserID := uuid.New()
	token := "valid-token"
	tokenID := "token-id"

	// Mock JWT validation
	claims := &jwtpkg.Claims{
		UserID:    requestingUserID.String(),
		Role:      "user", // Regular user
		TokenType: jwtpkg.TokenTypeAccess,
		SessionID: uuid.New().String(),
	}

	mockJWT.On("ExtractTokenID", token).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", token).Return(claims, nil)

	// Mock Ban Repository - Currently called because auth is missing
	mockBanRepo.On("IsUserBanned", mock.Anything, mock.Anything).Return(false, nil).Maybe()

	// Request
	req := httptest.NewRequest("GET", "/api/v1/users/"+targetUserID.String()+"/ban", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	// We expect 403 Forbidden, but currently it returns 200 OK (because authorized but not banned)
	assert.Equal(t, http.StatusForbidden, w.Code, "Regular user should not be able to view another user's ban status")
}
