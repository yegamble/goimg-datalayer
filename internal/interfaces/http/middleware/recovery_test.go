package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery_NoPanic_PassesThrough(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
}

func TestRecovery_PanicWithError_Returns500(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("something went wrong"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, problem.Status)
	assert.Equal(t, "Internal Server Error", problem.Title)
	assert.Contains(t, problem.Detail, "An unexpected error occurred")
	assert.Equal(t, "test-request-id", problem.TraceID)
}

func TestRecovery_PanicWithString_Returns500(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic message")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, problem.Status)
}

func TestRecovery_PanicWithInt_Returns500(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecovery_WithUserContext_LogsUserID(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()
	userID := uuid.New()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("test error"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	ctx = SetUserContext(ctx, userID, "test@example.com", "user", uuid.New(), false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, "test-request-id", problem.TraceID)
}

func TestRecovery_NoRequestID_StillRecovers(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("test error"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No request ID in context
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, problem.Status)
}

func TestRecovery_PanicInMiddleware_Recovers(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	panicMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("middleware panic")
		})
	}

	handler := Recovery(logger)(panicMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecovery_DoesNotExposeStackTrace(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("internal error with sensitive info"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	// Verify no stack trace or sensitive info in response
	assert.NotContains(t, problem.Detail, "goroutine")
	assert.NotContains(t, problem.Detail, "panic")
	assert.NotContains(t, problem.Detail, "sensitive info")

	// Generic error message only
	assert.Contains(t, problem.Detail, "An unexpected error occurred")
}

func TestRecovery_RFC7807Format(t *testing.T) {
	t.Parallel()

	// Arrange
	logger := zerolog.Nop()

	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("test error"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	// Verify RFC 7807 structure
	assert.NotEmpty(t, problem.Type, "type field should be present")
	assert.NotEmpty(t, problem.Title, "title field should be present")
	assert.NotZero(t, problem.Status, "status field should be present")
	assert.NotEmpty(t, problem.Detail, "detail field should be present")
	assert.NotEmpty(t, problem.Instance, "instance field should be present")
	assert.NotEmpty(t, problem.TraceID, "traceId field should be present")
	assert.NotEmpty(t, problem.Timestamp, "timestamp field should be present")

	// Verify correct values
	assert.Equal(t, "/api/v1/test", problem.Instance)
	assert.Equal(t, "test-request-id", problem.TraceID)
	assert.Contains(t, problem.Type, "https://api.goimg.dev/problems")
}

func TestIsResponseWritten(t *testing.T) {
	t.Parallel()

	t.Run("wrapped response writer - header written", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: rec,
			wroteHeader:    true,
		}

		result := isResponseWritten(rw)
		assert.True(t, result)
	})

	t.Run("wrapped response writer - header not written", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: rec,
			wroteHeader:    false,
		}

		result := isResponseWritten(rw)
		assert.False(t, result)
	})

	t.Run("unwrapped response writer - returns false", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		result := isResponseWritten(rec)
		assert.False(t, result, "should return false for unwrapped writers")
	})
}
