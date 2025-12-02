// Package shared provides common domain primitives and utilities shared across all bounded contexts.
// This is the Shared Kernel in DDD terms - types that multiple contexts depend on.
//
// The shared kernel includes:
//   - Common error types
//   - Pagination value objects
//   - Domain event interfaces
//   - Timestamp utilities
//
// All code in this package must:
//   - Use only the standard library (no external dependencies)
//   - Be pure and side-effect free where possible
//   - Maintain immutability for value objects
//   - Use UTC for all time operations
package shared

import "errors"

// Common domain errors that can be wrapped with additional context.
// Use fmt.Errorf("operation: %w", ErrNotFound) to provide context while preserving error identity.
var (
	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates an attempt to create a resource that already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates invalid input data that fails domain validation.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates the request lacks valid authentication credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the authenticated user lacks permission for the requested operation.
	ErrForbidden = errors.New("forbidden")
)
