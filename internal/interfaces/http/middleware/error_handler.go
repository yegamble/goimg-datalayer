package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ProblemDetails represents an RFC 7807 Problem Details response.
// This standard format provides machine-readable error details for HTTP APIs.
//
// Example response:
//
//	{
//	  "type": "https://api.goimg.dev/problems/validation-error",
//	  "title": "Validation Failed",
//	  "status": 400,
//	  "detail": "Email format is invalid",
//	  "instance": "/api/v1/users",
//	  "traceId": "550e8400-e29b-41d3-a456-426614174000",
//	  "timestamp": "2024-11-10T15:30:00Z"
//	}
//
// Reference: https://datatracker.ietf.org/doc/html/rfc7807
type ProblemDetails struct {
	// Type is a URI reference that identifies the problem type.
	// When dereferenced, it should provide human-readable documentation.
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem type.
	// It SHOULD NOT change between occurrences of the same problem type.
	Title string `json:"title"`

	// Status is the HTTP status code for this occurrence.
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference that identifies the specific occurrence.
	// Typically the request path that caused the error.
	Instance string `json:"instance,omitempty"`

	// TraceID is the request correlation ID for debugging and log aggregation.
	TraceID string `json:"traceId,omitempty"`

	// Timestamp is when the error occurred (ISO 8601 format).
	Timestamp string `json:"timestamp,omitempty"`

	// Extensions holds additional problem-specific fields.
	// For validation errors: {"errors": {"email": "invalid format"}}
	// For rate limits: {"retryAfter": 42, "limit": 100}
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// WriteError writes an RFC 7807 Problem Details response.
// It automatically includes the request ID (traceId), timestamp, and instance path.
//
// Usage:
//
//	middleware.WriteError(w, r, http.StatusBadRequest, "Invalid Input", "Email is required")
func WriteError(w http.ResponseWriter, r *http.Request, status int, title string, detail string) {
	problem := ProblemDetails{
		Type:      problemTypeURL(status),
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  r.URL.Path,
		TraceID:   GetRequestID(r.Context()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	WriteProblemDetails(w, r, problem)
}

// WriteErrorWithExtensions writes an RFC 7807 Problem Details response with custom extensions.
//
// Usage:
//
//	middleware.WriteErrorWithExtensions(w, r, http.StatusTooManyRequests,
//	    "Rate Limit Exceeded",
//	    "You have made too many requests",
//	    map[string]interface{}{
//	        "limit": 100,
//	        "remaining": 0,
//	        "retryAfter": 42,
//	    })
func WriteErrorWithExtensions(
	w http.ResponseWriter, r *http.Request,
	status int, title string, detail string,
	extensions map[string]interface{},
) {
	problem := ProblemDetails{
		Type:       problemTypeURL(status),
		Title:      title,
		Status:     status,
		Detail:     detail,
		Instance:   r.URL.Path,
		TraceID:    GetRequestID(r.Context()),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Extensions: extensions,
	}

	WriteProblemDetails(w, r, problem)
}

// WriteProblemDetails writes a ProblemDetails struct as JSON response.
func WriteProblemDetails(w http.ResponseWriter, r *http.Request, problem ProblemDetails) {
	// Ensure traceId and timestamp are set
	if problem.TraceID == "" {
		problem.TraceID = GetRequestID(r.Context())
	}
	if problem.Timestamp == "" {
		problem.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if problem.Instance == "" {
		problem.Instance = r.URL.Path
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)

	// JSON encoding error should not happen with our struct, but handle it gracefully
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		// Fallback: write a plain text error (we can't use JSON at this point)
		w.Header().Set("Content-Type", "text/plain")
		// Best-effort write; ignore error since response is already compromised
		_, _ = fmt.Fprintf(w, "Internal error encoding problem details: %v\n", err)
	}
}

// problemTypeURL generates the problem type URL based on HTTP status code.
func problemTypeURL(status int) string {
	baseURL := "https://api.goimg.dev/problems"

	switch status {
	case http.StatusBadRequest:
		return baseURL + "/bad-request"
	case http.StatusUnauthorized:
		return baseURL + "/unauthorized"
	case http.StatusForbidden:
		return baseURL + "/forbidden"
	case http.StatusNotFound:
		return baseURL + "/not-found"
	case http.StatusConflict:
		return baseURL + "/conflict"
	case http.StatusTooManyRequests:
		return baseURL + "/rate-limit-exceeded"
	case http.StatusInternalServerError:
		return baseURL + "/internal-error"
	case http.StatusServiceUnavailable:
		return baseURL + "/service-unavailable"
	default:
		return baseURL + "/unknown-error"
	}
}

// MapDomainError maps domain-level errors to HTTP status codes and problem details.
// This provides a central place to convert internal errors to HTTP responses.
//
// Returns: (status code, title, detail)
//
// Usage:
//
//	err := userRepo.FindByID(ctx, userID)
//	if err != nil {
//	    status, title, detail := middleware.MapDomainError(err)
//	    middleware.WriteError(w, r, status, title, detail)
//	    return
//	}
//
//nolint:cyclop // Comprehensive error mapping requires checking all domain error types and HTTP status codes
func MapDomainError(err error) (int, string, string) {
	// Define common error types (should be imported from domain layer)
	// For now, using error message matching as a fallback approach

	errMsg := err.Error()

	// Authentication errors (401)
	if errors.Is(err, ErrUnauthorized) ||
		containsAny(errMsg, "unauthorized", "invalid credentials", "authentication failed") {
		return http.StatusUnauthorized, "Unauthorized", "Invalid or missing authentication credentials"
	}

	// Authorization errors (403)
	if errors.Is(err, ErrForbidden) ||
		containsAny(errMsg, "forbidden", "insufficient permissions", "access denied") {
		return http.StatusForbidden, "Forbidden", "You do not have permission to access this resource"
	}

	// Not found errors (404)
	if errors.Is(err, ErrNotFound) || containsAny(errMsg, "not found", "does not exist") {
		return http.StatusNotFound, "Not Found", "The requested resource was not found"
	}

	// Conflict errors (409) - duplicate entries, optimistic lock failures
	if errors.Is(err, ErrConflict) ||
		containsAny(errMsg, "already exists", "duplicate", "conflict", "concurrent modification") {
		return http.StatusConflict, "Conflict", "The request conflicts with the current state of the resource"
	}

	// Validation errors (400)
	if errors.Is(err, ErrValidation) || containsAny(errMsg, "validation failed", "invalid", "malformed") {
		return http.StatusBadRequest, "Validation Failed", err.Error()
	}

	// Rate limit errors (429)
	if errors.Is(err, ErrRateLimitExceeded) {
		return http.StatusTooManyRequests, "Rate Limit Exceeded", "Too many requests, please try again later"
	}

	// Default to internal server error (500)
	// DO NOT expose internal error details to client
	return http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred. Please try again later."
}

// containsAny checks if the string contains any of the substrings (case-insensitive).
func containsAny(s string, substrings ...string) bool {
	sLower := toLower(s)
	for _, substr := range substrings {
		if contains(sLower, toLower(substr)) {
			return true
		}
	}
	return false
}

// Simple string helpers to avoid importing strings package.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexString(s, substr) >= 0)
}

func indexString(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

// Predefined error sentinels for domain error mapping.
// These should ideally be imported from domain packages.
var (
	// ErrUnauthorized indicates authentication failure.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates authorization failure.
	ErrForbidden = errors.New("forbidden")

	// ErrNotFound indicates resource not found.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates resource conflict (duplicate, concurrent modification).
	ErrConflict = errors.New("conflict")

	// ErrValidation indicates validation failure.
	ErrValidation = errors.New("validation failed")

	// ErrRateLimitExceeded indicates rate limit exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
