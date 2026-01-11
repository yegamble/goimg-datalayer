package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		assert.NotEmpty(t, requestID)

		// Verify it's a valid UUID
		_, err := uuid.Parse(requestID)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify X-Request-ID header is set in response
	responseRequestID := rec.Header().Get("X-Request-ID")
	assert.NotEmpty(t, responseRequestID)

	// Verify it's a valid UUID
	_, err := uuid.Parse(responseRequestID)
	assert.NoError(t, err)
}

func TestRequestID_AcceptsValidClientRequestID(t *testing.T) {
	t.Parallel()

	// Arrange
	clientRequestID := uuid.New().String()
	var capturedRequestID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", clientRequestID)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the client-provided request ID is used
	assert.Equal(t, clientRequestID, capturedRequestID)

	// Verify X-Request-ID header matches client ID
	responseRequestID := rec.Header().Get("X-Request-ID")
	assert.Equal(t, clientRequestID, responseRequestID)
}

func TestRequestID_RejectsInvalidClientRequestID(t *testing.T) {
	t.Parallel()

	// Arrange
	invalidRequestID := "not-a-valid-uuid"
	var capturedRequestID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", invalidRequestID)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify a new request ID was generated (not the invalid one)
	assert.NotEqual(t, invalidRequestID, capturedRequestID)
	assert.NotEmpty(t, capturedRequestID)

	// Verify it's a valid UUID
	_, err := uuid.Parse(capturedRequestID)
	assert.NoError(t, err)

	// Verify X-Request-ID header contains the generated ID
	responseRequestID := rec.Header().Get("X-Request-ID")
	assert.Equal(t, capturedRequestID, responseRequestID)
}

func TestRequestID_EmptyClientRequestID(t *testing.T) {
	t.Parallel()

	// Arrange
	var capturedRequestID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify a new request ID was generated
	assert.NotEmpty(t, capturedRequestID)

	// Verify it's a valid UUID
	_, err := uuid.Parse(capturedRequestID)
	assert.NoError(t, err)
}

func TestRequestID_MultipleCalls_DifferentIDs(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Act - Make two separate requests
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Assert - Each request should have a different ID
	requestID1 := rec1.Header().Get("X-Request-ID")
	requestID2 := rec2.Header().Get("X-Request-ID")

	require.NotEmpty(t, requestID1)
	require.NotEmpty(t, requestID2)
	assert.NotEqual(t, requestID1, requestID2)
}

func TestRequestID_ContextPropagation(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedRequestID := uuid.New().String()

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID := GetRequestID(r.Context())
		assert.Equal(t, expectedRequestID, requestID)

		// Simulate calling another function that needs request ID
		innerFunc(r.Context(), t, expectedRequestID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", expectedRequestID)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
}

// innerFunc simulates a function that needs the request ID from context
func innerFunc(ctx context.Context, t *testing.T, expectedID string) {
	t.Helper()
	requestID := GetRequestID(ctx)
	assert.Equal(t, expectedID, requestID)
}
