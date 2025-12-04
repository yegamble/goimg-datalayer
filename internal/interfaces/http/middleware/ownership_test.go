package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockOwnershipChecker is a mock implementation of OwnershipChecker.
type MockOwnershipChecker struct {
	mock.Mock
}

func (m *MockOwnershipChecker) CheckOwnership(ctx context.Context, userID, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockOwnershipChecker) ExistsByID(ctx context.Context, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, resourceID)
	return args.Bool(0), args.Error(1)
}

func TestRequireOwnership_Success(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()
	resourceID := uuid.New()

	// Setup mocks
	mockChecker.On("ExistsByID", mock.Anything, resourceID).Return(true, nil)
	mockChecker.On("CheckOwnership", mock.Anything, userID, resourceID).Return(true, nil)

	// Create middleware
	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
		AllowAdmins:  true,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/images/"+resourceID.String(), nil)

	// Setup chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Set user context
	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	// Execute
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
	mockChecker.AssertExpectations(t)
}

func TestRequireOwnership_NoUserContext(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	resourceID := uuid.New()

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	// Create request WITHOUT user context
	req := httptest.NewRequest(http.MethodGet, "/images/"+resourceID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireOwnership_MissingResourceID(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	// Create request WITHOUT resource ID in URL
	req := httptest.NewRequest(http.MethodGet, "/images", nil)
	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	// Add empty route context
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRequireOwnership_InvalidResourceID(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	// Create request with INVALID resource ID
	req := httptest.NewRequest(http.MethodGet, "/images/invalid-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRequireOwnership_ResourceNotFound(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()
	resourceID := uuid.New()

	// Setup mocks - resource does not exist
	mockChecker.On("ExistsByID", mock.Anything, resourceID).Return(false, nil)

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/images/"+resourceID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockChecker.AssertExpectations(t)
}

func TestRequireOwnership_NotOwner(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()
	resourceID := uuid.New()

	// Setup mocks - resource exists but user is not owner
	mockChecker.On("ExistsByID", mock.Anything, resourceID).Return(true, nil)
	mockChecker.On("CheckOwnership", mock.Anything, userID, resourceID).Return(false, nil)

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
		AllowAdmins:  false,
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/images/"+resourceID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, rr.Code)
	mockChecker.AssertExpectations(t)
}

func TestRequireOwnership_AdminBypass(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()
	resourceID := uuid.New()

	// Setup mocks - only check existence, not ownership
	mockChecker.On("ExistsByID", mock.Anything, resourceID).Return(true, nil)

	cfg := middleware.OwnershipConfig{
		ResourceType: middleware.ResourceTypeImage,
		Checker:      mockChecker,
		URLParam:     "imageID",
		Logger:       logger,
		AllowAdmins:  true, // Admin bypass enabled
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin access"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/images/"+resourceID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Set user context with ADMIN role
	ctx := middleware.SetUserContext(req.Context(), userID, "admin@example.com", "admin", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "admin access", rr.Body.String())
	mockChecker.AssertExpectations(t)
	// CheckOwnership should NOT be called for admins
	mockChecker.AssertNotCalled(t, "CheckOwnership", mock.Anything, mock.Anything, mock.Anything)
}

func TestRequireOwnership_ModeratorBypass(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	userID := uuid.New()
	resourceID := uuid.New()

	// Setup mocks - only check existence
	mockChecker.On("ExistsByID", mock.Anything, resourceID).Return(true, nil)

	cfg := middleware.OwnershipConfig{
		ResourceType:    middleware.ResourceTypeComment,
		Checker:         mockChecker,
		URLParam:        "commentID",
		Logger:          logger,
		AllowModerators: true, // Moderator bypass enabled
	}

	handler := middleware.RequireOwnership(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("moderator access"))
	}))

	req := httptest.NewRequest(http.MethodDelete, "/comments/"+resourceID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("commentID", resourceID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Set user context with MODERATOR role
	ctx := middleware.SetUserContext(req.Context(), userID, "mod@example.com", "moderator", uuid.New())
	ctx = middleware.SetRequestID(ctx, "test-request-id")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "moderator access", rr.Body.String())
	mockChecker.AssertExpectations(t)
	mockChecker.AssertNotCalled(t, "CheckOwnership", mock.Anything, mock.Anything, mock.Anything)
}

func TestRequireImageOwnership(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	handler := middleware.RequireImageOwnership(mockChecker, logger)
	assert.NotNil(t, handler)
}

func TestRequireAlbumOwnership(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	handler := middleware.RequireAlbumOwnership(mockChecker, logger)
	assert.NotNil(t, handler)
}

func TestRequireCommentOwnership(t *testing.T) {
	mockChecker := new(MockOwnershipChecker)
	logger := zerolog.Nop()

	handler := middleware.RequireCommentOwnership(mockChecker, logger)
	assert.NotNil(t, handler)
}
