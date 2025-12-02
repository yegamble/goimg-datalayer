// Package identity implements the Identity bounded context for user authentication and authorization.
//
// This package contains the domain layer of the Identity context, following Domain-Driven Design principles.
// It is responsible for managing user accounts, authentication credentials, roles, and permissions.
//
// # Core Components
//
// Value Objects:
//   - UserID: Unique identifier for users (UUID-based)
//   - Email: Validated email address with disposable email detection
//   - Username: Validated username (3-32 alphanumeric characters)
//   - PasswordHash: Argon2id-hashed password following OWASP 2024 recommendations
//   - Role: User role (user, moderator, admin)
//   - UserStatus: Account status (active, pending, suspended, deleted)
//
// Entities:
//   - User: Aggregate root representing a user account
//
// Repository:
//   - UserRepository: Interface for persisting User aggregates
//
// Domain Events:
//   - UserCreated: Emitted when a new user is created
//   - UserProfileUpdated: Emitted when user profile is updated
//   - UserRoleChanged: Emitted when user role changes
//   - UserSuspended: Emitted when user is suspended
//   - UserActivated: Emitted when user is activated
//   - UserPasswordChanged: Emitted when password is changed
//
// # Design Principles
//
//  1. No Infrastructure Dependencies: This package only imports standard library and shared domain components.
//     Infrastructure concerns (database, HTTP, etc.) must not leak into this layer.
//
//  2. Immutable Value Objects: All value objects are immutable after creation and validate their invariants
//     in factory functions.
//
// 3. Aggregate Root: The User entity is the aggregate root and enforces all invariants and business rules.
//
//  4. Domain Events: State changes emit domain events for eventual consistency and integration with other
//     bounded contexts.
//
//  5. Security First: Passwords are hashed with Argon2id using OWASP 2024 parameters. Constant-time
//     comparison prevents timing attacks. Passwords are never logged or exposed.
//
// # Usage Example
//
//	// Create value objects
//	email, err := identity.NewEmail("user@example.com")
//	username, err := identity.NewUsername("johndoe")
//	passwordHash, err := identity.NewPasswordHash("SecureP@ssw0rd123")
//
//	// Create user aggregate
//	user, err := identity.NewUser(email, username, passwordHash)
//
//	// Activate user
//	err = user.Activate()
//
//	// Verify password
//	err = user.VerifyPassword("SecureP@ssw0rd123")
//
//	// Change role
//	err = user.ChangeRole(identity.RoleModerator)
//
//	// Persist via repository
//	err = userRepo.Save(ctx, user)
package identity
