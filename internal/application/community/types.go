// Package community implements the application layer for the Community bounded context.
// It provides commands and queries for group management and membership operations.
package community

import (
	"context"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// Command represents a write operation in the Community bounded context.
// All commands must implement this marker interface.
type Command interface {
	isCommand()
}

// Query represents a read operation in the Community bounded context.
// All queries must implement this marker interface.
type Query interface {
	isQuery()
}

// EventPublisher publishes domain events to the messaging infrastructure.
// Implementations live in the infrastructure layer.
type EventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}
