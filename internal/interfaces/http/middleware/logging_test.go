package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_LogsRequest(t *testing.T) {
	t.Parallel()

	// Arrange
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer)

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "response body", rec.Body.String())

	// Verify log was written
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "test-request-id")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/api/v1/test")
	assert.Contains(t, logOutput, "http request completed")
}

func TestLogger_CapturesStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var logBuffer bytes.Buffer
			logger := zerolog.New(&logBuffer)

			handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := SetRequestID(req.Context(), "test-request-id")
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rec, req)

			// Assert
			assert.Equal(t, tt.statusCode, rec.Code)

			// Verify status code is logged
			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, "status")
		})
	}
}

func TestLogger_WithUserContext(t *testing.T) {
	t.Parallel()

	// Arrange
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer)
	userID := uuid.New()

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	ctx = SetUserContext(ctx, userID, "test@example.com", "user", uuid.New(), false)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, userID.String())
}

func TestLogger_WithQueryParams(t *testing.T) {
	t.Parallel()

	// Arrange
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer)

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?page=1&limit=10", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "page=1&limit=10")
}

func TestLogger_TracksBytesWritten(t *testing.T) {
	t.Parallel()

	// Arrange
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer)

	responseBody := "test response body"
	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "bytes_written")
}

func TestLogLevelForStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   int
		expected zerolog.Level
	}{
		{"2xx success", http.StatusOK, zerolog.InfoLevel},
		{"2xx created", http.StatusCreated, zerolog.InfoLevel},
		{"3xx redirect", http.StatusMovedPermanently, zerolog.InfoLevel},
		{"4xx bad request", http.StatusBadRequest, zerolog.WarnLevel},
		{"4xx unauthorized", http.StatusUnauthorized, zerolog.WarnLevel},
		{"4xx not found", http.StatusNotFound, zerolog.WarnLevel},
		{"5xx internal error", http.StatusInternalServerError, zerolog.ErrorLevel},
		{"5xx bad gateway", http.StatusBadGateway, zerolog.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			level := logLevelForStatus(tt.status)

			// Assert
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:54321"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "192.168.1.1", ip)
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
	req.RemoteAddr = "192.168.1.1:54321"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "203.0.113.1", ip, "should extract first IP from X-Forwarded-For")
}

func TestGetClientIP_XRealIP(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "203.0.113.1")
	req.RemoteAddr = "192.168.1.1:54321"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "203.0.113.1", ip)
}

func TestGetClientIP_IPv6(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:54321"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "2001:db8::1", ip)
}

func TestGetClientIP_NoPort(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "192.168.1.1", ip)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		status:         0,
		wroteHeader:    false,
	}

	// Act
	rw.WriteHeader(http.StatusCreated)

	// Assert
	assert.Equal(t, http.StatusCreated, rw.status)
	assert.True(t, rw.wroteHeader)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestResponseWriter_WriteHeader_CalledMultipleTimes(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		status:         0,
		wroteHeader:    false,
	}

	// Act
	rw.WriteHeader(http.StatusOK)
	rw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	// Assert
	assert.Equal(t, http.StatusOK, rw.status, "first status should be preserved")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponseWriter_Write(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		status:         0,
		wroteHeader:    false,
		bytesWritten:   0,
	}

	// Act
	n, err := rw.Write([]byte("test data"))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, int64(9), rw.bytesWritten)
	assert.Equal(t, http.StatusOK, rw.status, "Write should set default status")
	assert.True(t, rw.wroteHeader)
}

func TestResponseWriter_Write_MultipleCalls(t *testing.T) {
	t.Parallel()

	// Arrange
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		status:         0,
		wroteHeader:    false,
		bytesWritten:   0,
	}

	// Act
	_, _ = rw.Write([]byte("first"))
	_, _ = rw.Write([]byte("second"))

	// Assert
	assert.Equal(t, int64(11), rw.bytesWritten)
	assert.Equal(t, "firstsecond", rec.Body.String())
}

func TestLogger_ContentType(t *testing.T) {
	t.Parallel()

	// Arrange
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer)

	handler := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "application/json")
}

func TestGetClientIP_IPv6_MalformedBracket(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[incomplete"

	// Act
	ip := getClientIP(req)

	// Assert - should handle gracefully
	assert.NotEmpty(t, ip)
}

func TestGetClientIP_SingleXForwardedFor(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.RemoteAddr = "192.168.1.1:54321"

	// Act
	ip := getClientIP(req)

	// Assert
	assert.Equal(t, "203.0.113.1", ip, "should extract single IP from X-Forwarded-For")
}
