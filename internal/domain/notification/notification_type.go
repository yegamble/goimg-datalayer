package notification

import "github.com/yegamble/goimg-datalayer/internal/domain/shared"

// NotificationType is an alias for the shared kernel type.
// This allows the notification package to use the type without breaking
// existing code that imports from here.
type NotificationType = shared.NotificationType

// Re-export constants for convenience
const (
	TypeNewFollower       = shared.NotificationTypeNewFollower
	TypeNewPhotos         = shared.NotificationTypeNewPhotos
	TypeAccountSuspended  = shared.NotificationTypeAccountSuspended
	TypeAccountBanned     = shared.NotificationTypeAccountBanned
	TypeAccountReinstated = shared.NotificationTypeAccountReinstated
	TypeMalwareDetected   = shared.NotificationTypeMalwareDetected

	TypeAbuseReport       = shared.NotificationTypeAbuseReport
	TypeReportEscalated   = shared.NotificationTypeReportEscalated
	TypeModActionRequired = shared.NotificationTypeModActionRequired
)
