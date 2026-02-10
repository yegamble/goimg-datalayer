package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "uses first forwarded IP",
			remoteAddr: "127.0.0.1:8080",
			xff:        "203.0.113.10, 198.51.100.2",
			want:       "203.0.113.10",
		},
		{
			name:       "uses X-Real-IP when forwarded missing",
			remoteAddr: "127.0.0.1:8080",
			xRealIP:    "198.51.100.7",
			want:       "198.51.100.7",
		},
		{
			name:       "strips port from IPv4 remote addr",
			remoteAddr: "192.168.1.15:54321",
			want:       "192.168.1.15",
		},
		{
			name:       "strips port from IPv6 remote addr",
			remoteAddr: "[2001:db8::42]:54321",
			want:       "2001:db8::42",
		},
		{
			name:       "keeps host when remote addr has no port",
			remoteAddr: "10.0.0.99",
			want:       "10.0.0.99",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr

			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}

			if got := GetClientIP(r); got != tt.want {
				t.Fatalf("GetClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
