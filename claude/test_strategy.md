# Comprehensive Test Strategy

> Complete test strategy for the goimg-datalayer backend image gallery system.
> Load this guide when designing test suites, writing tests, or reviewing test coverage.

## Executive Summary

This test strategy defines a comprehensive, multi-layered testing approach aligned with Domain-Driven Design principles. The strategy prioritizes domain logic coverage while ensuring infrastructure reliability and API contract compliance.

**Key Metrics:**
- Overall coverage target: **80%**
- Domain layer coverage: **90%+** (business logic is critical)
- Test pyramid ratio: **60-70% unit, 20-25% integration, 10-15% E2E**
- Test execution time target: **< 2 minutes** (unit + integration)
- Flakiness tolerance: **Zero** (all tests must be deterministic)

---

## Test Pyramid Architecture

```
                    ┌─────────────────────┐
                    │   E2E Tests         │  10-15% (~50-75 tests)
                    │   Newman/Postman    │  Full API workflows
                    │   Contract Tests    │  OpenAPI validation
                    ├─────────────────────┤
                    │   Integration Tests │  20-25% (~150-200 tests)
                    │   Testcontainers    │  DB, Redis, ClamAV
                    │   Repository Layer  │  Multi-component
                    ├─────────────────────┤
                    │                     │
                    │   Unit Tests        │  60-70% (~500-700 tests)
                    │   Domain Entities   │  Pure logic
                    │   Value Objects     │  Isolated
                    │   Application       │  Mocked deps
                    │   Handlers          │  httptest
                    └─────────────────────┘
```

### Why This Distribution?

1. **Unit Tests (60-70%)**: Fast feedback, isolate defects, enable refactoring
2. **Integration Tests (20-25%)**: Verify component interactions, database contracts
3. **E2E Tests (10-15%)**: Business workflow validation, user journey testing

---

## Coverage Requirements by Layer

| Layer | Minimum | Target | Rationale |
|-------|---------|--------|-----------|
| **Domain** | 90% | 95% | Core business logic; highest risk area |
| **Application** | 85% | 90% | Use case orchestration; critical workflows |
| **Infrastructure** | 70% | 80% | External integrations; lower priority |
| **HTTP Handlers** | 75% | 85% | API contracts; user-facing |
| **Overall** | 80% | 85% | Project baseline for CI enforcement |

### Coverage Enforcement

```bash
# CI pipeline enforces these thresholds
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//' | \
  awk '{if ($1 < 80.0) exit 1}'
```

---

## Layer-Specific Test Patterns

### 1. Domain Layer Tests (90%+ Coverage)

The domain layer contains pure business logic with zero infrastructure dependencies. Tests must validate:
- Value object construction and validation
- Entity invariants and state transitions
- Aggregate behaviors and domain events
- Domain service logic

#### Value Object Test Pattern

```go
// internal/domain/identity/email_test.go
package identity_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/domain/identity"
)

func TestNewEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantErr   error
	}{
		{
			name:    "valid email",
			input:   "user@example.com",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "normalizes to lowercase",
			input:   "User@Example.COM",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "trims whitespace",
			input:   "  user@example.com  ",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "empty email",
			input:   "",
			wantErr: identity.ErrEmailEmpty,
		},
		{
			name:    "invalid format - no @",
			input:   "notanemail",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "invalid format - no domain",
			input:   "user@",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "invalid format - no local part",
			input:   "@example.com",
			wantErr: identity.ErrEmailInvalid,
		},
		{
			name:    "disposable email provider",
			input:   "test@tempmail.com",
			wantErr: identity.ErrEmailDisposable,
		},
		{
			name:    "too long (>255 chars)",
			input:   string(make([]byte, 256)) + "@example.com",
			wantErr: identity.ErrEmailTooLong,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			email, err := identity.NewEmail(tt.input)

			// Assert
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "expected error %v, got %v", tt.wantErr, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, email.String())
		})
	}
}

func TestEmail_Equals(t *testing.T) {
	t.Parallel()

	email1, _ := identity.NewEmail("user@example.com")
	email2, _ := identity.NewEmail("user@example.com")
	email3, _ := identity.NewEmail("other@example.com")

	assert.True(t, email1.Equals(email2), "same email should be equal")
	assert.False(t, email1.Equals(email3), "different emails should not be equal")
}
```

#### Entity Test Pattern

