package identity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeviceFingerprint(t *testing.T) {
	t.Parallel()

	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	device := NewDeviceFingerprint(ipAddress, userAgent)

	assert.Equal(t, ipAddress, device.IPAddress())
	assert.Equal(t, userAgent, device.UserAgent())
	assert.NotEmpty(t, device.FingerprintHash())
	assert.Equal(t, "Mac", device.DeviceName())
	assert.False(t, device.IsTrusted())
	assert.True(t, device.IsUnusual())
	assert.False(t, device.FirstSeenAt().IsZero())
	assert.False(t, device.LastSeenAt().IsZero())
	assert.Equal(t, device.FirstSeenAt(), device.LastSeenAt())
}

func TestReconstructDeviceFingerprint(t *testing.T) {
	t.Parallel()

	fingerprintHash := "abc123hash"
	ipAddress := "10.0.0.1"
	userAgent := "test-agent"
	deviceName := "Test Device"
	trusted := true
	firstSeenAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	lastSeenAt := time.Date(2023, 1, 15, 14, 30, 0, 0, time.UTC)

	device := ReconstructDeviceFingerprint(
		fingerprintHash,
		ipAddress,
		userAgent,
		deviceName,
		trusted,
		firstSeenAt,
		lastSeenAt,
	)

	assert.Equal(t, fingerprintHash, device.FingerprintHash())
	assert.Equal(t, ipAddress, device.IPAddress())
	assert.Equal(t, userAgent, device.UserAgent())
	assert.Equal(t, deviceName, device.DeviceName())
	assert.Equal(t, trusted, device.IsTrusted())
	assert.Equal(t, firstSeenAt, device.FirstSeenAt())
	assert.Equal(t, lastSeenAt, device.LastSeenAt())
}

func TestDeviceFingerprint_FingerprintHash(t *testing.T) {
	t.Parallel()

	t.Run("same IP and user agent produce same hash", func(t *testing.T) {
		t.Parallel()

		device1 := NewDeviceFingerprint("192.168.1.1", "user-agent")
		device2 := NewDeviceFingerprint("192.168.1.1", "user-agent")

		assert.Equal(t, device1.FingerprintHash(), device2.FingerprintHash())
	})

	t.Run("different IP produces different hash", func(t *testing.T) {
		t.Parallel()

		device1 := NewDeviceFingerprint("192.168.1.1", "user-agent")
		device2 := NewDeviceFingerprint("192.168.1.2", "user-agent")

		assert.NotEqual(t, device1.FingerprintHash(), device2.FingerprintHash())
	})

	t.Run("different user agent produces different hash", func(t *testing.T) {
		t.Parallel()

		device1 := NewDeviceFingerprint("192.168.1.1", "user-agent-1")
		device2 := NewDeviceFingerprint("192.168.1.1", "user-agent-2")

		assert.NotEqual(t, device1.FingerprintHash(), device2.FingerprintHash())
	})

	t.Run("hash is SHA-256 hex string", func(t *testing.T) {
		t.Parallel()

		device := NewDeviceFingerprint("192.168.1.1", "user-agent")
		hash := device.FingerprintHash()

		// SHA-256 produces 64 hex characters
		assert.Len(t, hash, 64)
		// Should only contain hex characters
		for _, ch := range hash {
			assert.True(t,
				(ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
				"hash should only contain hex characters")
		}
	})
}

func TestDeviceFingerprint_DeviceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		userAgent    string
		expectedName string
	}{
		{
			name:         "iPhone detection",
			userAgent:    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X)",
			expectedName: "iPhone",
		},
		{
			name:         "iPad detection",
			userAgent:    "Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X)",
			expectedName: "iPad",
		},
		{
			name:         "Android phone detection",
			userAgent:    "Mozilla/5.0 (Linux; Android 11; SM-G991B) Mobile",
			expectedName: "Android Phone",
		},
		{
			name:         "Android tablet detection",
			userAgent:    "Mozilla/5.0 (Linux; Android 11; SM-T870)",
			expectedName: "Android Tablet",
		},
		{
			name:         "Windows PC detection",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expectedName: "Windows PC",
		},
		{
			name:         "Mac detection",
			userAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			expectedName: "Mac",
		},
		{
			name:         "Linux PC detection",
			userAgent:    "Mozilla/5.0 (X11; Linux x86_64)",
			expectedName: "Linux PC",
		},
		{
			name:         "Chrome browser detection",
			userAgent:    "Chrome/91.0.4472.124",
			expectedName: "Chrome Browser",
		},
		{
			name:         "Firefox browser detection",
			userAgent:    "Firefox/89.0",
			expectedName: "Firefox Browser",
		},
		{
			name:         "Safari browser detection",
			userAgent:    "Safari/14.1.1",
			expectedName: "Safari Browser",
		},
		{
			name:         "Unknown device",
			userAgent:    "CustomBot/1.0",
			expectedName: "Unknown Device",
		},
		{
			name:         "Empty user agent",
			userAgent:    "",
			expectedName: "Unknown Device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			device := NewDeviceFingerprint("192.168.1.1", tt.userAgent)
			assert.Equal(t, tt.expectedName, device.DeviceName())
		})
	}
}

