package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	"github.com/yegamble/goimg-datalayer/internal/application/identity/dto"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// RegisterUserCommand represents the intent to create a new user account.
// It encapsulates all information needed for user registration including
// IP and UserAgent for security auditing.
type RegisterUserCommand struct {
	Email     string
	Username  string
	Password  string
	IPAddress string
	UserAgent string
}

// Implement Command interface from types.go
func (RegisterUserCommand) isCommand() {}

// RegisterUserHandler processes user registration commands.
// It orchestrates the registration workflow: validation, uniqueness checks,
// password hashing, user creation, and event publishing.
type RegisterUserHandler struct {
	users          identity.UserRepository
	eventPublisher appidentity.EventPublisher
	logger         *zerolog.Logger
}

// NewRegisterUserHandler creates a new RegisterUserHandler with the given dependencies.
// All dependencies are injected via constructor for testability and maintainability.
func NewRegisterUserHandler(
	users identity.UserRepository,
	eventPublisher appidentity.EventPublisher,
	logger *zerolog.Logger,
) *RegisterUserHandler {
	return &RegisterUserHandler{
		users:          users,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// Handle executes the user registration use case.
//
// Process flow:
//  1. Convert DTOs to domain value objects (validation happens here)
//  2. Check email uniqueness (business rule)
//  3. Check username uniqueness (business rule)
//  4. Hash password using Argon2id
//  5. Create User aggregate via domain factory
//  6. Persist user via repository
//  7. Publish domain events after successful save
//  8. Return UserDTO (without password hash)
//
// Returns:
//   - UserDTO on successful registration
//   - ErrEmailAlreadyExists if email is taken
//   - ErrUsernameAlreadyExists if username is taken
//   - Validation errors from domain value objects
func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (*dto.UserDTO, error) {
	// 1. Convert primitives to domain value objects (validation happens here)
	email, err := identity.NewEmail(cmd.Email)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("email", cmd.Email).
			Msg("invalid email format during registration")
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	username, err := identity.NewUsername(cmd.Username)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Str("username", cmd.Username).
			Msg("invalid username during registration")
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	// 2. Check email uniqueness (business rule)
	existingByEmail, err := h.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, identity.ErrUserNotFound) {
		h.logger.Error().
			Err(err).
			Str("email", email.String()).
			Msg("failed to check email uniqueness")
		return nil, fmt.Errorf("check email uniqueness: %w", err)
	}
	if existingByEmail != nil {
		h.logger.Debug().
			Str("email", email.String()).
			Msg("registration attempt with existing email")
		return nil, appidentity.ErrEmailAlreadyExists
	}

	// 3. Check username uniqueness (business rule)
	existingByUsername, err := h.users.FindByUsername(ctx, username)
	if err != nil && !errors.Is(err, identity.ErrUserNotFound) {
		h.logger.Error().
			Err(err).
			Str("username", username.String()).
			Msg("failed to check username uniqueness")
		return nil, fmt.Errorf("check username uniqueness: %w", err)
	}
	if existingByUsername != nil {
		h.logger.Debug().
			Str("username", username.String()).
			Msg("registration attempt with existing username")
		return nil, appidentity.ErrUsernameAlreadyExists
	}

	// 4. Hash password using Argon2id (application layer concern)
	passwordHash, err := identity.NewPasswordHash(cmd.Password)
	if err != nil {
		h.logger.Debug().
			Err(err).
			Msg("password validation failed during registration")
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// 5. Create User aggregate via domain factory
	user, err := identity.NewUser(email, username, passwordHash)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("email", email.String()).
			Str("username", username.String()).
			Msg("failed to create user aggregate")
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 6. Persist user to repository
	if err := h.users.Save(ctx, user); err != nil {
		h.logger.Error().
			Err(err).
			Str("user_id", user.ID().String()).
			Str("email", email.String()).
			Str("username", username.String()).
			Msg("failed to save user")
		return nil, fmt.Errorf("save user: %w", err)
	}

	// 7. Publish domain events AFTER successful save
	// Event publishing failures should NOT fail the registration
	for _, event := range user.Events() {
		if err := h.eventPublisher.Publish(ctx, event); err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", user.ID().String()).
				Str("event_type", event.EventType()).
				Msg("failed to publish domain event after user registration")
		}
	}
	user.ClearEvents()

	h.logger.Info().
		Str("user_id", user.ID().String()).
		Str("email", email.String()).
		Str("username", username.String()).
		Str("ip_address", cmd.IPAddress).
		Str("user_agent", cmd.UserAgent).
		Msg("user registered successfully")

	// 8. Convert to DTO (exclude password hash)
	userDTO := dto.FromDomain(user)
	return &userDTO, nil
}