```go
// internal/domain/identity/user_test.go
package identity_test

import (
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/domain/identity"
)

func TestUser_UpdateProfile(t *testing.T) {
	t.Parallel()

	t.Run("updates display name successfully", func(t *testing.T) {
		t.Parallel()

		// Arrange
		user := newTestUser(t)
		newDisplayName := "John Doe Updated"

		// Act
		err := user.UpdateProfile(newDisplayName, user.Bio())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, newDisplayName, user.DisplayName())
		assert.Len(t, user.Events(), 1, "should emit UserUpdated event")

		event := user.Events()[0]
		assert.IsType(t, &identity.UserUpdatedEvent{}, event)
	})

	t.Run("rejects display name exceeding 100 chars", func(t *testing.T) {
		t.Parallel()

		user := newTestUser(t)
		tooLongName := string(make([]byte, 101))

		err := user.UpdateProfile(tooLongName, user.Bio())

		require.ErrorIs(t, err, identity.ErrDisplayNameTooLong)
		assert.Empty(t, user.Events(), "should not emit event on error")
	})

	t.Run("rejects bio exceeding 500 chars", func(t *testing.T) {
		t.Parallel()

		user := newTestUser(t)
		tooLongBio := string(make([]byte, 501))

		err := user.UpdateProfile(user.DisplayName(), tooLongBio)

		require.ErrorIs(t, err, identity.ErrBioTooLong)
	})
}

func TestUser_ChangePassword(t *testing.T) {
	t.Parallel()

	t.Run("changes password with valid old password", func(t *testing.T) {
		t.Parallel()

		user := newTestUser(t)
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword456!"

		err := user.ChangePassword(oldPassword, newPassword)

		require.NoError(t, err)
		assert.True(t, user.PasswordMatches(newPassword))
		assert.Len(t, user.Events(), 1)
	})

	t.Run("rejects change with wrong old password", func(t *testing.T) {
		t.Parallel()

		user := newTestUser(t)
		wrongOldPassword := "WrongPassword999!"
		newPassword := "NewPassword456!"

		err := user.ChangePassword(wrongOldPassword, newPassword)

		require.ErrorIs(t, err, identity.ErrPasswordMismatch)
	})

	t.Run("rejects weak new password", func(t *testing.T) {
		t.Parallel()

		user := newTestUser(t)
		oldPassword := "OldPassword123!"
		weakPassword := "weak"

		err := user.ChangePassword(oldPassword, weakPassword)

		require.ErrorIs(t, err, identity.ErrPasswordTooWeak)
	})
}

// Test helper - creates a valid user for testing
func newTestUser(t *testing.T) *identity.User {
	t.Helper()

	email, _ := identity.NewEmail("test@example.com")
	username, _ := identity.NewUsername("testuser")
	password := "TestPassword123!"

	user, err := identity.NewUser(email, username, password)
	require.NoError(t, err)

	return user
}
```

#### Aggregate Test Pattern

```go
// internal/domain/gallery/image_test.go
package gallery_test

import (
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/domain/gallery"
)

func TestImage_AddVariant(t *testing.T) {
	t.Parallel()

	t.Run("adds variant successfully", func(t *testing.T) {
		t.Parallel()

		image := newTestImage(t)
		variant := newTestVariant(t, gallery.VariantSizeSmall)

		err := image.AddVariant(variant)

		require.NoError(t, err)
		assert.Len(t, image.Variants(), 1)

		retrieved := image.GetVariant(gallery.VariantSizeSmall)
		require.NotNil(t, retrieved)
		assert.Equal(t, variant.Width(), retrieved.Width())
	})

	t.Run("rejects duplicate variant size", func(t *testing.T) {
		t.Parallel()

		image := newTestImage(t)
		variant1 := newTestVariant(t, gallery.VariantSizeSmall)
		variant2 := newTestVariant(t, gallery.VariantSizeSmall)

		_ = image.AddVariant(variant1)
		err := image.AddVariant(variant2)

		require.ErrorIs(t, err, gallery.ErrVariantAlreadyExists)
		assert.Len(t, image.Variants(), 1, "should still have only one variant")
	})

	t.Run("allows multiple different variant sizes", func(t *testing.T) {
		t.Parallel()

		image := newTestImage(t)
		variants := []gallery.VariantSize{
			gallery.VariantSizeThumbnail,
			gallery.VariantSizeSmall,
			gallery.VariantSizeMedium,
			gallery.VariantSizeLarge,
		}

		for _, size := range variants {
			variant := newTestVariant(t, size)
			err := image.AddVariant(variant)
			require.NoError(t, err)
		}

		assert.Len(t, image.Variants(), 4)
	})
}

func TestImage_UpdateMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		title       string
		description string
		wantErr     error
	}{
		{
			name:        "valid update",
			title:       "New Title",
			description: "New description",
			wantErr:     nil,
		},
		{
			name:        "empty title allowed",
			title:       "",
			description: "Description only",
			wantErr:     nil,
		},
		{
			name:        "title too long",
			title:       string(make([]byte, 256)),
			description: "Valid description",
			wantErr:     gallery.ErrTitleTooLong,
		},
		{
			name:        "description too long",
			title:       "Valid title",
			description: string(make([]byte, 2001)),
			wantErr:     gallery.ErrDescriptionTooLong,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			image := newTestImage(t)

			err := image.UpdateMetadata(tt.title, tt.description)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.title, image.Title())
			assert.Equal(t, tt.description, image.Description())
		})
	}
}

func TestImage_IncrementViewCount(t *testing.T) {
	t.Parallel()

	image := newTestImage(t)
	initialCount := image.ViewCount()

	image.IncrementViewCount()
	image.IncrementViewCount()
	image.IncrementViewCount()

	assert.Equal(t, initialCount+3, image.ViewCount())
}

// Test helpers
func newTestImage(t *testing.T) *gallery.Image {
	t.Helper()

	ownerID := uuid.New()
	metadata := gallery.ImageMetadata{
		OriginalFilename: "test.jpg",
		MimeType:        "image/jpeg",
		FileSize:        1024,
		Width:           1920,
		Height:          1080,
	}

	image, err := gallery.NewImage(ownerID, metadata)
	require.NoError(t, err)

	return image
}

func newTestVariant(t *testing.T, size gallery.VariantSize) *gallery.ImageVariant {
	t.Helper()

	variant, err := gallery.NewImageVariant(size, "storage-key", 800, 600, 50000, "jpeg")
	require.NoError(t, err)

	return variant
}
```

