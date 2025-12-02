// Package moderation implements the Moderation bounded context for content review and user enforcement.
//
// This package contains the domain layer of the Moderation context, following Domain-Driven Design principles.
// It is responsible for managing content reports, user bans, and moderation review workflows.
//
// # Core Components
//
// Value Objects:
//   - ReportID: Unique identifier for content reports (UUID-based)
//   - BanID: Unique identifier for user bans (UUID-based)
//   - ReviewID: Unique identifier for moderation reviews (UUID-based)
//   - ReportReason: Enumeration of report reasons (spam, inappropriate, copyright, harassment, other)
//   - ReportStatus: Report workflow status (pending, reviewing, resolved, dismissed)
//   - ReviewAction: Action taken by moderator (dismiss, warn, remove, ban)
//
// Entities:
//   - Report: Aggregate root representing a content report with workflow state
//   - Ban: Aggregate root representing a user ban (temporary or permanent)
//   - Review: Entity representing a moderation review decision (audit trail)
//
// Repository:
//   - ReportRepository: Interface for persisting Report aggregates
//   - BanRepository: Interface for persisting Ban aggregates
//   - ReviewRepository: Interface for persisting Review entities
//
// Domain Events:
//   - ReportCreated: Emitted when a new report is submitted
//   - ReportReviewStarted: Emitted when a moderator starts reviewing a report
//   - ReportResolved: Emitted when a report is resolved with action taken
//   - ReportDismissed: Emitted when a report is dismissed as invalid
//   - UserBanned: Emitted when a user is banned
//   - BanRevoked: Emitted when a ban is manually revoked
//   - BanExpired: Emitted when a temporary ban expires
//
// # Design Principles
//
//  1. No Infrastructure Dependencies: This package only imports standard library, shared domain components,
//     and other bounded contexts (identity, gallery) for cross-context references.
//
//  2. Immutable Value Objects: All value objects are immutable after creation and validate their invariants
//     in factory functions.
//
//  3. Aggregate Roots: Report and Ban are aggregate roots that enforce all invariants and business rules.
//
//  4. Domain Events: State changes emit domain events for eventual consistency and integration with other
//     bounded contexts (e.g., suspending users, hiding images).
//
//  5. State Machine Enforcement: Report status transitions follow a defined state machine with validation.
//     Terminal states (resolved, dismissed) cannot be re-opened.
//
//  6. Audit Trail: All moderation actions are recorded via Review entities for accountability and compliance.
//
// # Business Rules
//
//  1. Report Lifecycle:
//     - pending -> reviewing (when moderator starts review)
//     - reviewing -> resolved (when action is taken)
//     - reviewing -> dismissed (when report is invalid)
//     - Terminal states (resolved, dismissed) cannot transition to other states
//
//  2. Ban Rules:
//     - Bans can be temporary (with expiration) or permanent (no expiration)
//     - Active bans prevent user actions across the system
//     - Expired bans are automatically inactive (IsActive() = false)
//     - Bans can be manually revoked before expiration
//     - Revoked bans cannot be un-revoked
//
//  3. Self-Reporting Prevention:
//     - Users cannot report their own content (enforced at application layer)
//     - Document this constraint in Report godoc
//
//  4. Report Description:
//     - Maximum 1000 characters
//     - Required for "other" reason type
//
// # Usage Example
//
//	// Report an image
//	reporterID := identity.MustParseUserID("...")
//	imageID := gallery.MustParseImageID("...")
//	reason := moderation.ReasonInappropriate
//	description := "This image contains inappropriate content"
//
//	report, err := moderation.NewReport(reporterID, imageID, reason, description)
//	if err != nil {
//	    // Handle validation error
//	}
//
//	// Moderator starts review
//	err = report.StartReview()
//
//	// Resolve the report
//	moderatorID := identity.MustParseUserID("...")
//	err = report.Resolve(moderatorID, "Content removed per community guidelines")
//
//	// Create audit trail
//	review, err := moderation.NewReview(
//	    report.ID(),
//	    moderatorID,
//	    moderation.ActionRemove,
//	    "Violated community standards - section 3.2",
//	)
//
//	// Ban the user
//	duration := 7 * 24 * time.Hour // 7 days
//	ban, err := moderation.NewBan(
//	    offenderID,
//	    moderatorID,
//	    "Repeated violations of community guidelines",
//	    &duration,
//	)
//
//	// Check if user is banned
//	if ban.IsActive() {
//	    // Prevent user action
//	}
//
//	// Revoke ban early
//	err = ban.Revoke(moderatorID)
package moderation
