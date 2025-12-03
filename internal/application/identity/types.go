package identity

import "context"

// Command is a marker interface for write operations.
// Commands represent state-changing operations and follow the Command pattern.
// They encapsulate all information needed to perform an action.
type Command interface {
	// isCommand is an unexported method to enforce implementation
	isCommand()
}

// CommandHandler processes a command and returns a result.
// The generic type parameters are:
//   - C: The command type (must implement Command interface)
//   - R: The result type returned after processing
//
// Handlers should:
//   - Validate input and convert to domain value objects
//   - Enforce business rules via domain methods
//   - Coordinate persistence operations
//   - Publish domain events after successful save
type CommandHandler[C Command, R any] interface {
	Handle(ctx context.Context, cmd C) (R, error)
}

// Query is a marker interface for read operations.
// Queries represent read-only operations with no side effects.
// They encapsulate all parameters needed to retrieve data.
type Query interface {
	// isQuery is an unexported method to enforce implementation
	isQuery()
}

// QueryHandler processes a query and returns a result.
// The generic type parameters are:
//   - Q: The query type (must implement Query interface)
//   - R: The result type returned from the query
//
// Handlers should:
//   - Validate input parameters
//   - Convert to domain value objects
//   - Delegate to repository for data retrieval
//   - Never modify state
type QueryHandler[Q Query, R any] interface {
	Handle(ctx context.Context, q Q) (R, error)
}

// EventPublisher publishes domain events to the message bus.
// This interface is defined in the application layer and implemented
// in the infrastructure layer (e.g., using asynq, RabbitMQ, Kafka).
//
// Events should only be published AFTER successful persistence to ensure
// consistency. If publishing fails, the error should be logged but not
// cause the operation to fail.
type EventPublisher interface {
	// Publish sends a domain event to the message bus.
	// The event parameter should implement the shared.DomainEvent interface.
	Publish(ctx context.Context, event interface{}) error
}
