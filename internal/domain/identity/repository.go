package identity

import "context"

// UserRepository defines the interface for persisting and retrieving User aggregates.
// Implementations should be provided in the infrastructure layer.
type UserRepository interface {
	// NextID generates the next available UserID.
	// This is primarily used for pre-generating IDs when needed.
	NextID() UserID

	// FindByID retrieves a user by their unique ID.
	// Returns ErrUserNotFound if the user does not exist.
	FindByID(ctx context.Context, id UserID) (*User, error)

	// FindByEmail retrieves a user by their email address.
	// Returns ErrUserNotFound if no user with that email exists.
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// FindByUsername retrieves a user by their username.
	// Returns ErrUserNotFound if no user with that username exists.
	FindByUsername(ctx context.Context, username Username) (*User, error)

	// Save persists a user to the repository.
	// If the user already exists, it is updated; otherwise, it is created.
	Save(ctx context.Context, user *User) error

	// Delete removes a user from the repository.
	// This should typically be a soft delete (status change) rather than hard delete.
	Delete(ctx context.Context, id UserID) error
}
