package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteError(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Act
	WriteError(rec, req, http.StatusBadRequest, "Invalid Input", "Email is required")

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, problem.Status)
	assert.Equal(t, "Invalid Input", problem.Title)
	assert.Equal(t, "Email is required", problem.Detail)
	assert.Equal(t, "/api/v1/users", problem.Instance)
	assert.Equal(t, "test-request-id", problem.TraceID)
	assert.NotEmpty(t, problem.Timestamp)
	assert.Contains(t, problem.Type, "https://api.goimg.dev/problems")
}

func TestWriteErrorWithExtensions(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	extensions := map[string]interface{}{
		"limit":      100,
		"remaining":  0,
		"retryAfter": 42,
	}

	// Act
	WriteErrorWithExtensions(rec, req, http.StatusTooManyRequests,
		"Rate Limit Exceeded",
		"You have made too many requests",
		extensions)

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, http.StatusTooManyRequests, problem.Status)
	assert.Equal(t, "Rate Limit Exceeded", problem.Title)
	assert.Equal(t, "/api/v1/upload", problem.Instance)
	assert.NotNil(t, problem.Extensions)
	assert.Equal(t, float64(100), problem.Extensions["limit"])
	assert.Equal(t, float64(0), problem.Extensions["remaining"])
	assert.Equal(t, float64(42), problem.Extensions["retryAfter"])
}

func TestWriteProblemDetails(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	problem := ProblemDetails{
		Type:   "https://api.goimg.dev/problems/test",
		Title:  "Test Error",
		Status: http.StatusBadRequest,
		Detail: "This is a test error",
	}

	// Act
	WriteProblemDetails(rec, req, problem)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var result ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "Test Error", result.Title)
	assert.Equal(t, "test-request-id", result.TraceID)
	assert.NotEmpty(t, result.Timestamp)
	assert.Equal(t, "/test", result.Instance)
}

func TestWriteProblemDetails_FillsMissingFields(t *testing.T) {
	t.Parallel()

	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	ctx := SetRequestID(req.Context(), "test-request-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	// Problem with missing traceID, timestamp, and instance
	problem := ProblemDetails{
		Type:   "https://api.goimg.dev/problems/test",
		Title:  "Test Error",
		Status: http.StatusBadRequest,
	}

	// Act
	WriteProblemDetails(rec, req, problem)

	// Assert
	var result ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "test-request-id", result.TraceID, "should fill traceID")
	assert.NotEmpty(t, result.Timestamp, "should fill timestamp")
	assert.Equal(t, "/api/v1/test", result.Instance, "should fill instance")
}

func TestProblemTypeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{
			name:     "bad request",
			status:   http.StatusBadRequest,
			expected: "https://api.goimg.dev/problems/bad-request",
		},
		{
			name:     "unauthorized",
			status:   http.StatusUnauthorized,
			expected: "https://api.goimg.dev/problems/unauthorized",
		},
		{
			name:     "forbidden",
			status:   http.StatusForbidden,
			expected: "https://api.goimg.dev/problems/forbidden",
		},
		{
			name:     "not found",
			status:   http.StatusNotFound,
			expected: "https://api.goimg.dev/problems/not-found",
		},
		{
			name:     "conflict",
			status:   http.StatusConflict,
			expected: "https://api.goimg.dev/problems/conflict",
		},
		{
			name:     "rate limit exceeded",
			status:   http.StatusTooManyRequests,
			expected: "https://api.goimg.dev/problems/rate-limit-exceeded",
		},
		{
			name:     "internal server error",
			status:   http.StatusInternalServerError,
			expected: "https://api.goimg.dev/problems/internal-error",
		},
		{
			name:     "service unavailable",
			status:   http.StatusServiceUnavailable,
			expected: "https://api.goimg.dev/problems/service-unavailable",
		},
		{
			name:     "unknown status",
			status:   999,
			expected: "https://api.goimg.dev/problems/unknown-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := problemTypeURL(tt.status)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapDomainError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedTitle  string
	}{
		{
			name:           "unauthorized error sentinel",
			err:            ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedTitle:  "Unauthorized",
		},
		{
			name:           "unauthorized error message",
			err:            errors.New("authentication failed"),
			expectedStatus: http.StatusUnauthorized,
			expectedTitle:  "Unauthorized",
		},
		{
			name:           "forbidden error sentinel",
			err:            ErrForbidden,
			expectedStatus: http.StatusForbidden,
			expectedTitle:  "Forbidden",
		},
		{
			name:           "forbidden error message",
			err:            errors.New("insufficient permissions"),
			expectedStatus: http.StatusForbidden,
			expectedTitle:  "Forbidden",
		},
		{
			name:           "not found error sentinel",
			err:            ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedTitle:  "Not Found",
		},
		{
			name:           "not found error message",
			err:            errors.New("user not found"),
			expectedStatus: http.StatusNotFound,
			expectedTitle:  "Not Found",
		},
		{
			name:           "conflict error sentinel",
			err:            ErrConflict,
			expectedStatus: http.StatusConflict,
			expectedTitle:  "Conflict",
		},
		{
			name:           "conflict error message - already exists",
			err:            errors.New("email already exists"),
			expectedStatus: http.StatusConflict,
			expectedTitle:  "Conflict",
		},
		{
			name:           "validation error sentinel",
			err:            ErrValidation,
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "Validation Failed",
		},
		{
			name:           "validation error message",
			err:            errors.New("validation failed: email is invalid"),
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "Validation Failed",
		},
		{
			name:           "rate limit error",
			err:            ErrRateLimitExceeded,
			expectedStatus: http.StatusTooManyRequests,
			expectedTitle:  "Rate Limit Exceeded",
		},
		{
			name:           "unknown error",
			err:            errors.New("some random error"),
			expectedStatus: http.StatusInternalServerError,
			expectedTitle:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			status, title, detail := MapDomainError(tt.err)

			// Assert
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedTitle, title)
			assert.NotEmpty(t, detail)
		})
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		str        string
		substrings []string
		expected   bool
	}{
		{
			name:       "contains one substring",
			str:        "user not found",
			substrings: []string{"not found", "missing"},
			expected:   true,
		},
		{
			name:       "contains multiple substrings",
			str:        "email already exists",
			substrings: []string{"already exists", "duplicate"},
			expected:   true,
		},
		{
			name:       "case insensitive match",
			str:        "Unauthorized Access",
			substrings: []string{"unauthorized"},
			expected:   true,
		},
		{
			name:       "no match",
			str:        "some error",
			substrings: []string{"not found", "forbidden"},
			expected:   false,
		},
		{
			name:       "empty substrings",
			str:        "some error",
			substrings: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := containsAny(tt.str, tt.substrings...)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "HeLLo WoRLd",
			expected: "hello world",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with numbers",
			input:    "Test123",
			expected: "test123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := toLower(tt.input)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			str:      "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "does not contain",
			str:      "hello world",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "exact match",
			str:      "test",
			substr:   "test",
			expected: true,
		},
		{
			name:     "empty substring",
			str:      "test",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring at beginning",
			str:      "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at end",
			str:      "hello world",
			substr:   "world",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := contains(tt.str, tt.substr)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		str      string
		substr   string
		expected int
	}{
		{
			name:     "found at beginning",
			str:      "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			str:      "hello world",
			substr:   "lo wo",
			expected: 3,
		},
		{
			name:     "found at end",
			str:      "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			str:      "hello world",
			substr:   "foo",
			expected: -1,
		},
		{
			name:     "empty substring",
			str:      "hello",
			substr:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := indexString(tt.str, tt.substr)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}