### 2. Application Layer Tests (85%+ Coverage)

Application services orchestrate domain entities and infrastructure. Tests should:
- Use mocked repositories and services
- Verify use case workflows
- Test error handling and validation
- Validate domain event emission

#### Command Handler Test Pattern

```go
// internal/application/commands/register_user_test.go
package commands_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/application/commands"
	"goimg-datalayer/internal/domain/identity"
)

func TestRegisterUserHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("registers user successfully", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockRepo := new(MockUserRepository)
		handler := commands.NewRegisterUserHandler(mockRepo)

		cmd := commands.RegisterUserCommand{
			Email:    "newuser@example.com",
			Username: "newuser",
			Password: "SecurePass123!",
		}

		mockRepo.On("ExistsByEmail", mock.Anything, mock.Anything).Return(false, nil)
		mockRepo.On("ExistsByUsername", mock.Anything, "newuser").Return(false, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*identity.User")).Return(nil)

		// Act
		userID, err := handler.Handle(context.Background(), cmd)

		// Assert
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, userID)

		mockRepo.AssertExpectations(t)
		mockRepo.AssertCalled(t, "Save", mock.Anything, mock.MatchedBy(func(user *identity.User) bool {
			return user.Email().String() == "newuser@example.com" &&
				user.Username().String() == "newuser" &&
				user.Role() == identity.RoleUser &&
				user.Status() == identity.StatusActive
		}))
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockUserRepository)
		handler := commands.NewRegisterUserHandler(mockRepo)

		cmd := commands.RegisterUserCommand{
			Email:    "existing@example.com",
			Username: "newuser",
			Password: "SecurePass123!",
		}

		mockRepo.On("ExistsByEmail", mock.Anything, mock.Anything).Return(true, nil)

		err := handler.Handle(context.Background(), cmd)

		require.ErrorIs(t, err, commands.ErrEmailAlreadyExists)
		mockRepo.AssertNotCalled(t, "Save")
	})

	t.Run("rejects duplicate username", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockUserRepository)
		handler := commands.NewRegisterUserHandler(mockRepo)

		cmd := commands.RegisterUserCommand{
			Email:    "newuser@example.com",
			Username: "existinguser",
			Password: "SecurePass123!",
		}

		mockRepo.On("ExistsByEmail", mock.Anything, mock.Anything).Return(false, nil)
		mockRepo.On("ExistsByUsername", mock.Anything, "existinguser").Return(true, nil)

		err := handler.Handle(context.Background(), cmd)

		require.ErrorIs(t, err, commands.ErrUsernameAlreadyExists)
		mockRepo.AssertNotCalled(t, "Save")
	})

	t.Run("handles repository errors", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockUserRepository)
		handler := commands.NewRegisterUserHandler(mockRepo)

		cmd := commands.RegisterUserCommand{
			Email:    "newuser@example.com",
			Username: "newuser",
			Password: "SecurePass123!",
		}

		dbError := errors.New("database connection failed")
		mockRepo.On("ExistsByEmail", mock.Anything, mock.Anything).Return(false, dbError)

		err := handler.Handle(context.Background(), cmd)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
	})
}

// Mock repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Save(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email identity.Email) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}
```

#### Query Handler Test Pattern

```go
// internal/application/queries/get_image_test.go
package queries_test

import (
	"context"
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/application/queries"
	"goimg-datalayer/internal/domain/gallery"
)

func TestGetImageHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("returns image successfully", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockImageRepository)
		handler := queries.NewGetImageHandler(mockRepo)

		imageID := uuid.New()
		ownerID := uuid.New()
		expectedImage := &gallery.Image{
			ID:      imageID,
			OwnerID: ownerID,
			Title:   "Test Image",
		}

		mockRepo.On("FindByID", mock.Anything, imageID).Return(expectedImage, nil)

		query := queries.GetImageQuery{ImageID: imageID}
		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Equal(t, imageID, result.ID)
		assert.Equal(t, "Test Image", result.Title)
	})

	t.Run("returns error for non-existent image", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockImageRepository)
		handler := queries.NewGetImageHandler(mockRepo)

		imageID := uuid.New()
		mockRepo.On("FindByID", mock.Anything, imageID).Return(nil, gallery.ErrImageNotFound)

		query := queries.GetImageQuery{ImageID: imageID}
		_, err := handler.Handle(context.Background(), query)

		require.ErrorIs(t, err, gallery.ErrImageNotFound)
	})
}
```

