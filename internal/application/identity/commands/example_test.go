package commands_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/testhelpers"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// ExampleCommand represents a command for demonstration purposes.
// In actual implementation, replace with your real command struct.
type ExampleCommand struct {
	Email    string
	Username string
	Password string
}

// Validate validates the command input.
func (c ExampleCommand) Validate() error {
	if c.Email == "" {
		return fmt.Errorf("email is required")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// ExampleCommandHandler handles the example command.
// In actual implementation, replace with your real handler.
type ExampleCommandHandler struct {
	userRepo identity.UserRepository
}

// NewExampleCommandHandler creates a new example command handler.
func NewExampleCommandHandler(userRepo identity.UserRepository) *ExampleCommandHandler {
	return &ExampleCommandHandler{
		userRepo: userRepo,
	}
}

// Handle executes the command.
//
//nolint:cyclop // Test example command handler demonstrates full validation flow
func (h *ExampleCommandHandler) Handle(ctx context.Context, cmd ExampleCommand) (identity.UserID, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return identity.UserID{}, fmt.Errorf("invalid command: %w", err)
	}

	// Parse value objects from command
	email, err := identity.NewEmail(cmd.Email)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("invalid email: %w", err)
	}

	username, err := identity.NewUsername(cmd.Username)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("invalid username: %w", err)
	}

	// Check uniqueness (business rule)
	existingUser, err := h.userRepo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return identity.UserID{}, fmt.Errorf("email already exists")
	}

	existingUser, err = h.userRepo.FindByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return identity.UserID{}, fmt.Errorf("username already exists")
	}

	// Create password hash
	passwordHash, err := identity.NewPasswordHash(cmd.Password)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("invalid password: %w", err)
	}

	// Create user aggregate
	user, err := identity.NewUser(email, username, passwordHash)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Persist user
	if err := h.userRepo.Save(ctx, user); err != nil {
		return identity.UserID{}, fmt.Errorf("failed to save user: %w", err)
	}

	return user.ID(), nil
}

// TestExampleCommand demonstrates comprehensive table-driven testing pattern.
// This is the recommended pattern for all command handler tests.
//
//nolint:funlen // Table-driven test with comprehensive test cases
func TestExampleCommand_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     ExampleCommand
		setup   func(t *testing.T, suite *testhelpers.TestSuite)
		want    identity.UserID
		wantErr string // Use string for partial error matching
	}{
		{
			name: "successful user creation",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// User doesn't exist (unique check passes)
				suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()

				// Save succeeds
				suite.UserRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
					// Verify user was created correctly
					return !u.ID().IsZero() &&
						u.Email().String() == testhelpers.ValidEmail &&
						u.Username().String() == testhelpers.ValidUsername
				})).Return(nil).Once()
			},
			want: testhelpers.ValidUserID, // Will be non-zero
		},
		{
			name: "email already exists",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: "newusername",
				Password: testhelpers.ValidPassword,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				existingUser := testhelpers.ValidUser()
				suite.SetupEmailAlreadyExists(existingUser)
			},
			wantErr: "email already exists",
		},
		{
			name: "username already exists",
			cmd: ExampleCommand{
				Email:    "newemail@example.com",
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Email check passes
				suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()

				// Username exists
				existingUser := testhelpers.ValidUser()
				suite.SetupUsernameAlreadyExists(existingUser)
			},
			wantErr: "username already exists",
		},
		{
			name: "invalid email format",
			cmd: ExampleCommand{
				Email:    "not-an-email",
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// No repository calls should be made
			},
			wantErr: "invalid email",
		},
		{
			name: "empty email",
			cmd: ExampleCommand{
				Email:    "",
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			setup:   nil, // No setup needed for validation errors
			wantErr: "email is required",
		},
		{
			name: "empty username",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: "",
				Password: testhelpers.ValidPassword,
			},
			setup:   nil,
			wantErr: "username is required",
		},
		{
			name: "weak password",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: testhelpers.ValidUsername,
				Password: "weak",
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Uniqueness checks pass
				suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()
			},
			wantErr: "invalid password",
		},
		{
			name: "database save error",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			setup: func(t *testing.T, suite *testhelpers.TestSuite) {
				// Uniqueness checks pass
				suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()
				suite.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
					Return(nil, identity.ErrUserNotFound).Once()

				// Database error on save
				suite.UserRepo.On("Save", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database connection failed")).Once()
			},
			wantErr: "failed to save user",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable for parallel tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			suite := testhelpers.NewTestSuite(t)
			if tt.setup != nil {
				tt.setup(t, suite)
			}

			handler := NewExampleCommandHandler(suite.UserRepo)
			ctx := context.Background()

			// Act
			result, err := handler.Handle(ctx, tt.cmd)

			// Assert
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				// Verify user ID is zero on error
				assert.True(t, result.IsZero())
			} else {
				require.NoError(t, err)
				// Verify user ID is not zero on success
				assert.False(t, result.IsZero())
			}

			// Verify all mock expectations were met
			suite.AssertExpectations()
		})
	}
}

// TestExampleCommand_Validate demonstrates testing command validation separately.
func TestExampleCommand_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     ExampleCommand
		wantErr string
	}{
		{
			name: "valid command",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
		},
		{
			name: "missing email",
			cmd: ExampleCommand{
				Email:    "",
				Username: testhelpers.ValidUsername,
				Password: testhelpers.ValidPassword,
			},
			wantErr: "email is required",
		},
		{
			name: "missing username",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: "",
				Password: testhelpers.ValidPassword,
			},
			wantErr: "username is required",
		},
		{
			name: "missing password",
			cmd: ExampleCommand{
				Email:    testhelpers.ValidEmail,
				Username: testhelpers.ValidUsername,
				Password: "",
			},
			wantErr: "password is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cmd.Validate()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestExampleCommand_EventEmission demonstrates testing domain event emission.
func TestExampleCommand_EventEmission(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	// Arrange
	var capturedUser *identity.User
	suite.UserRepo.On("FindByEmail", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	suite.UserRepo.On("FindByUsername", mock.Anything, mock.Anything).
		Return(nil, identity.ErrUserNotFound)
	suite.UserRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *identity.User) bool {
		capturedUser = u
		return true
	})).Return(nil)

	handler := NewExampleCommandHandler(suite.UserRepo)
	cmd := ExampleCommand{
		Email:    testhelpers.ValidEmail,
		Username: testhelpers.ValidUsername,
		Password: testhelpers.ValidPassword,
	}

	// Act
	_, err := handler.Handle(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, capturedUser)

	// Verify domain event was emitted
	events := capturedUser.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "identity.user.created", events[0].EventType())

	suite.AssertExpectations()
}

// TestExampleCommand_UsesTestSuiteHelper demonstrates using pre-built setup helpers.
func TestExampleCommand_UsesTestSuiteHelper(t *testing.T) {
	t.Parallel()

	suite := testhelpers.NewTestSuite(t)

	// Use pre-built helper for common scenario
	suite.SetupSuccessfulUserCreation()

	handler := NewExampleCommandHandler(suite.UserRepo)
	cmd := ExampleCommand{
		Email:    testhelpers.ValidEmail,
		Username: testhelpers.ValidUsername,
		Password: testhelpers.ValidPassword,
	}

	result, err := handler.Handle(context.Background(), cmd)

	require.NoError(t, err)
	assert.False(t, result.IsZero())
	suite.AssertExpectations()
}