func TestDeviceFingerprint_IsTrusted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trusted bool
	}{
		{
			name:    "trusted device",
			trusted: true,
		},
		{
			name:    "untrusted device",
			trusted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			device := ReconstructDeviceFingerprint(
				"hash",
				"192.168.1.1",
				"user-agent",
				"Device",
				tt.trusted,
				time.Now(),
				time.Now(),
			)

			assert.Equal(t, tt.trusted, device.IsTrusted())
		})
	}
}

func TestDeviceFingerprint_IsUnusual(t *testing.T) {
	t.Parallel()

	t.Run("trusted device is not unusual", func(t *testing.T) {
		t.Parallel()

		device := ReconstructDeviceFingerprint(
			"hash", "ip", "ua", "name", true, time.Now(), time.Now(),
		)

		assert.False(t, device.IsUnusual())
	})

	t.Run("untrusted device is unusual", func(t *testing.T) {
		t.Parallel()

		device := ReconstructDeviceFingerprint(
			"hash", "ip", "ua", "name", false, time.Now(), time.Now(),
		)

		assert.True(t, device.IsUnusual())
	})
}

func TestDeviceFingerprint_MarkTrusted(t *testing.T) {
	t.Parallel()

	device := NewDeviceFingerprint("192.168.1.1", "user-agent")
	assert.False(t, device.IsTrusted())
	assert.True(t, device.IsUnusual())

	device.MarkTrusted()

	assert.True(t, device.IsTrusted())
	assert.False(t, device.IsUnusual())
}

func TestDeviceFingerprint_UpdateLastSeen(t *testing.T) {
	t.Parallel()

	device := NewDeviceFingerprint("192.168.1.1", "user-agent")
	originalLastSeen := device.LastSeenAt()

	// Small delay to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	before := time.Now().UTC()
	device.UpdateLastSeen()
	after := time.Now().UTC()

	newLastSeen := device.LastSeenAt()
	assert.True(t, newLastSeen.After(originalLastSeen))
	assert.True(t, newLastSeen.After(before.Add(-time.Second)))
	assert.True(t, newLastSeen.Before(after.Add(time.Second)))
}

func TestDeviceFingerprint_Matches(t *testing.T) {
	t.Parallel()

	device1 := NewDeviceFingerprint("192.168.1.1", "user-agent")
	device2 := NewDeviceFingerprint("192.168.1.1", "user-agent")
	device3 := NewDeviceFingerprint("192.168.1.2", "user-agent")

	t.Run("same fingerprint matches", func(t *testing.T) {
		assert.True(t, device1.Matches(device2))
		assert.True(t, device2.Matches(device1))
	})

	t.Run("different fingerprint does not match", func(t *testing.T) {
		assert.False(t, device1.Matches(device3))
		assert.False(t, device3.Matches(device1))
	})
}

func TestDeviceFingerprint_MatchesHash(t *testing.T) {
	t.Parallel()

	device := NewDeviceFingerprint("192.168.1.1", "user-agent")
	hash := device.FingerprintHash()

	t.Run("matching hash returns true", func(t *testing.T) {
		assert.True(t, device.MatchesHash(hash))
	})

	t.Run("non-matching hash returns false", func(t *testing.T) {
		assert.False(t, device.MatchesHash("different-hash"))
	})

	t.Run("empty hash returns false", func(t *testing.T) {
		assert.False(t, device.MatchesHash(""))
	})
}

func TestDeviceList_FindByHash(t *testing.T) {
	t.Parallel()

	device1 := NewDeviceFingerprint("192.168.1.1", "user-agent-1")
	device2 := NewDeviceFingerprint("192.168.1.2", "user-agent-2")
	device3 := NewDeviceFingerprint("192.168.1.3", "user-agent-3")

	list := DeviceList{device1, device2, device3}

	t.Run("finds existing device", func(t *testing.T) {
		found := list.FindByHash(device2.FingerprintHash())
		require.NotNil(t, found)
		assert.Equal(t, device2.FingerprintHash(), found.FingerprintHash())
	})

	t.Run("returns nil for non-existent hash", func(t *testing.T) {
		found := list.FindByHash("non-existent-hash")
		assert.Nil(t, found)
	})

	t.Run("returns nil for empty list", func(t *testing.T) {
		emptyList := DeviceList{}
		found := emptyList.FindByHash(device1.FingerprintHash())
		assert.Nil(t, found)
	})
}

