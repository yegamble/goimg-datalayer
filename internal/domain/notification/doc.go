// Package notification defines the Notification bounded context.
// This context handles in-app notifications and email notification preferences.
//
// Key Aggregates:
//   - Notification: In-app notification entity
//
// Business Rules:
//   - All users receive internal notifications (always saved)
//   - Email notifications are opt-in by default
//   - Account status notifications (suspend/ban) always send email
//   - Notification preferences are stored on the User aggregate
package notification
