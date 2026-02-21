package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/middleware"
)

func makeUnverifiedUserContext(ctx context.Context) context.Context {
	userID := identity.NewUserID()
	sessionID := uuid.New()
	return middleware.SetUserContext(ctx, userID.UUID(), "unverified@example.com", "user", sessionID, false, false)
}

func makeVerifiedUserContext(ctx context.Context) context.Context {
	userID := identity.NewUserID()
	sessionID := uuid.New()
	return middleware.SetUserContext(ctx, userID.UUID(), "verified@example.com", "user", sessionID, false, true)
}

func assertForbiddenEmailVerification(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusForbidden, rec.Code, "expected 403 Forbidden for unverified email")

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err == nil {
		if detail, ok := body["detail"].(string); ok {
			assert.Contains(t, strings.ToLower(detail), "email", "response body should mention email verification")
		}
	}
}

func TestImageHandler_Upload_Returns403WhenEmailUnverified(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	imageHandler := NewImageHandler(nil, nil, nil, nil, nil, nil, nil, nil, "", logger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", nil)
	ctx := makeUnverifiedUserContext(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	imageHandler.Upload(rec, req)

	assertForbiddenEmailVerification(t, rec)
}

func TestImageHandler_Upload_AllowsVerifiedUser(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	imageHandler := NewImageHandler(nil, nil, nil, nil, nil, nil, nil, nil, "", logger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", nil)
	ctx := makeVerifiedUserContext(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	imageHandler.Upload(rec, req)

	assert.NotEqual(t, http.StatusForbidden, rec.Code,
		"verified user should not receive 403 from email check")
}

func TestSocialHandler_AddComment_Returns403WhenEmailUnverified(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	socialHandler := NewSocialHandler(nil, nil, nil, nil, nil, nil, logger)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+uuid.New().String()+"/comments", nil)
	ctx := makeUnverifiedUserContext(req.Context())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	socialHandler.AddComment(rec, req)

	assertForbiddenEmailVerification(t, rec)
}

func TestSocialHandler_AddComment_AllowsVerifiedUser(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	socialHandler := NewSocialHandler(nil, nil, nil, nil, nil, nil, logger)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("imageID", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+uuid.New().String()+"/comments", nil)
	req.Header.Set("Content-Type", "application/json")

	ctx := makeVerifiedUserContext(req.Context())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	socialHandler.AddComment(rec, req)

	assert.NotEqual(t, http.StatusForbidden, rec.Code,
		"verified user should not receive 403 from email check")
}

func TestGroupHandler_CreateGroup_Returns403WhenEmailUnverified(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	groupHandler := NewGroupHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
		logger,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", nil)
	ctx := makeUnverifiedUserContext(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	groupHandler.CreateGroup(rec, req)

	assertForbiddenEmailVerification(t, rec)
}

func TestGroupHandler_CreateGroup_AllowsVerifiedUser(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	groupHandler := NewGroupHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
		logger,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", nil)
	req.Header.Set("Content-Type", "application/json")

	ctx := makeVerifiedUserContext(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	groupHandler.CreateGroup(rec, req)

	require.NotEqual(t, http.StatusForbidden, rec.Code,
		"verified user should not receive 403 from email check")
}