### 3. Infrastructure Layer Tests (70%+ Coverage)

Infrastructure tests verify interactions with external systems. Use testcontainers for real dependencies.

#### Repository Test Pattern with Testcontainers

```go
// tests/integration/repository/user_repository_test.go
package repository_test

import (
	"context"
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/domain/identity"
	"goimg-datalayer/internal/infrastructure/persistence/postgres"
	"goimg-datalayer/tests/integration/testhelpers"
)

func TestUserRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	suite := testhelpers.SetupTestSuite(t)
	repo := postgres.NewUserRepository(suite.DB)

	t.Run("saves new user successfully", func(t *testing.T) {
		user := newTestUser(t)

		err := repo.Save(context.Background(), user)

		require.NoError(t, err)
	})

	t.Run("updates existing user", func(t *testing.T) {
		user := newTestUser(t)
		_ = repo.Save(context.Background(), user)

		user.UpdateProfile("New Display Name", user.Bio())
		err := repo.Save(context.Background(), user)

		require.NoError(t, err)

		found, _ := repo.FindByID(context.Background(), user.ID())
		assert.Equal(t, "New Display Name", found.DisplayName())
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	suite := testhelpers.SetupTestSuite(t)
	repo := postgres.NewUserRepository(suite.DB)

	t.Run("finds existing user by email", func(t *testing.T) {
		user := newTestUser(t)
		_ = repo.Save(context.Background(), user)

		found, err := repo.FindByEmail(context.Background(), user.Email())

		require.NoError(t, err)
		assert.Equal(t, user.ID(), found.ID())
		assert.True(t, user.Email().Equals(found.Email()))
	})

	t.Run("returns not found for non-existent email", func(t *testing.T) {
		email, _ := identity.NewEmail("nonexistent@example.com")

		_, err := repo.FindByEmail(context.Background(), email)

		require.ErrorIs(t, err, identity.ErrUserNotFound)
	})
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	suite := testhelpers.SetupTestSuite(t)
	repo := postgres.NewUserRepository(suite.DB)

	t.Run("returns true for existing email", func(t *testing.T) {
		user := newTestUser(t)
		_ = repo.Save(context.Background(), user)

		exists, err := repo.ExistsByEmail(context.Background(), user.Email())

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent email", func(t *testing.T) {
		email, _ := identity.NewEmail("nonexistent@example.com")

		exists, err := repo.ExistsByEmail(context.Background(), email)

		require.NoError(t, err)
		assert.False(t, exists)
	})
}
```

#### Testcontainers Setup

```go
// tests/integration/testhelpers/setup.go
package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestSuite struct {
	PostgresContainer testcontainers.Container
	RedisContainer    testcontainers.Container
	DB                *sqlx.DB
	RedisClient       *redis.Client
	cleanup           func()
}

func SetupTestSuite(t *testing.T) *TestSuite {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("goimg_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Start Redis container
	redisContainer, err := rediscontainer.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	// Get PostgreSQL connection string
	pgConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	// Connect to PostgreSQL
	db, err := sqlx.Connect("postgres", pgConnStr)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	// Run migrations
	if err := goose.Up(db.DB, "../../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Get Redis connection
	redisHost, _ := redisContainer.Host(ctx)
	redisPort, _ := redisContainer.MappedPort(ctx, "6379")
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})

	// Verify Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to connect to redis: %v", err)
	}

	suite := &TestSuite{
		PostgresContainer: pgContainer,
		RedisContainer:    redisContainer,
		DB:                db,
		RedisClient:       redisClient,
	}

	// Cleanup function
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
		if err := redisClient.Close(); err != nil {
			t.Logf("failed to close redis client: %v", err)
		}
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	})

	return suite
}

// CleanDatabase truncates all tables for test isolation
func (s *TestSuite) CleanDatabase(t *testing.T) {
	t.Helper()

	tables := []string{
		"users",
		"sessions",
		"images",
		"image_variants",
		"albums",
		"album_images",
		"tags",
		"image_tags",
		"likes",
		"comments",
		"reports",
		"user_bans",
		"audit_logs",
	}

	for _, table := range tables {
		_, err := s.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("failed to truncate table %s: %v", table, err)
		}
	}
}
```

### 4. HTTP Handler Tests (75%+ Coverage)

Handler tests verify HTTP contracts, authentication, authorization, and error responses.

