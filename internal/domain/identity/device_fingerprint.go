package identity

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DeviceFingerprint is a value object representing a known device for a user.
// Used to detect unusual logins from new devices/locations and trigger notifications.
type DeviceFingerprint struct {
	fingerprintHash string    // SHA-256 hash of device identifiers
	ipAddress       string    // IP address of the device
	userAgent       string    // User-Agent string from the browser/client
	deviceName      string    // Human-readable device name (derived from user agent)
	trusted         bool      // Whether this device is marked as trusted
	firstSeenAt     time.Time // First login from this device
	lastSeenAt      time.Time // Most recent login from this device
}

// NewDeviceFingerprint creates a device fingerprint from login metadata.
func NewDeviceFingerprint(ipAddress, userAgent string) DeviceFingerprint {
	now := time.Now().UTC()

	return DeviceFingerprint{
		fingerprintHash: generateFingerprintHash(ipAddress, userAgent),
		ipAddress:       ipAddress,
		userAgent:       userAgent,
		deviceName:      parseDeviceName(userAgent),
		trusted:         false, // New devices start as untrusted
		firstSeenAt:     now,
		lastSeenAt:      now,
	}
}

// ReconstructDeviceFingerprint reconstitutes a device fingerprint from storage.
// This should only be used by the repository layer when loading from the database.
func ReconstructDeviceFingerprint(
	fingerprintHash, ipAddress, userAgent, deviceName string,
	trusted bool,
	firstSeenAt, lastSeenAt time.Time,
) DeviceFingerprint {
	return DeviceFingerprint{
		fingerprintHash: fingerprintHash,
		ipAddress:       ipAddress,
		userAgent:       userAgent,
		deviceName:      deviceName,
		trusted:         trusted,
		firstSeenAt:     firstSeenAt,
		lastSeenAt:      lastSeenAt,
	}
}

// FingerprintHash returns the SHA-256 hash identifying this device.
func (d DeviceFingerprint) FingerprintHash() string {
	return d.fingerprintHash
}

// IPAddress returns the IP address associated with this device.
func (d DeviceFingerprint) IPAddress() string {
	return d.ipAddress
}

// UserAgent returns the User-Agent string from this device.
func (d DeviceFingerprint) UserAgent() string {
	return d.userAgent
}

// DeviceName returns a human-readable name for this device.
func (d DeviceFingerprint) DeviceName() string {
	return d.deviceName
}

// IsTrusted returns whether this device is marked as trusted.
func (d DeviceFingerprint) IsTrusted() bool {
	return d.trusted
}

// IsUnusual returns whether this is an unusual (untrusted) device.
func (d DeviceFingerprint) IsUnusual() bool {
	return !d.trusted
}

// FirstSeenAt returns when this device was first seen.
func (d DeviceFingerprint) FirstSeenAt() time.Time {
	return d.firstSeenAt
}

// LastSeenAt returns when this device was last seen.
func (d DeviceFingerprint) LastSeenAt() time.Time {
	return d.lastSeenAt
}

// MarkTrusted marks this device as trusted.
func (d *DeviceFingerprint) MarkTrusted() {
	d.trusted = true
}

// UpdateLastSeen updates the last seen timestamp.
func (d *DeviceFingerprint) UpdateLastSeen() {
	d.lastSeenAt = time.Now().UTC()
}

// Matches checks if another fingerprint matches this one (same hash).
func (d DeviceFingerprint) Matches(other DeviceFingerprint) bool {
	return subtle.ConstantTimeCompare([]byte(d.fingerprintHash), []byte(other.fingerprintHash)) == 1
}

// MatchesHash checks if the given hash matches this fingerprint.
func (d DeviceFingerprint) MatchesHash(hash string) bool {
	return subtle.ConstantTimeCompare([]byte(d.fingerprintHash), []byte(hash)) == 1
}

// generateFingerprintHash creates a SHA-256 hash from device identifiers.
// Uses IP + UserAgent as the fingerprint components.
func generateFingerprintHash(ipAddress, userAgent string) string {
	combined := fmt.Sprintf("%s|%s", ipAddress, userAgent)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// parseDeviceName extracts a human-readable device name from the user agent.
// This is a simplified implementation; production could use a user agent parsing library.
func parseDeviceName(userAgent string) string {
	if userAgent == "" {
		return "Unknown Device"
	}

	// Simple heuristics for common browsers/devices
	switch {
	case contains(userAgent, "iPhone"):
		return "iPhone"
	case contains(userAgent, "iPad"):
		return "iPad"
	case contains(userAgent, "Android"):
		if contains(userAgent, "Mobile") {
			return "Android Phone"
		}
		return "Android Tablet"
	case contains(userAgent, "Windows"):
		return "Windows PC"
	case contains(userAgent, "Macintosh"):
		return "Mac"
	case contains(userAgent, "Linux"):
		return "Linux PC"
	case contains(userAgent, "Chrome"):
		return "Chrome Browser"
	case contains(userAgent, "Firefox"):
		return "Firefox Browser"
	case contains(userAgent, "Safari"):
		return "Safari Browser"
	default:
		return "Unknown Device"
	}
}

// contains is a simple case-insensitive contains check.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// DeviceList is a collection of device fingerprints for a user.
type DeviceList []DeviceFingerprint

// FindByHash finds a device by its fingerprint hash.
func (dl DeviceList) FindByHash(hash string) *DeviceFingerprint {
	for i := range dl {
		if dl[i].MatchesHash(hash) {
			return &dl[i]
		}
	}
	return nil
}

// HasDevice checks if a device with the given hash exists.
func (dl DeviceList) HasDevice(hash string) bool {
	return dl.FindByHash(hash) != nil
}

// TrustedDevices returns only the trusted devices.
func (dl DeviceList) TrustedDevices() DeviceList {
	trusted := make(DeviceList, 0)
	for _, d := range dl {
		if d.IsTrusted() {
			trusted = append(trusted, d)
		}
	}
	return trusted
}
