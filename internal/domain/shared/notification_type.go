package shared

// NotificationType represents the type of notification.
// Different types have different behaviors and templates.
//
// This is defined in the shared kernel because it's used by both
// the identity and notification bounded contexts.
type NotificationType string

const (
	// User notifications
	NotificationTypeNewFollower       NotificationType = "new_follower"
	NotificationTypeNewPhotos         NotificationType = "new_photos"
	NotificationTypeAccountSuspended  NotificationType = "account_suspended"
	NotificationTypeAccountBanned     NotificationType = "account_banned"
	NotificationTypeAccountReinstated NotificationType = "account_reinstated"
	NotificationTypeMalwareDetected   NotificationType = "malware_detected"

	// Admin/Moderator notifications
	NotificationTypeAbuseReport       NotificationType = "abuse_report"
	NotificationTypeReportEscalated   NotificationType = "report_escalated"
	NotificationTypeModActionRequired NotificationType = "mod_action_required"
)

// RequiresEmail returns true for notifications that always send email
// regardless of user preferences (account status changes).
func (t NotificationType) RequiresEmail() bool {
	switch t {
	case NotificationTypeAccountSuspended, NotificationTypeAccountBanned, NotificationTypeAccountReinstated, NotificationTypeMalwareDetected:
		return true
	default:
		return false
	}
}

// IsAdminOnly returns true for admin/moderator-only notifications.
func (t NotificationType) IsAdminOnly() bool {
	switch t {
	case NotificationTypeAbuseReport, NotificationTypeReportEscalated, NotificationTypeModActionRequired:
		return true
	default:
		return false
	}
}

// IsValid returns true if the notification type is recognized.
func (t NotificationType) IsValid() bool {
	switch t {
	case NotificationTypeNewFollower, NotificationTypeNewPhotos,
		NotificationTypeAccountSuspended, NotificationTypeAccountBanned, NotificationTypeAccountReinstated,
		NotificationTypeAbuseReport, NotificationTypeReportEscalated, NotificationTypeModActionRequired,
		NotificationTypeMalwareDetected:
		return true
	default:
		return false
	}
}

// String returns the string representation of the notification type.
func (t NotificationType) String() string {
	return string(t)
}
