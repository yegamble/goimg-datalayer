package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/rs/zerolog"

	"github.com/yegamble/goimg-datalayer/internal/interfaces/http/handlers"
)

func TestImageHandler_WithRateLimiter(t *testing.T) {
	// specific middleware to test execution
	middlewareCalled := false
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Test-Middleware", "executed")
			next.ServeHTTP(w, r)
		})
	}

	// Create handler with nil dependencies
	// This is safe because h.Upload fails at authentication step (checking context)
	// before accessing any nil command handlers.
	h := handlers.NewImageHandler(
		nil, nil, nil, nil, nil, nil, nil, nil,
		zerolog.Nop(),
	)

	// Inject middleware
	h.WithRateLimiter(testMiddleware)

	// Get router
	r := h.Routes()

	// Create request to the upload endpoint
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	// Execute request
	// This should run the middleware, then call h.Upload, which will return 401
	// because of missing user context, safely avoiding nil pointer dereference.
	r.ServeHTTP(w, req)

	// Assert middleware execution
	assert.True(t, middlewareCalled, "Middleware should have been executed")
	assert.Equal(t, "executed", w.Header().Get("X-Test-Middleware"), "Middleware header should be present")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")
}
