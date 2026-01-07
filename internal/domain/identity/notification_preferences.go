package identity

import "github.com/yegamble/goimg-datalayer/internal/domain/notification"

// DigestFrequency controls how often batched notifications are sent via email.
type DigestFrequency string

const (
	// DigestImmediate sends notifications immediately as they occur.
	DigestImmediate DigestFrequency = "immediate"

	// DigestHourly batches notifications and sends every hour.
	DigestHourly DigestFrequency = "hourly"

	// DigestDaily batches notifications and sends once per day.
	DigestDaily DigestFrequency = "daily"

	// DigestWeekly batches notifications and sends once per week.
	DigestWeekly DigestFrequency = "weekly"
)

// IsValid returns true if the digest frequency is recognized.
func (d DigestFrequency) IsValid() bool {
	switch d {
	case DigestImmediate, DigestHourly, DigestDaily, DigestWeekly:
		return true
	default:
		return false
	}
}

// String returns the string representation of the digest frequency.
func (d DigestFrequency) String() string {
	return string(d)
}

// NotificationPreferences is a value object that controls how a user receives notifications.
// It supports both in-app and email notification channels.
//
// Business Rules:
// - Email notifications are opt-in by default (emailEnabled = false)
// - Account status notifications always send email regardless of preferences
// - Per-type preferences override the global emailEnabled flag
// - Digest frequency controls email batching for non-urgent notifications
type NotificationPreferences struct {
	emailEnabled    bool                                   // Global email opt-in
	emailTypes      map[notification.NotificationType]bool // Per-type email opt-in/out
	digestFrequency DigestFrequency                        // Batching frequency for grouped notifications
}

// DefaultNotificationPreferences returns the default preferences for new users.
// Email is disabled by default (opt-in model for privacy).
func DefaultNotificationPreferences() NotificationPreferences {
	return NotificationPreferences{
		emailEnabled:    false, // Opt-in by default
		emailTypes:      make(map[notification.NotificationType]bool),
		digestFrequency: DigestDaily, // Default to daily digest
	}
}

// NewNotificationPreferences creates notification preferences with the given settings.
func NewNotificationPreferences(
	emailEnabled bool,
	emailTypes map[notification.NotificationType]bool,
	digestFrequency DigestFrequency,
) NotificationPreferences {
	if emailTypes == nil {
		emailTypes = make(map[notification.NotificationType]bool)
	}

	// Default to daily if invalid frequency
	if !digestFrequency.IsValid() {
		digestFrequency = DigestDaily
	}

	return NotificationPreferences{
		emailEnabled:    emailEnabled,
		emailTypes:      emailTypes,
		digestFrequency: digestFrequency,
	}
}

// EmailEnabled returns whether email notifications are globally enabled.
func (p NotificationPreferences) EmailEnabled() bool {
	return p.emailEnabled
}

// EmailTypes returns the per-type email preferences.
func (p NotificationPreferences) EmailTypes() map[notification.NotificationType]bool {
	// Return a copy to prevent external modification
	types := make(map[notification.NotificationType]bool, len(p.emailTypes))
	for k, v := range p.emailTypes {
		types[k] = v
	}
	return types
}

// DigestFrequency returns the digest batching frequency.
func (p NotificationPreferences) DigestFrequency() DigestFrequency {
	return p.digestFrequency
}

// ShouldEmail determines whether an email should be sent for a given notification type.
// This respects both global and per-type preferences.
//
// Logic:
// 1. Account status notifications always send email (TypeAccountSuspended, TypeAccountBanned, etc.)
// 2. If email globally disabled, return false (except for #1)
// 3. Check per-type preference - default to true if not explicitly set
func (p NotificationPreferences) ShouldEmail(notifType notification.NotificationType) bool {
	// Account status emails always send (ignore user preferences)
	if notifType.RequiresEmail() {
		return true
	}

	// Otherwise respect user preferences
	if !p.emailEnabled {
		return false
	}

	// Check per-type preference (default to true if email is globally enabled)
	enabled, exists := p.emailTypes[notifType]
	if !exists {
		return true // Default to enabled if not explicitly set
	}

	return enabled
}

// EnableEmail enables email notifications globally.
func (p *NotificationPreferences) EnableEmail() {
	p.emailEnabled = true
}

// DisableEmail disables email notifications globally.
// Note: Account status emails will still send.
func (p *NotificationPreferences) DisableEmail() {
	p.emailEnabled = false
}

// SetEmailTypePreference sets whether email should be sent for a specific notification type.
func (p *NotificationPreferences) SetEmailTypePreference(notifType notification.NotificationType, enabled bool) {
	if p.emailTypes == nil {
		p.emailTypes = make(map[notification.NotificationType]bool)
	}
	p.emailTypes[notifType] = enabled
}

// SetDigestFrequency updates the digest batching frequency.
func (p *NotificationPreferences) SetDigestFrequency(frequency DigestFrequency) {
	if frequency.IsValid() {
		p.digestFrequency = frequency
	}
}
