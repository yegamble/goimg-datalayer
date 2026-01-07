package notification

// NotificationType represents the type of notification.
// Different types have different behaviors and templates.
type NotificationType string

const (
	// User notifications
	TypeNewFollower      NotificationType = "new_follower"
	TypeNewPhotos        NotificationType = "new_photos"
	TypeAccountSuspended NotificationType = "account_suspended"
	TypeAccountBanned    NotificationType = "account_banned"
	TypeAccountReinstated NotificationType = "account_reinstated"

	// Admin/Moderator notifications
	TypeAbuseReport       NotificationType = "abuse_report"
	TypeReportEscalated   NotificationType = "report_escalated"
	TypeModActionRequired NotificationType = "mod_action_required"
)

// RequiresEmail returns true for notifications that always send email
// regardless of user preferences (account status changes).
func (t NotificationType) RequiresEmail() bool {
	switch t {
	case TypeAccountSuspended, TypeAccountBanned, TypeAccountReinstated:
		return true
	default:
		return false
	}
}

// IsAdminOnly returns true for admin/moderator-only notifications.
func (t NotificationType) IsAdminOnly() bool {
	switch t {
	case TypeAbuseReport, TypeReportEscalated, TypeModActionRequired:
		return true
	default:
		return false
	}
}

// IsValid returns true if the notification type is recognized.
func (t NotificationType) IsValid() bool {
	switch t {
	case TypeNewFollower, TypeNewPhotos,
		TypeAccountSuspended, TypeAccountBanned, TypeAccountReinstated,
		TypeAbuseReport, TypeReportEscalated, TypeModActionRequired:
		return true
	default:
		return false
	}
}

// String returns the string representation of the notification type.
func (t NotificationType) String() string {
	return string(t)
}
