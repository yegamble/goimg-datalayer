package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.Claims), args.Error(1)
}

func (m *MockJWTService) ExtractTokenID(tokenString string) (string, error) {
	args := m.Called(tokenString)
	return args.String(0), args.Error(1)
}

type MockTokenBlacklist struct {
	mock.Mock
}

func (m *MockTokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

// Note: MetricsCollector is optional in AuthConfig, so we use nil in most tests.

func createValidClaims() *jwt.Claims {
	userID := uuid.New()
	sessionID := uuid.New()

	return &jwt.Claims{
		UserID:    userID.String(),
		Email:     "test@example.com",
		Role:      "user",
		SessionID: sessionID.String(),
		TokenType: jwt.TokenTypeAccess,
	}
}

func TestJWTAuth_ValidToken_Success(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	claims := createValidClaims()
	tokenString := "valid.jwt.token"
	tokenID := "token-id-123"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).Return(claims, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	var contextUserID uuid.UUID
	var contextEmail string
	var contextRole string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextUserID, _ = middleware.GetUserID(r.Context())
		contextEmail, _ = middleware.GetUserEmail(r.Context())
		contextRole, _ = middleware.GetUserRole(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())

	assert.Equal(t, claims.UserID, contextUserID.String())
	assert.Equal(t, claims.Email, contextEmail)
	assert.Equal(t, claims.Role, contextRole)

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_MissingAuthorizationHeader_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, problem.Status)
	assert.Equal(t, "Unauthorized", problem.Title)
	assert.Contains(t, problem.Detail, "Missing authorization header")

	mockJWT.AssertNotCalled(t, "ValidateToken", mock.Anything)
	mockBlacklist.AssertNotCalled(t, "IsBlacklisted", mock.Anything, mock.Anything)
}

func TestJWTAuth_InvalidHeaderFormat_NoBearerPrefix_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "InvalidScheme token123")
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid authorization scheme")
}

func TestJWTAuth_MalformedHeader_SinglePart_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "OnlyOnePartNoSpace")
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid authorization header format")
}

func TestJWTAuth_EmptyToken_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Authorization token is empty")
}

func TestJWTAuth_BlacklistedToken_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "blacklisted.jwt.token"
	tokenID := "blacklisted-token-id"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(true, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Token has been revoked")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
	mockJWT.AssertNotCalled(t, "ValidateToken", mock.Anything)
}

func TestJWTAuth_BlacklistCheckError_Returns500(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "valid.jwt.token"
	tokenID := "token-id"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).
		Return(false, errors.New("redis connection error"))

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Authentication service temporarily unavailable")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "expired.jwt.token"
	tokenID := "token-id"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).
		Return(nil, errors.New("token is expired"))

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid or expired token")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_InvalidSignature_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "invalid.signature.token"
	tokenID := "token-id"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).
		Return(nil, errors.New("signature verification failed"))

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid or expired token")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_WrongTokenType_RefreshInsteadOfAccess_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "refresh.jwt.token"
	tokenID := "token-id"

	claims := createValidClaims()
	claims.TokenType = jwt.TokenTypeRefresh

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).Return(claims, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid token type")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_InvalidUserID_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "valid.jwt.token"
	tokenID := "token-id"

	claims := createValidClaims()
	claims.UserID = "not-a-valid-uuid"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).Return(claims, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid token claims")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_InvalidSessionID_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "valid.jwt.token"
	tokenID := "token-id"

	claims := createValidClaims()
	claims.SessionID = "invalid-session-id"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).Return(claims, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid token claims")

	mockJWT.AssertExpectations(t)
	mockBlacklist.AssertExpectations(t)
}

func TestJWTAuth_OptionalMode_MissingToken_PassesThrough(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         true,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := middleware.GetUserID(r.Context())
		assert.False(t, ok, "no user context should be set")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("public access"))
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "public access", rr.Body.String())

	mockJWT.AssertNotCalled(t, "ValidateToken", mock.Anything)
	mockJWT.AssertNotCalled(t, "ExtractTokenID", mock.Anything)
	mockBlacklist.AssertNotCalled(t, "IsBlacklisted", mock.Anything, mock.Anything)
}

func TestJWTAuth_OptionalMode_InvalidToken_PassesThrough(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         true,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("public access"))
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "public access", rr.Body.String())
}

