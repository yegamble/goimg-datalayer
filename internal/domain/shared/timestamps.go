package shared

import (
	"fmt"
	"time"
)

// Now returns the current time in UTC.
// Use this instead of time.Now() to ensure all timestamps are UTC.
func Now() time.Time {
	return time.Now().UTC()
}

// ParseISO8601 parses an ISO 8601 formatted timestamp string.
// Supported formats: RFC3339, RFC3339Nano.
// Returns an error if the string cannot be parsed.
func ParseISO8601(s string) (time.Time, error) {
	// Try RFC3339 first (most common)
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t.UTC(), nil
	}

	// Try RFC3339Nano for higher precision
	t, err = time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("parsing ISO8601 timestamp: %w", err)
}

// FormatISO8601 formats a time as an ISO 8601 string (RFC3339).
// The timestamp is converted to UTC before formatting.
func FormatISO8601(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
