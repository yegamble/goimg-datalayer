package middleware

import (
	"fmt"
	"net/http"
)

// SecurityHeadersConfig holds configuration for security headers middleware.
type SecurityHeadersConfig struct {
	// EnableHSTS enables HTTP Strict Transport Security header (production only)
	EnableHSTS bool
	// HSTSMaxAge is the max-age value for HSTS header (default: 31536000 = 1 year)
	HSTSMaxAge int
	// HSTSIncludeSubDomains adds includeSubDomains directive to HSTS
	HSTSIncludeSubDomains bool
	// HSTSPreload adds preload directive to HSTS (only if on preload list)
	HSTSPreload bool
	// CSPDirectives allows customization of Content-Security-Policy header
	CSPDirectives string
	// FrameOptions sets X-Frame-Options header (DENY, SAMEORIGIN, or ALLOW-FROM uri)
	FrameOptions string
}

// DefaultSecurityHeadersConfig returns secure defaults for security headers.
func DefaultSecurityHeadersConfig(isProd bool) SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableHSTS:            isProd, // Only enable HSTS in production
		HSTSMaxAge:            31536000,
		HSTSIncludeSubDomains: true,
		HSTSPreload:           false, // Only enable after submitting to preload list
		CSPDirectives:         "default-src 'self'; img-src 'self' data: https:; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'",
		FrameOptions:          "DENY",
	}
}

// SecurityHeaders is a middleware that sets security-related HTTP headers on all responses.
//
// Headers set:
//   - X-Content-Type-Options: nosniff (prevents MIME-type sniffing)
//   - X-Frame-Options: DENY (prevents clickjacking via iframe)
//   - X-XSS-Protection: 1; mode=block (legacy XSS protection)
//   - Referrer-Policy: strict-origin-when-cross-origin (controls referrer leakage)
//   - Content-Security-Policy: restrictive policy (prevents XSS and injection attacks)
//   - Strict-Transport-Security: enforces HTTPS (production only)
//   - Permissions-Policy: restricts dangerous browser features
//
// Usage:
//
//	r := chi.NewRouter()
//	cfg := middleware.DefaultSecurityHeadersConfig(isProd)
//	r.Use(middleware.SecurityHeaders(cfg))
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Content-Type-Options: Prevents MIME-type sniffing attacks
			// Browsers will not try to guess content type, respecting Content-Type header
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// X-Frame-Options: Prevents clickjacking by disabling iframe embedding
			// DENY = cannot be embedded in any frame
			// SAMEORIGIN = can only be embedded in same origin
			w.Header().Set("X-Frame-Options", cfg.FrameOptions)

			// X-XSS-Protection: Legacy header for older browsers
			// Modern browsers use CSP instead, but this provides defense-in-depth
			// 1; mode=block = enable XSS filter and block page if attack detected
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer-Policy: Controls how much referrer information is sent
			// strict-origin-when-cross-origin = send full URL on same-origin, origin only on cross-origin HTTPS
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content-Security-Policy: Primary defense against XSS and injection attacks
			// Restricts sources of content (scripts, styles, images, etc.)
			// frame-ancestors 'none' = same as X-Frame-Options: DENY
			w.Header().Set("Content-Security-Policy", cfg.CSPDirectives)

			// Strict-Transport-Security (HSTS): Forces HTTPS for specified duration
			// WARNING: Only enable in production with working HTTPS
			// includeSubDomains = applies to all subdomains
			// preload = eligible for browser HSTS preload list (irreversible)
			if cfg.EnableHSTS {
				hstsValue := buildHSTSHeader(cfg)
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// Permissions-Policy: Restricts access to browser features and APIs
			// Disables dangerous features like geolocation, camera, microphone
			// () = feature disabled for all origins (including same-origin)
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

			next.ServeHTTP(w, r)
		})
	}
}

// buildHSTSHeader constructs the HSTS header value from configuration.
func buildHSTSHeader(cfg SecurityHeadersConfig) string {
	hsts := ""

	// max-age directive (required)
	hsts += fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)

	// includeSubDomains directive (optional)
	if cfg.HSTSIncludeSubDomains {
		hsts += "; includeSubDomains"
	}

	// preload directive (optional, only use if submitted to preload list)
	if cfg.HSTSPreload {
		hsts += "; preload"
	}

	return hsts
}
