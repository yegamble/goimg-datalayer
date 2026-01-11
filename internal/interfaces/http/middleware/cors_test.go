package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCORSConfig(t *testing.T) {
	t.Parallel()

	// Act
	cfg := DefaultCORSConfig()

	// Assert
	assert.NotEmpty(t, cfg.AllowedOrigins)
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:3000")
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:5173")
	assert.Contains(t, cfg.AllowedOrigins, "https://app.goimg.dev")

	assert.Contains(t, cfg.AllowedMethods, http.MethodGet)
	assert.Contains(t, cfg.AllowedMethods, http.MethodPost)
	assert.Contains(t, cfg.AllowedMethods, http.MethodPut)
	assert.Contains(t, cfg.AllowedMethods, http.MethodPatch)
	assert.Contains(t, cfg.AllowedMethods, http.MethodDelete)
	assert.Contains(t, cfg.AllowedMethods, http.MethodOptions)

	assert.Contains(t, cfg.AllowedHeaders, "Authorization")
	assert.Contains(t, cfg.AllowedHeaders, "Content-Type")

	assert.Contains(t, cfg.ExposedHeaders, "X-Request-ID")
	assert.Contains(t, cfg.ExposedHeaders, "X-RateLimit-Limit")

	assert.True(t, cfg.AllowCredentials)
	assert.Equal(t, defaultCORSMaxAge, cfg.MaxAge)
}

func TestDevelopmentCORSConfig(t *testing.T) {
	t.Parallel()

	// Act
	cfg := DevelopmentCORSConfig()

	// Assert
	assert.Equal(t, []string{"*"}, cfg.AllowedOrigins)
	assert.Contains(t, cfg.AllowedMethods, http.MethodGet)
	assert.Contains(t, cfg.AllowedMethods, http.MethodHead)
	assert.Equal(t, []string{"*"}, cfg.AllowedHeaders)
	assert.False(t, cfg.AllowCredentials, "credentials should be false with wildcard origin")
}

func TestCORS_AllowedOrigin(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightRequest(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	// The go-chi/cors library returns 200 for preflight requests
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	// The library returns the requested method, not all configured methods
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestCORS_WildcardOrigin(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false, // Must be false with wildcard
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PanicsWithInvalidConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: true, // Invalid: cannot be true with wildcard
		MaxAge:           3600,
	}

	// Act & Assert
	assert.Panics(t, func() {
		CORS(cfg)
	}, "should panic when AllowCredentials=true with wildcard origin")
}

func TestContainsWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		origins  []string
		expected bool
	}{
		{
			name:     "contains wildcard",
			origins:  []string{"*"},
			expected: true,
		},
		{
			name:     "contains wildcard with spaces",
			origins:  []string{" * "},
			expected: true,
		},
		{
			name:     "contains wildcard among others",
			origins:  []string{"https://example.com", "*"},
			expected: true,
		},
		{
			name:     "no wildcard",
			origins:  []string{"https://example.com", "https://test.com"},
			expected: false,
		},
		{
			name:     "empty list",
			origins:  []string{},
			expected: false,
		},
		{
			name:     "wildcard-like but not exact",
			origins:  []string{"*.example.com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := containsWildcard(tt.origins)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{
			name:           "exact match",
			origin:         "https://example.com",
			allowedOrigins: []string{"https://example.com"},
			expected:       true,
		},
		{
			name:           "case insensitive match",
			origin:         "https://Example.COM",
			allowedOrigins: []string{"https://example.com"},
			expected:       true,
		},
		{
			name:           "with trailing spaces",
			origin:         "https://example.com ",
			allowedOrigins: []string{"https://example.com"},
			expected:       true,
		},
		{
			name:           "not in allowed list",
			origin:         "https://malicious.com",
			allowedOrigins: []string{"https://example.com"},
			expected:       false,
		},
		{
			name:           "wildcard allows all",
			origin:         "https://any-origin.com",
			allowedOrigins: []string{"*"},
			expected:       true,
		},
		{
			name:           "multiple allowed origins",
			origin:         "https://app.example.com",
			allowedOrigins: []string{"https://example.com", "https://app.example.com", "https://admin.example.com"},
			expected:       true,
		},
		{
			name:           "empty origin",
			origin:         "",
			allowedOrigins: []string{"https://example.com"},
			expected:       false,
		},
		{
			name:           "empty allowed list",
			origin:         "https://example.com",
			allowedOrigins: []string{},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := IsOriginAllowed(tt.origin, tt.allowedOrigins)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCORS_WithCredentials(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_ExposedHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit"},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	exposedHeaders := rec.Header().Get("Access-Control-Expose-Headers")
	// The go-chi/cors library normalizes header names
	assert.NotEmpty(t, exposedHeaders, "Exposed headers should be set")
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://malicious.com")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	// The go-chi/cors library should not set CORS headers for disallowed origins
	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_MaxAge(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           7200, // 2 hours
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, "7200", rec.Header().Get("Access-Control-Max-Age"))
}
