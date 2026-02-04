package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/commands"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockOAuthProvider is a mock implementation of commands.OAuthProvider.
type MockOAuthProvider struct {
	mock.Mock
}

func (m *MockOAuthProvider) GetAuthorizationURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockOAuthProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockOAuthProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*security.OAuthUserInfo, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.OAuthUserInfo), args.Error(1)
}

// MockTokenEncryptor is a mock implementation of commands.TokenEncryptor.
type MockTokenEncryptor struct {
	mock.Mock
}

func (m *MockTokenEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	args := m.Called(plaintext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTokenEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	args := m.Called(ciphertext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// MockOAuthProviderFactory is a mock implementation of commands.OAuthProviderFactory.
type MockOAuthProviderFactory struct {
	mock.Mock
}

func (m *MockOAuthProviderFactory) CreateProvider(provider identity.OAuthProvider) (commands.OAuthProvider, error) {
	args := m.Called(provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(commands.OAuthProvider), args.Error(1)
}

func (m *MockOAuthProviderFactory) Encryptor() commands.TokenEncryptor {
	args := m.Called()
	return args.Get(0).(commands.TokenEncryptor)
}

func TestOAuthHandler_InitiateOAuth(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})

		mockFactory := new(MockOAuthProviderFactory)
		mockProvider := new(MockOAuthProvider)
		logger := zerolog.Nop()

		// Handler under test
		oauthHandler := handlers.NewOAuthHandler(nil, nil, nil, nil, mockFactory, redisClient, logger)

		// Create router to handle path params
		r := chi.NewRouter()
		r.Get("/auth/oauth/{provider}", oauthHandler.InitiateOAuth)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google", nil)
		rr := httptest.NewRecorder()

		mockFactory.On("CreateProvider", identity.OAuthProviderGoogle).Return(mockProvider, nil)
		mockProvider.On("GetAuthorizationURL", mock.AnythingOfType("string")).Return("https://accounts.google.com/o/oauth2/auth?state=xyz")

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, "https://accounts.google.com/o/oauth2/auth?state=xyz", rr.Header().Get("Location"))

		// Verify state was stored in Redis
		keys, err := redisClient.Keys(context.Background(), "goimg:oauth:state:*").Result()
		assert.NoError(t, err)
		assert.Len(t, keys, 1)

		val, err := redisClient.Get(context.Background(), keys[0]).Result()
		assert.NoError(t, err)
		assert.Equal(t, "google", val)
	})

	t.Run("InvalidProvider", func(t *testing.T) {
		logger := zerolog.Nop()
		oauthHandler := handlers.NewOAuthHandler(nil, nil, nil, nil, nil, nil, logger)

		r := chi.NewRouter()
		r.Get("/auth/oauth/{provider}", oauthHandler.InitiateOAuth)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/invalid", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestOAuthHandler_HandleCallback(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})

		// Setup Redis state
		state := "valid_state"
		err = redisClient.Set(context.Background(), fmt.Sprintf("goimg:oauth:state:%s", state), "google", 10*time.Minute).Err()
		require.NoError(t, err)

		mockFactory := new(MockOAuthProviderFactory)
		mockProvider := new(MockOAuthProvider)
		mockRepo := new(testhelpers.MockUserRepository)
		mockOAuthRepo := new(testhelpers.MockOAuthAccountRepository)
		mockJWT := new(testhelpers.MockJWTService)
		mockRefresh := new(testhelpers.MockRefreshTokenService)
		mockSession := new(testhelpers.MockSessionStore)
		mockEncryptor := new(MockTokenEncryptor)
		logger := zerolog.Nop()

		// Mock dependencies for AuthenticateWithOAuthHandler
		authHandler := commands.NewAuthenticateWithOAuthHandler(
			mockRepo,
			mockOAuthRepo,
			mockFactory,
			mockJWT,
			mockRefresh,
			mockSession,
			&logger,
		)

		handler := handlers.NewOAuthHandler(authHandler, nil, nil, nil, mockFactory, redisClient, logger)

		r := chi.NewRouter()
		r.Get("/auth/oauth/{provider}/callback", handler.HandleCallback)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/auth/oauth/google/callback?code=auth_code&state=%s", state), nil)
		rr := httptest.NewRecorder()

		// Expectations
		mockFactory.On("CreateProvider", identity.OAuthProviderGoogle).Return(mockProvider, nil)
		mockFactory.On("Encryptor").Return(mockEncryptor)
		mockEncryptor.On("Encrypt", mock.Anything).Return([]byte("encrypted"), nil)
		mockProvider.On("ExchangeCode", mock.Anything, "auth_code").Return(&oauth2.Token{AccessToken: "token"}, nil)
		mockProvider.On("GetUserInfo", mock.Anything, mock.Anything).Return(&security.OAuthUserInfo{
			ProviderUserID: "provider_uid",
			Email:          "test@example.com",
			EmailVerified:  true,
		}, nil)

		// User exists mock
		email, _ := identity.NewEmail("test@example.com")
		username, _ := identity.NewUsername("testuser")
		pwd, _ := identity.NewPasswordHash("hash")

		userID := identity.NewUserID()
		user := identity.ReconstructUser(
			userID, email, username, pwd, identity.RoleUser, identity.StatusActive,
			"User", "Bio", 0, time.Now(), time.Now(), identity.UserTypeRegistered, nil, nil,
		)

		providerUserID, _ := identity.NewProviderUserID("provider_uid")
		mockOAuthRepo.On("FindByProviderAndUserID", mock.Anything, identity.OAuthProviderGoogle, providerUserID).Return(nil, identity.ErrOAuthAccountNotFound)
		mockRepo.On("FindByEmail", mock.Anything, email).Return(user, nil)

		// Create/Update OAuth account
		mockOAuthRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

		// Token generation
		mockJWT.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("access_token", nil)
		mockRefresh.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("refresh_token", &services.RefreshTokenMetadata{ExpiresAt: time.Now().Add(time.Hour)}, nil)
		mockSession.On("Create", mock.Anything, mock.Anything).Return(nil)
		mockJWT.On("GetTokenExpiration", "access_token").Return(time.Now().Add(15*time.Minute), nil)

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify state was deleted
		exists, err := redisClient.Exists(context.Background(), fmt.Sprintf("goimg:oauth:state:%s", state)).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})

	t.Run("MissingState", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()
		redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		logger := zerolog.Nop()

		handler := handlers.NewOAuthHandler(nil, nil, nil, nil, nil, redisClient, logger)

		r := chi.NewRouter()
		r.Get("/auth/oauth/{provider}/callback", handler.HandleCallback)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=auth_code", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("InvalidState", func(t *testing.T) {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		defer mr.Close()
		redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		logger := zerolog.Nop()

		handler := handlers.NewOAuthHandler(nil, nil, nil, nil, nil, redisClient, logger)

		r := chi.NewRouter()
		r.Get("/auth/oauth/{provider}/callback", handler.HandleCallback)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=auth_code&state=invalid", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestOAuthHandler_ListAccounts(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(testhelpers.MockOAuthAccountRepository)
		logger := zerolog.Nop()

		listHandler := queries.NewListOAuthAccountsHandler(mockRepo, &logger)

		handler := handlers.NewOAuthHandler(nil, nil, nil, listHandler, nil, nil, logger)

		r := chi.NewRouter()
		r.Get("/auth/oauth/accounts", handler.ListAccounts)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/accounts", nil)

		// Inject user context
		domainUserID := identity.NewUserID()
		ctx := middleware.SetUserContext(req.Context(), domainUserID.UUID(), "test@example.com", "user", uuid.New(), false)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		mockRepo.On("FindByUserID", mock.Anything, domainUserID).Return([]*identity.OAuthAccount{}, nil)

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.OAuthAccountListDTO
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Accounts)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		logger := zerolog.Nop()
		handler := handlers.NewOAuthHandler(nil, nil, nil, nil, nil, nil, logger)

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/accounts", nil)
		rr := httptest.NewRecorder()

		handler.ListAccounts(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