#### Handler Test Pattern

```go
// internal/interfaces/http/handlers/auth_handler_test.go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/interfaces/http/handlers"
)

func TestAuthHandler_Register(t *testing.T) {
	t.Parallel()

	t.Run("registers user successfully", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		handler := handlers.NewAuthHandler(mockService)

		requestBody := map[string]string{
			"email":    "newuser@example.com",
			"username": "newuser",
			"password": "SecurePass123!",
		}
		body, _ := json.Marshal(requestBody)

		mockService.On("Register", mock.Anything, mock.Anything).Return(uuid.New(), nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&response)
		assert.Contains(t, response, "id")
		assert.Contains(t, response, "email")
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		handler := handlers.NewAuthHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var problem map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&problem)
		assert.Equal(t, "Validation Error", problem["title"])
	})

	t.Run("returns 409 for duplicate email", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		handler := handlers.NewAuthHandler(mockService)

		requestBody := map[string]string{
			"email":    "existing@example.com",
			"username": "newuser",
			"password": "SecurePass123!",
		}
		body, _ := json.Marshal(requestBody)

		mockService.On("Register", mock.Anything, mock.Anything).
			Return(uuid.Nil, commands.ErrEmailAlreadyExists)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	t.Run("logs in successfully with valid credentials", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		handler := handlers.NewAuthHandler(mockService)

		requestBody := map[string]string{
			"email":    "user@example.com",
			"password": "ValidPass123!",
		}
		body, _ := json.Marshal(requestBody)

		expectedResponse := &handlers.TokenResponse{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh.token.here",
			ExpiresIn:    900,
			TokenType:    "Bearer",
		}

		mockService.On("Login", mock.Anything, "user@example.com", "ValidPass123!").
			Return(expectedResponse, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response handlers.TokenResponse
		json.NewDecoder(rec.Body).Decode(&response)
		assert.Equal(t, expectedResponse.AccessToken, response.AccessToken)
		assert.Equal(t, expectedResponse.RefreshToken, response.RefreshToken)
	})

	t.Run("returns 401 for invalid credentials", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		handler := handlers.NewAuthHandler(mockService)

		requestBody := map[string]string{
			"email":    "user@example.com",
			"password": "WrongPassword!",
		}
		body, _ := json.Marshal(requestBody)

		mockService.On("Login", mock.Anything, "user@example.com", "WrongPassword!").
			Return(nil, handlers.ErrInvalidCredentials)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("returns 429 when rate limited", func(t *testing.T) {
		t.Parallel()

		mockService := new(MockAuthService)
		mockRateLimiter := new(MockRateLimiter)
		handler := handlers.NewAuthHandler(mockService, handlers.WithRateLimiter(mockRateLimiter))

		requestBody := map[string]string{
			"email":    "user@example.com",
			"password": "ValidPass123!",
		}
		body, _ := json.Marshal(requestBody)

		mockRateLimiter.On("Allow", mock.Anything).Return(false)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Header().Get("X-RateLimit-Limit"), "5")
	})
}
```

---

## E2E and Contract Tests

### Newman/Postman E2E Tests

```json
// tests/e2e/postman/goimg-collection.json
{
  "info": {
    "name": "goimg API Tests",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Authentication Flow",
      "item": [
        {
          "name": "Register User",
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Status code is 201', function () {",
                  "    pm.response.to.have.status(201);",
                  "});",
                  "",
                  "pm.test('Response has user ID', function () {",
                  "    var jsonData = pm.response.json();",
                  "    pm.expect(jsonData).to.have.property('id');",
                  "    pm.collectionVariables.set('userId', jsonData.id);",
                  "});"
                ]
              }
            }
          ],
          "request": {
            "method": "POST",
            "header": [],
            "body": {
              "mode": "raw",
              "raw": "{\n  \"email\": \"{{$randomEmail}}\",\n  \"username\": \"{{$randomUserName}}\",\n  \"password\": \"SecurePass123!\"\n}",
              "options": {
                "raw": {
                  "language": "json"
                }
              }
            },
            "url": {
              "raw": "{{baseUrl}}/api/v1/auth/register",
              "host": ["{{baseUrl}}"],
              "path": ["api", "v1", "auth", "register"]
            }
          }
        },
        {
          "name": "Login",
          "event": [
            {
              "listen": "test",
              "script": {
                "exec": [
                  "pm.test('Status code is 200', function () {",
                  "    pm.response.to.have.status(200);",
                  "});",
                  "",
                  "pm.test('Response has access token', function () {",
                  "    var jsonData = pm.response.json();",
                  "    pm.expect(jsonData).to.have.property('access_token');",
                  "    pm.expect(jsonData).to.have.property('refresh_token');",
                  "    pm.collectionVariables.set('accessToken', jsonData.access_token);",
                  "    pm.collectionVariables.set('refreshToken', jsonData.refresh_token);",
                  "});"
                ]
              }
            }
          ],
          "request": {
            "method": "POST",
            "header": [],
            "body": {
              "mode": "raw",
              "raw": "{\n  \"email\": \"{{userEmail}}\",\n  \"password\": \"{{userPassword}}\"\n}",
              "options": {
                "raw": {
                  "language": "json"
                }
              }
            },
            "url": {
              "raw": "{{baseUrl}}/api/v1/auth/login",
              "host": ["{{baseUrl}}"],
              "path": ["api", "v1", "auth", "login"]
            }
          }
        }
      ]
    }
  ]
}
```

