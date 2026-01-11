package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSecurityHeadersConfig_Production(t *testing.T) {
	t.Parallel()

	// Act
	cfg := DefaultSecurityHeadersConfig(true)

	// Assert
	assert.True(t, cfg.EnableHSTS)
	assert.Equal(t, hstsMaxAgeOneYear, cfg.HSTSMaxAge)
	assert.True(t, cfg.HSTSIncludeSubDomains)
	assert.False(t, cfg.HSTSPreload)
	assert.NotEmpty(t, cfg.CSPDirectives)
	assert.Equal(t, "DENY", cfg.FrameOptions)
}

func TestDefaultSecurityHeadersConfig_Development(t *testing.T) {
	t.Parallel()

	// Act
	cfg := DefaultSecurityHeadersConfig(false)

	// Assert
	assert.False(t, cfg.EnableHSTS, "HSTS should be disabled in development")
}

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := DefaultSecurityHeadersConfig(true)
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	headers := rec.Header()

	// X-Content-Type-Options
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))

	// X-Frame-Options
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))

	// X-XSS-Protection
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))

	// Referrer-Policy
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))

	// Content-Security-Policy
	csp := headers.Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")

	// Strict-Transport-Security (production only)
	hsts := headers.Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts)
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")

	// Permissions-Policy
	permissions := headers.Get("Permissions-Policy")
	assert.NotEmpty(t, permissions)
	assert.Contains(t, permissions, "geolocation=()")
	assert.Contains(t, permissions, "camera=()")
	assert.Contains(t, permissions, "microphone=()")
}

func TestSecurityHeaders_HSTS_DisabledInDevelopment(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := DefaultSecurityHeadersConfig(false)
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts, "HSTS should not be set in development")
}

func TestSecurityHeaders_HSTS_CustomMaxAge(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            300, // 5 minutes
		HSTSIncludeSubDomains: false,
		HSTSPreload:           false,
		CSPDirectives:         "default-src 'self'",
		FrameOptions:          "DENY",
	}

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Equal(t, "max-age=300", hsts)
}

func TestSecurityHeaders_HSTS_WithSubdomains(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubDomains: true,
		HSTSPreload:           false,
		CSPDirectives:         "default-src 'self'",
		FrameOptions:          "DENY",
	}

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestSecurityHeaders_HSTS_WithPreload(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubDomains: true,
		HSTSPreload:           true,
		CSPDirectives:         "default-src 'self'",
		FrameOptions:          "DENY",
	}

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	t.Parallel()

	// Arrange
	customCSP := "default-src 'none'; script-src 'self'; img-src 'self' https:"
	cfg := SecurityHeadersConfig{
		EnableHSTS:            false,
		HSTSMaxAge:            0,
		HSTSIncludeSubDomains: false,
		HSTSPreload:           false,
		CSPDirectives:         customCSP,
		FrameOptions:          "DENY",
	}

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Equal(t, customCSP, csp)
}

func TestSecurityHeaders_CustomFrameOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		frameOptions string
	}{
		{"DENY", "DENY"},
		{"SAMEORIGIN", "SAMEORIGIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			cfg := SecurityHeadersConfig{
				EnableHSTS:    false,
				CSPDirectives: "default-src 'self'",
				FrameOptions:  tt.frameOptions,
			}

			handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rec, req)

			// Assert
			frameOptions := rec.Header().Get("X-Frame-Options")
			assert.Equal(t, tt.frameOptions, frameOptions)
		})
	}
}

func TestSecurityHeaders_DoesNotOverrideHandlerHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := DefaultSecurityHeadersConfig(false)
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler sets its own Content-Type
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
}

func TestBuildHSTSHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      SecurityHeadersConfig
		expected string
	}{
		{
			name: "basic HSTS",
			cfg: SecurityHeadersConfig{
				HSTSMaxAge:            300,
				HSTSIncludeSubDomains: false,
				HSTSPreload:           false,
			},
			expected: "max-age=300",
		},
		{
			name: "HSTS with subdomains",
			cfg: SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           false,
			},
			expected: "max-age=31536000; includeSubDomains",
		},
		{
			name: "HSTS with subdomains and preload",
			cfg: SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name: "HSTS with preload only (without subdomains)",
			cfg: SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: false,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := buildHSTSHeader(tt.cfg)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}
