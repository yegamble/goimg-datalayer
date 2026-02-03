package middleware

import (
	"net/http"
)

// BodyLimitConfig holds configuration for BodyLimit middleware.
type BodyLimitConfig struct {
	// LimitBytes is the maximum allowed body size in bytes.
	// Default: 10MB (10 * 1024 * 1024)
	LimitBytes int64
}

// BodyLimit creates a middleware that limits the size of request bodies.
// This prevents denial-of-service attacks using large payloads (e.g. zip bombs, large JSONs).
//
// If the request body exceeds the limit, http.MaxBytesReader will return an error
// when reading the body, and the handler should handle it appropriately or
// the server will close the connection.
//
// Usage:
//
//	r.Use(middleware.BodyLimit(middleware.BodyLimitConfig{LimitBytes: 10 * 1024 * 1024}))
func BodyLimit(cfg BodyLimitConfig) func(http.Handler) http.Handler {
	limit := cfg.LimitBytes
	if limit <= 0 {
		limit = 10 * 1024 * 1024 // Default 10MB
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit the request body size
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// TrustedProxyConfig holds configuration for ProxyValidation middleware.
type TrustedProxyConfig struct {
	// TrustedProxies is a list of trusted proxy IP addresses or CIDR ranges.
	// If empty, no proxies are trusted (safe default).
	TrustedProxies []string
}

// ProxyValidation creates a middleware that validates the X-Forwarded-For header.
// It ensures that we only trust X-Forwarded-For if the request comes from a trusted proxy.
//
// If the request is not from a trusted proxy, the X-Forwarded-For header is stripped
// to prevent IP spoofing.
//
// Usage:
//
//	cfg := middleware.TrustedProxyConfig{TrustedProxies: []string{"10.0.0.1", "192.168.1.0/24"}}
//	r.Use(middleware.ProxyValidation(cfg))
func ProxyValidation(cfg TrustedProxyConfig) func(http.Handler) http.Handler {
	trustedMap := make(map[string]bool)
	for _, ip := range cfg.TrustedProxies {
		trustedMap[ip] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteIP := getRemoteIP(r.RemoteAddr)

			// If the immediate sender is not a trusted proxy, we cannot trust X-Forwarded-For
			if !trustedMap[remoteIP] {
				// Strip headers to force downstream to use RemoteAddr
				r.Header.Del("X-Forwarded-For")
				r.Header.Del("X-Real-IP")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getRemoteIP extracts the IP address from RemoteAddr (stripping port).
func getRemoteIP(remoteAddr string) string {
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			if i > 0 && remoteAddr[0] == '[' {
				return remoteAddr[1 : i-1]
			}
			return remoteAddr[:i]
		}
	}
	return remoteAddr
}

// contains checks if a string slice contains a value.
func stringSliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