func TestDeviceList_HasDevice(t *testing.T) {
	t.Parallel()

	device1 := NewDeviceFingerprint("192.168.1.1", "user-agent-1")
	device2 := NewDeviceFingerprint("192.168.1.2", "user-agent-2")

	list := DeviceList{device1}

	t.Run("returns true for existing device", func(t *testing.T) {
		assert.True(t, list.HasDevice(device1.FingerprintHash()))
	})

	t.Run("returns false for non-existent device", func(t *testing.T) {
		assert.False(t, list.HasDevice(device2.FingerprintHash()))
	})

	t.Run("returns false for empty hash", func(t *testing.T) {
		assert.False(t, list.HasDevice(""))
	})
}

func TestDeviceList_TrustedDevices(t *testing.T) {
	t.Parallel()

	device1 := NewDeviceFingerprint("192.168.1.1", "user-agent-1")
	device2 := NewDeviceFingerprint("192.168.1.2", "user-agent-2")
	device3 := NewDeviceFingerprint("192.168.1.3", "user-agent-3")

	device1.MarkTrusted()
	device3.MarkTrusted()

	list := DeviceList{device1, device2, device3}

	t.Run("filters only trusted devices", func(t *testing.T) {
		trusted := list.TrustedDevices()

		assert.Len(t, trusted, 2)
		for _, device := range trusted {
			assert.True(t, device.IsTrusted())
		}
	})

	t.Run("returns empty list when no trusted devices", func(t *testing.T) {
		untrustedList := DeviceList{device2}
		trusted := untrustedList.TrustedDevices()

		assert.Len(t, trusted, 0)
	})

	t.Run("returns all devices when all are trusted", func(t *testing.T) {
		device2.MarkTrusted()
		allTrustedList := DeviceList{device1, device2, device3}
		trusted := allTrustedList.TrustedDevices()

		assert.Len(t, trusted, 3)
	})

	t.Run("returns empty list for empty device list", func(t *testing.T) {
		emptyList := DeviceList{}
		trusted := emptyList.TrustedDevices()

		assert.Len(t, trusted, 0)
	})
}

func TestDeviceFingerprint_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full device tracking lifecycle", func(t *testing.T) {
		t.Parallel()

		// User logs in from new device
		ipAddress := "203.0.113.42"
		userAgent := "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X)"

		device := NewDeviceFingerprint(ipAddress, userAgent)

		// Verify initial state
		assert.Equal(t, "iPhone", device.DeviceName())
		assert.False(t, device.IsTrusted())
		assert.True(t, device.IsUnusual())

		// Simulate persistence: store and reload
		hash := device.FingerprintHash()
		stored := ReconstructDeviceFingerprint(
			hash,
			device.IPAddress(),
			device.UserAgent(),
			device.DeviceName(),
			device.IsTrusted(),
			device.FirstSeenAt(),
			device.LastSeenAt(),
		)

		// User trusts this device
		stored.MarkTrusted()
		assert.True(t, stored.IsTrusted())
		assert.False(t, stored.IsUnusual())

		// User logs in again from same device
		newLogin := NewDeviceFingerprint(ipAddress, userAgent)
		assert.Equal(t, hash, newLogin.FingerprintHash())

		// Update last seen
		originalLastSeen := stored.LastSeenAt()
		time.Sleep(10 * time.Millisecond)
		stored.UpdateLastSeen()
		assert.True(t, stored.LastSeenAt().After(originalLastSeen))

		// Add to device list
		list := DeviceList{stored}
		assert.True(t, list.HasDevice(hash))

		found := list.FindByHash(hash)
		require.NotNil(t, found)
		assert.True(t, found.IsTrusted())

		trusted := list.TrustedDevices()
		assert.Len(t, trusted, 1)
	})
}

// Helper functions tests
func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "substring found",
			s:      "Mozilla/5.0 (iPhone; CPU)",
			substr: "iPhone",
			want:   true,
		},
		{
			name:   "substring not found",
			s:      "Mozilla/5.0 (Android; CPU)",
			substr: "iPhone",
			want:   false,
		},
		{
			name:   "exact match",
			s:      "iPhone",
			substr: "iPhone",
			want:   true,
		},
		{
			name:   "empty substring",
			s:      "test",
			substr: "",
			want:   true,
		},
		{
			name:   "substring longer than string",
			s:      "test",
			substr: "testing",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := contains(tt.s, tt.substr)
			assert.Equal(t, strings.Contains(tt.s, tt.substr), got,
				"contains() should match strings.Contains()")
			assert.Equal(t, tt.want, got)
		})
	}
}
