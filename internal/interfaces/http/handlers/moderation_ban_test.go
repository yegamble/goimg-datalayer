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
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

// MockBanRepositoryLocal is a local mock for testing
type MockBanRepositoryLocal struct {
	mock.Mock
}

func (m *MockBanRepositoryLocal) NextID() moderation.BanID {
	return moderation.BanID{}
}

func (m *MockBanRepositoryLocal) FindByID(ctx context.Context, id moderation.BanID) (*moderation.Ban, error) {
	return nil, moderation.ErrBanNotFound
}

func (m *MockBanRepositoryLocal) IsUserBanned(ctx context.Context, userID identity.UserID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockBanRepositoryLocal) FindByUserID(ctx context.Context, userID identity.UserID) (*moderation.Ban, error) {
	return nil, moderation.ErrBanNotFound
}

func (m *MockBanRepositoryLocal) FindActiveBans(ctx context.Context) ([]*moderation.Ban, error) {
	return nil, nil
}

func (m *MockBanRepositoryLocal) FindExpiredBans(ctx context.Context) ([]*moderation.Ban, error) {
	return nil, nil
}

func (m *MockBanRepositoryLocal) Save(ctx context.Context, ban *moderation.Ban) error {
	return nil
}

func TestGetUserBanStatus_IDOR(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	sessionID := uuid.New()

	testCases := []struct {
		name           string
		requestUserID  string
		contextUserID  uuid.UUID
		contextRole    string
		expectedStatus int
	}{
		{
			name:           "User requesting own ban status",
			requestUserID:  userID.String(),
			contextUserID:  userID,
			contextRole:    "user",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User requesting other's ban status (IDOR attempt)",
			requestUserID:  otherUserID.String(),
			contextUserID:  userID,
			contextRole:    "user",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Admin requesting other's ban status",
			requestUserID:  otherUserID.String(),
			contextUserID:  userID,
			contextRole:    "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Moderator requesting other's ban status",
			requestUserID:  otherUserID.String(),
			contextUserID:  userID,
			contextRole:    "moderator",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock repository
			mockRepo := new(MockBanRepositoryLocal)

			// Setup only needed if not forbidden
			if tc.expectedStatus == http.StatusOK {
				mockRepo.On("IsUserBanned", mock.Anything, mock.Anything).Return(false, nil)
			}

			// Handlers
			q := queries.NewGetUserBanStatusHandler(mockRepo)

			logger := zerolog.Nop()
			handler := handlers.NewModerationHandler(
				nil, // createReport
				nil, // startReview
				nil, // resolveReport
				nil, // dismissReport
				nil, // banUser
				nil, // unbanUser
				nil, // scanNSFW
				nil, // getReport
				nil, // listPendingReports
				q,   // getBanStatus
				nil, // listBans
				nil, // getNSFWScan
				nil, // listNSFWFlagged
				nil, // listNSFWScansByImage
				logger,
			)

			// Setup request
			req := httptest.NewRequest(http.MethodGet, "/users/"+tc.requestUserID+"/ban", nil)

			// Set chi URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", tc.requestUserID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Set user context
			ctx := req.Context()
			ctx = context.WithValue(ctx, middleware.UserIDKey, tc.contextUserID)
			ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
			ctx = context.WithValue(ctx, middleware.UserRoleKey, tc.contextRole)
			ctx = context.WithValue(ctx, middleware.SessionIDKey, sessionID)
			ctx = context.WithValue(ctx, middleware.TwoFAVerifiedKey, false)
			ctx = context.WithValue(ctx, middleware.EmailVerifiedKey, false)
			req = req.WithContext(ctx)

			// Execute
			rr := httptest.NewRecorder()
			handler.GetUserBanStatus(rr, req)

			// Assert
			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}