### Contract Tests (OpenAPI Validation)

```go
// tests/contract/openapi_test.go
package contract_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/interfaces/http"
)

func TestAPIMatchesOpenAPISpec(t *testing.T) {
	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../api/openapi/openapi.yaml")
	require.NoError(t, err)

	err = doc.Validate(context.Background())
	require.NoError(t, err)

	// Create router from spec
	router, err := gorillamux.NewRouter(doc)
	require.NoError(t, err)

	// Create HTTP server
	server := http.NewServer()
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	t.Run("POST /api/v1/auth/register validates request", func(t *testing.T) {
		requestBody := `{"email":"test@example.com","username":"testuser","password":"SecurePass123!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := router.FindRoute(req)
		require.NoError(t, err)

		requestValidationInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		err = openapi3filter.ValidateRequest(context.Background(), requestValidationInput)
		assert.NoError(t, err, "request should match OpenAPI spec")
	})

	t.Run("POST /api/v1/auth/register rejects invalid request", func(t *testing.T) {
		requestBody := `{"email":"invalid-email","username":"ab","password":"weak"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := router.FindRoute(req)
		require.NoError(t, err)

		requestValidationInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		err = openapi3filter.ValidateRequest(context.Background(), requestValidationInput)
		assert.Error(t, err, "invalid request should fail validation")
	})
}
```

---

## Test Execution Matrix by Sprint

| Sprint | Test Focus | Deliverables |
|--------|------------|--------------|
| **1-2** | Domain layer unit tests | • All value object tests<br>• Entity factory tests<br>• Aggregate invariant tests<br>• 90%+ domain coverage |
| **3** | Infrastructure integration tests | • PostgreSQL repository tests<br>• Redis session store tests<br>• JWT service tests<br>• Migration validation |
| **4** | Application + HTTP tests | • Command handler tests (Register, Login)<br>• Query handler tests<br>• Auth handler E2E<br>• Rate limiting tests |
| **5** | Storage + ClamAV integration | • Image processor tests<br>• S3 storage provider tests<br>• ClamAV scanner tests<br>• File upload security tests |
| **6** | Gallery E2E tests | • Image upload workflow<br>• Album CRUD tests<br>• Search query tests<br>• Social features (likes, comments) |
| **7** | Moderation tests | • Report workflow tests<br>• RBAC authorization tests<br>• Audit logging tests<br>• Ban system tests |
| **8** | Full regression + security | • Security test suite (OWASP)<br>• Load testing (k6)<br>• Penetration testing<br>• Performance benchmarks |
| **9** | Smoke tests + final regression | • Production readiness tests<br>• Health check validation<br>• Monitoring integration<br>• Final regression suite |

---

## Test Fixtures and Helpers

### Directory Structure

```
tests/
├── fixtures/
│   ├── images/
│   │   ├── valid_jpeg.jpg          # 1920x1080, 1MB
│   │   ├── valid_png.png           # 800x600, 500KB
│   │   ├── valid_gif.gif           # Animated, 200KB
│   │   ├── valid_webp.webp         # 1024x768, 300KB
│   │   ├── oversized.jpg           # 20MB (exceeds limit)
│   │   ├── malformed.jpg           # Corrupted file
│   │   ├── polyglot.jpg            # JPEG + ZIP polyglot
│   │   └── malware_sample.bin      # EICAR test file
│   └── data/
│       ├── seed_users.sql          # Test user data
│       └── seed_images.sql         # Test image data
└── testhelpers/
    ├── setup.go                     # Testcontainer setup
    ├── fixtures.go                  # Test data builders
    └── assertions.go                # Custom assertions
```

### Test Helper Examples

```go
// tests/testhelpers/fixtures.go
package testhelpers

import (
	"testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"goimg-datalayer/internal/domain/identity"
	"goimg-datalayer/internal/domain/gallery"
)

// UserBuilder provides fluent API for creating test users
type UserBuilder struct {
	email    string
	username string
	password string
	role     identity.Role
	status   identity.Status
}

func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		email:    "test@example.com",
		username: "testuser",
		password: "SecurePass123!",
		role:     identity.RoleUser,
		status:   identity.StatusActive,
	}
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.username = username
	return b
}

func (b *UserBuilder) WithRole(role identity.Role) *UserBuilder {
	b.role = role
	return b
}

func (b *UserBuilder) Build(t *testing.T) *identity.User {
	t.Helper()

	email, err := identity.NewEmail(b.email)
	require.NoError(t, err)

	username, err := identity.NewUsername(b.username)
	require.NoError(t, err)

	user, err := identity.NewUser(email, username, b.password)
	require.NoError(t, err)

	if b.role != identity.RoleUser {
		user.SetRole(b.role)
	}

	if b.status != identity.StatusActive {
		user.SetStatus(b.status)
	}

	return user
}

// ImageBuilder provides fluent API for creating test images
type ImageBuilder struct {
	ownerID     uuid.UUID
	title       string
	description string
	visibility  gallery.Visibility
	mimeType    string
	fileSize    int64
	width       int
	height      int
}

func NewImageBuilder() *ImageBuilder {
	return &ImageBuilder{
		ownerID:     uuid.New(),
		title:       "Test Image",
		description: "Test description",
		visibility:  gallery.VisibilityPrivate,
		mimeType:    "image/jpeg",
		fileSize:    1024000,
		width:       1920,
		height:      1080,
	}
}

func (b *ImageBuilder) WithOwner(ownerID uuid.UUID) *ImageBuilder {
	b.ownerID = ownerID
	return b
}

func (b *ImageBuilder) WithVisibility(visibility gallery.Visibility) *ImageBuilder {
	b.visibility = visibility
	return b
}

func (b *ImageBuilder) Build(t *testing.T) *gallery.Image {
	t.Helper()

	metadata := gallery.ImageMetadata{
		OriginalFilename: "test.jpg",
		MimeType:        b.mimeType,
		FileSize:        b.fileSize,
		Width:           b.width,
		Height:          b.height,
	}

	image, err := gallery.NewImage(b.ownerID, metadata)
	require.NoError(t, err)

	if b.title != "" {
		image.UpdateMetadata(b.title, b.description)
	}

	if b.visibility != gallery.VisibilityPrivate {
		image.SetVisibility(b.visibility)
	}

	return image
}
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Test Suite

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run unit tests
        run: make test-unit
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unit

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run integration tests
        run: make test-integration
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage-integration.out
          flags: integration

  e2e-tests:
    name: E2E Tests
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: goimg_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Install Newman
        run: npm install -g newman
      - name: Run migrations
        run: make migrate-up
      - name: Start API server
        run: make run &
      - name: Wait for server
        run: timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done'
      - name: Run E2E tests
        run: newman run tests/e2e/postman/goimg-collection.json -e tests/e2e/postman/environment/ci.json

  contract-tests:
    name: Contract Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run contract tests
        run: go test -v ./tests/contract/...
      - name: Validate OpenAPI spec
        run: make validate-openapi

  coverage-gate:
    name: Coverage Gate
    needs: [unit-tests, integration-tests]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Check coverage threshold
        run: |
          make test-coverage
          COVERAGE=$(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 80.0" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 80% threshold"
            exit 1
          fi
          echo "Coverage $COVERAGE% meets threshold"
```

---

## Agent Collaboration Model

### Agent Responsibilities

| Agent | Primary Responsibility | Test Ownership |
|-------|------------------------|----------------|
| **backend-test-architect** | Unit and integration test design | • Domain layer tests<br>• Application layer tests<br>• Repository integration tests<br>• Test helpers and fixtures |
| **test-strategist** | E2E and contract test design | • Newman/Postman collections<br>• OpenAPI contract tests<br>• Security test scenarios<br>• Load testing scripts |
| **senior-go-architect** | Production code review | • Verify testability of code<br>• Review test coverage reports<br>• Ensure DDD compliance |

### Collaboration Workflow

1. **Development Phase**:
   - Developer writes feature code
   - `backend-test-architect` writes unit/integration tests
   - `test-strategist` adds E2E scenarios

2. **Review Phase**:
   - `senior-go-architect` reviews code and test coverage
   - Ensures 80%+ coverage threshold met
   - Validates test quality (not just quantity)

3. **CI Phase**:
   - All tests run in CI pipeline
   - Coverage reports generated
   - Contract validation enforced

---

## Security Test Requirements

### OWASP Top 10 Coverage

```go
// tests/security/owasp_test.go
package security_test

import (
	"testing"
	"net/http"
	"net/http/httptest"
)

// A01:2021 – Broken Access Control
func TestOWASP_A01_BrokenAccessControl(t *testing.T) {
	t.Run("prevents IDOR on user profile", func(t *testing.T) {
		// User A tries to access User B's profile
		// Should return 403 Forbidden
	})

	t.Run("prevents privilege escalation", func(t *testing.T) {
		// Regular user tries to access admin endpoint
		// Should return 403 Forbidden
	})

	t.Run("prevents forced browsing to admin panel", func(t *testing.T) {
		// Unauthenticated user tries /admin
		// Should return 401 Unauthorized
	})
}

// A02:2021 – Cryptographic Failures
func TestOWASP_A02_CryptographicFailures(t *testing.T) {
	t.Run("passwords stored with Argon2id", func(t *testing.T) {
		// Verify password hashing uses Argon2id
	})

	t.Run("JWT signed with RS256", func(t *testing.T) {
		// Verify JWT uses asymmetric signing
	})

	t.Run("refresh tokens stored hashed", func(t *testing.T) {
		// Verify refresh tokens not stored in plaintext
	})
}

// A03:2021 – Injection
func TestOWASP_A03_Injection(t *testing.T) {
	t.Run("prevents SQL injection in search", func(t *testing.T) {
		// Try SQL injection payloads in search query
		// Should safely parameterize queries
	})

	t.Run("prevents command injection in image processing", func(t *testing.T) {
		// Try command injection in filename
		// Should sanitize inputs
	})
}

// A04:2021 – Insecure Design
func TestOWASP_A04_InsecureDesign(t *testing.T) {
	t.Run("implements account lockout", func(t *testing.T) {
		// 5 failed login attempts should lock account
	})

	t.Run("implements rate limiting on uploads", func(t *testing.T) {
		// 50 uploads/hour should be enforced
	})
}

// A05:2021 – Security Misconfiguration
func TestOWASP_A05_SecurityMisconfiguration(t *testing.T) {
	t.Run("sets security headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
	})

	t.Run("does not expose stack traces in production", func(t *testing.T) {
		// Error responses should not include stack traces
	})
}

// A07:2021 – Identification and Authentication Failures
func TestOWASP_A07_IdentificationAuthenticationFailures(t *testing.T) {
	t.Run("prevents account enumeration", func(t *testing.T) {
		// Login with non-existent email should return same error as wrong password
	})

	t.Run("detects token replay attacks", func(t *testing.T) {
		// Reusing a refresh token should revoke entire token family
	})

	t.Run("enforces strong password policy", func(t *testing.T) {
		// Passwords must be 12+ chars with complexity
	})
}

// A08:2021 – Software and Data Integrity Failures
func TestOWASP_A08_SoftwareDataIntegrityFailures(t *testing.T) {
	t.Run("validates file MIME type by magic bytes", func(t *testing.T) {
		// File extension should not be trusted
		// Must detect MIME by content
	})

	t.Run("re-encodes uploaded images", func(t *testing.T) {
		// Prevents polyglot file exploits
	})
}

// A09:2021 – Security Logging and Monitoring Failures
func TestOWASP_A09_SecurityLoggingMonitoring(t *testing.T) {
	t.Run("logs authentication events", func(t *testing.T) {
		// Login, logout, failed attempts should be logged
	})

	t.Run("logs authorization failures", func(t *testing.T) {
		// 403 Forbidden should be logged with user ID
	})

	t.Run("does not log sensitive data", func(t *testing.T) {
		// Passwords, tokens should never appear in logs
	})
}

// A10:2021 – Server-Side Request Forgery (SSRF)
func TestOWASP_A10_SSRF(t *testing.T) {
	t.Run("validates image URLs if fetching from URL", func(t *testing.T) {
		// Prevent fetching from localhost/internal IPs
	})
}
```

---

## Performance and Load Testing

### Load Testing with k6

```javascript
// tests/load/image_upload.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '1m', target: 10 },  // Ramp up to 10 users
    { duration: '3m', target: 10 },  // Stay at 10 users
    { duration: '1m', target: 50 },  // Ramp up to 50 users
    { duration: '5m', target: 50 },  // Stay at 50 users
    { duration: '1m', target: 0 },   // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests under 500ms
    'errors': ['rate<0.1'],              // Error rate under 10%
  },
};

export default function() {
  const url = 'http://localhost:8080/api/v1/images';

  const payload = {
    file: http.file(open('./fixtures/test_image.jpg', 'b'), 'test.jpg'),
    title: 'Load Test Image',
    visibility: 'public',
  };

  const params = {
    headers: {
      'Authorization': `Bearer ${__ENV.ACCESS_TOKEN}`,
    },
  };

  const res = http.post(url, payload, params);

  const success = check(res, {
    'status is 201': (r) => r.status === 201,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  errorRate.add(!success);

  sleep(1);
}
```

---

## Summary

This comprehensive test strategy provides:

1. **Clear coverage targets** by layer (80% overall, 90% domain)
2. **Concrete test patterns** for each architectural layer
3. **Testcontainers integration** for realistic integration tests
4. **E2E and contract tests** for API validation
5. **Security test coverage** aligned with OWASP Top 10
6. **CI/CD integration** with automated enforcement
7. **Agent collaboration model** for distributed test ownership
8. **Sprint-by-sprint test plan** aligned with development roadmap

**Next Steps:**
1. Review and approve this strategy
2. Set up test infrastructure (testcontainers, Newman)
3. Begin Sprint 1-2 with domain layer test implementation
4. Establish CI pipeline with coverage gates