func TestJWTAuth_ExtractTokenIDError_Returns401(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	tokenString := "malformed.token"

	mockJWT.On("ExtractTokenID", tokenString).
		Return("", errors.New("failed to parse token"))

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Contains(t, problem.Detail, "Invalid token format")

	mockJWT.AssertExpectations(t)
}

func TestRequireRole_UserHasRequiredRole_PassesThrough(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireRole(logger, nil, "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("admin access granted"))
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin", nil)

	userID := uuid.New()
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "admin@example.com", "admin", sessionID, false, true)
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "admin access granted", rr.Body.String())
}

func TestRequireRole_UserLacksRequiredRole_Returns403(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireRole(logger, nil, "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin", nil)

	userID := uuid.New()
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false, true)
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, "Forbidden", problem.Title)
	assert.Contains(t, problem.Detail, "This action requires admin role")
}

func TestRequireRole_NoUserContext_Returns401(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireRole(logger, nil, "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin", nil)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, "Unauthorized", problem.Title)
	assert.Contains(t, problem.Detail, "User role not found in context")
}

func TestRequireAnyRole_UserHasFirstAllowedRole_PassesThrough(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireAnyRole(logger, nil, "moderator", "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("moderator access granted"))
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/moderate", nil)

	userID := uuid.New()
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "mod@example.com", "moderator", sessionID, false, true)
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "moderator access granted", rr.Body.String())
}

func TestRequireAnyRole_UserHasSecondAllowedRole_PassesThrough(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireAnyRole(logger, nil, "moderator", "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("admin access granted"))
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/moderate", nil)

	userID := uuid.New()
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "admin@example.com", "admin", sessionID, false, true)
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "admin access granted", rr.Body.String())
}

func TestRequireAnyRole_UserHasNoneOfAllowedRoles_Returns403(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireAnyRole(logger, nil, "moderator", "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/moderate", nil)

	userID := uuid.New()
	sessionID := uuid.New()
	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false, true)
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, "Forbidden", problem.Title)
	assert.Contains(t, problem.Detail, "This action requires one of the following roles")
}

func TestRequireAnyRole_NoUserContext_Returns401(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	roleMiddleware := middleware.RequireAnyRole(logger, nil, "moderator", "admin")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := roleMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/moderate", nil)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, "Unauthorized", problem.Title)
	assert.Contains(t, problem.Detail, "User role not found in context")
}

func TestJWTAuth_BearerPrefix_CaseInsensitive(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	claims := createValidClaims()
	tokenString := "valid.jwt.token"
	tokenID := "token-id-123"

	mockJWT.On("ExtractTokenID", tokenString).Return(tokenID, nil)
	mockBlacklist.On("IsBlacklisted", mock.Anything, tokenID).Return(false, nil)
	mockJWT.On("ValidateToken", tokenString).Return(claims, nil)

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := authMiddleware(testHandler)

	testCases := []string{
		"Bearer " + tokenString,
		"bearer " + tokenString,
		"BEARER " + tokenString,
		"BeArEr " + tokenString,
	}

	for _, authHeader := range testCases {
		t.Run(authHeader, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			req.Header.Set("Authorization", authHeader)
			ctx := middleware.SetRequestID(req.Context(), "test-request-id")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestJWTAuth_RFC7807ErrorFormat(t *testing.T) {
	t.Parallel()

	mockJWT := new(MockJWTService)
	mockBlacklist := new(MockTokenBlacklist)
	logger := zerolog.Nop()

	cfg := middleware.AuthConfig{
		JWTService:       mockJWT,
		TokenBlacklist:   mockBlacklist,
		MetricsCollector: nil,
		Logger:           logger,
		Optional:         false,
	}

	authMiddleware := middleware.JWTAuth(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	wrappedHandler := authMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	ctx := middleware.SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))

	var problem middleware.ProblemDetails
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err)

	assert.NotEmpty(t, problem.Type, "type field should be present")
	assert.NotEmpty(t, problem.Title, "title field should be present")
	assert.NotZero(t, problem.Status, "status field should be present")
	assert.NotEmpty(t, problem.Detail, "detail field should be present")
	assert.NotEmpty(t, problem.Instance, "instance field should be present")
	assert.NotEmpty(t, problem.TraceID, "traceId field should be present")
	assert.NotEmpty(t, problem.Timestamp, "timestamp field should be present")

	assert.Equal(t, "/api/v1/protected", problem.Instance)
	assert.Equal(t, "test-request-id", problem.TraceID)
	assert.Contains(t, problem.Type, "https://api.goimg.dev/problems")
}
