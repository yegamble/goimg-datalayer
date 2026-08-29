package handlers_test

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

	"github.com/yegamble/goimg-datalayer/internal/application/moderation/queries"
	"github.com/yegamble/goimg-datalayer/internal/application/moderation/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func TestModerationHandler_GetUserBanStatus(t *testing.T) {
	// Setup mocks
	mockBanRepo := new(testhelpers.MockBanRepository)
	logger := zerolog.Nop()

	// Instantiate the query handler with the mock repo
	getBanStatusHandler := queries.NewGetUserBanStatusHandler(mockBanRepo)

	// Instantiate the moderation handler
	// We only need to provide the getBanStatusHandler, others can be nil as they are not used in this test
	modHandler := handlers.NewModerationHandler(
		nil, nil, nil, nil, // report command handlers
		nil, nil, // ban command handlers
		nil, // nsfw command handler
		nil, nil, // report query handlers
		getBanStatusHandler, // ban query handler
		nil, // list bans query handler
		nil, nil, nil, // nsfw query handlers
		logger,
	)

	t.Run("Unauthorized_AnotherUser_NoRole", func(t *testing.T) {
		// User A tries to check ban status of User B
		requestingUserID := uuid.New()
		targetUserID := uuid.New()
		sessionID := uuid.New()

		// Create request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetUserID.String()+"/ban", nil)

		// Set authenticated user context (User A)
		ctx := middleware.SetUserContext(req.Context(), requestingUserID, "user@example.com", "user", sessionID, false, true)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Setup router context for path param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userID", targetUserID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Execute handler
		modHandler.GetUserBanStatus(rr, req)

		// Assertions
		// Should be Forbidden because User A cannot check User B's ban status
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("Authorized_Self", func(t *testing.T) {
		// User A checks their own ban status
		userID := uuid.New()
		sessionID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/ban", nil)

		ctx := middleware.SetUserContext(req.Context(), userID, "user@example.com", "user", sessionID, false, true)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userID", userID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Mock expectations - should call repository
		mockBanRepo.On("IsUserBanned", mock.Anything, mock.Anything).Return(false, nil).Once()

		modHandler.GetUserBanStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Authorized_Admin", func(t *testing.T) {
		// Admin checks User B's ban status
		adminID := uuid.New()
		targetUserID := uuid.New()
		sessionID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetUserID.String()+"/ban", nil)

		ctx := middleware.SetUserContext(req.Context(), adminID, "admin@example.com", "admin", sessionID, false, true)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userID", targetUserID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Mock expectations - should call repository
		mockBanRepo.On("IsUserBanned", mock.Anything, mock.Anything).Return(true, nil).Once()
		// Since IsUserBanned returns true, it will call FindByUserID
		// We mock FindByUserID to return an error to simplify test (or a mock ban)
		// Assuming queries.GetUserBanStatusHandler implementation
		// If FindByUserID fails, it returns isBanned=true but no details, which is fine for this test
		// Or we can verify it doesn't fail. Let's return error to keep it simple.
		// Actually, let's return a ban object to be correct. But mocking a ban object might be complex.
		// Let's just return error so it returns result without details.
		// Wait, testhelpers.MockBanRepository.FindByUserID returns (*moderation.Ban, error).
		// I'll return an error "ban not found" which causes the handler to return basic info.
		// This is enough to prove authorization passed.
		mockBanRepo.On("FindByUserID", mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()

		modHandler.GetUserBanStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Authorized_Moderator", func(t *testing.T) {
		// Moderator checks User B's ban status
		modID := uuid.New()
		targetUserID := uuid.New()
		sessionID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetUserID.String()+"/ban", nil)

		ctx := middleware.SetUserContext(req.Context(), modID, "mod@example.com", "moderator", sessionID, false, true)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("userID", targetUserID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Mock expectations
		mockBanRepo.On("IsUserBanned", mock.Anything, mock.Anything).Return(false, nil).Once()

		modHandler.GetUserBanStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
