// Package shared provides the Shared Kernel for the goimg-datalayer domain layer.
//
// # Shared Kernel in Domain-Driven Design
//
// The Shared Kernel is a bounded context that contains domain primitives and utilities
// that are shared across multiple bounded contexts. This package serves as the foundation
// that other domain contexts (identity, gallery, moderation) can safely import.
//
// # Design Principles
//
//   - Minimal dependencies: Only stdlib + google/uuid for ID generation
//   - Immutability: All value objects are immutable after construction
//   - UTC timestamps: All time operations use UTC
//   - No business logic: Only generic primitives and helpers
//   - High test coverage: >= 90% coverage required
//
// # Components
//
// Pagination:
//   - Immutable value object for pagination parameters
//   - Page is 1-indexed (first page = 1)
//   - PerPage constrained to 1-100, defaults to 20
//   - Provides offset/limit calculations for database queries
//   - Supports total count and page navigation helpers
//
// Domain Events:
//   - DomainEvent interface for event-driven architecture
//   - BaseEvent provides common fields (eventID, eventType, occurredAt, aggregateID)
//   - Each event has a unique UUID for idempotency
//   - All events track occurrence time in UTC
//
// Timestamps:
//   - Now() returns current UTC time
//   - ParseISO8601() parses RFC3339/RFC3339Nano timestamps
//   - FormatISO8601() formats to RFC3339 in UTC
//   - All timestamp operations guarantee UTC
//
// Common Errors:
//   - ErrNotFound: Resource not found
//   - ErrAlreadyExists: Resource already exists
//   - ErrInvalidInput: Invalid input validation
//   - ErrUnauthorized: Missing authentication
//   - ErrForbidden: Insufficient permissions
//
// # Usage Examples
//
// Pagination:
//
//	// Create pagination with validation
//	page, perPage := 2, 50
//	pagination, err := shared.NewPagination(page, perPage)
//	if err != nil {
//	    return err
//	}
//
//	// Use in repository query
//	offset := pagination.Offset()  // 50
//	limit := pagination.Limit()    // 50
//
//	// Set total after query
//	pagination = pagination.WithTotal(250)
//	hasNext := pagination.HasNext()        // true
//	totalPages := pagination.TotalPages()  // 5
//
// Domain Events:
//
//	// Define a concrete event
//	type UserRegistered struct {
//	    shared.BaseEvent
//	    Email    string
//	    Username string
//	}
//
//	// Create and emit event
//	event := UserRegistered{
//	    BaseEvent: shared.NewBaseEvent("user.registered", userID.String()),
//	    Email:     "user@example.com",
//	    Username:  "johndoe",
//	}
//
//	// Use event ID for idempotency
//	eventID := event.EventID()  // UUID string
//
// Timestamps:
//
//	// Always use Now() for UTC timestamps
//	createdAt := shared.Now()
//
//	// Parse ISO8601 timestamps from external sources
//	timestamp, err := shared.ParseISO8601("2023-12-01T15:04:05Z")
//	if err != nil {
//	    return err
//	}
//
//	// Format for API responses
//	formatted := shared.FormatISO8601(timestamp)
//
// Error Handling:
//
//	// Wrap shared errors with context
//	if user == nil {
//	    return fmt.Errorf("finding user %s: %w", userID, shared.ErrNotFound)
//	}
//
//	// Check error types
//	if errors.Is(err, shared.ErrNotFound) {
//	    return http.StatusNotFound
//	}
//
// # Cross-Context Usage
//
// This package can be safely imported by all bounded contexts:
//
//	import "github.com/yegamble/goimg-datalayer/internal/domain/shared"
//
// However, bounded contexts should NOT import each other directly.
// Use application layer orchestration or domain events for cross-context communication.
//
// # Testing
//
// All code in this package has >= 90% test coverage with comprehensive test cases
// covering edge cases, boundary conditions, and parallel execution safety.
package shared
