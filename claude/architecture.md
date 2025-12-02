# Architecture & Domain Model

> Load this guide when working on domain modeling, bounded contexts, or understanding the system structure.

## Overview

Backend for an image gallery with upload, moderation, and user management.

**Tech**: Go 1.22+, PostgreSQL 16+, Redis 7+, bimg/libvips, Goose migrations.

## Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                        │
│              (HTTP Handlers, Middleware, DTOs)               │
│              internal/interfaces/http/                       │
├─────────────────────────────────────────────────────────────┤
│                    Application Layer                         │
│         (Use Cases, Commands, Queries, App Services)         │
│              internal/application/                           │
├─────────────────────────────────────────────────────────────┤
│                      Domain Layer                            │
│    (Entities, Value Objects, Aggregates, Domain Services,    │
│     Repository Interfaces, Domain Events)                    │
│              internal/domain/                                │
├─────────────────────────────────────────────────────────────┤
│                   Infrastructure Layer                       │
│   (Repository Implementations, External Services, Storage)   │
│              internal/infrastructure/                        │
└─────────────────────────────────────────────────────────────┘
```

**Key Rule**: Dependencies point inward. Domain has NO external dependencies.

## Bounded Contexts

| Context | Aggregates | Location |
|---------|------------|----------|
| Identity & Access | User, Role, Session, OAuth | `internal/domain/identity/` |
| Gallery | Image, Album, Tag, Comment | `internal/domain/gallery/` |
| Moderation | Report, Review, Ban, Appeal | `internal/domain/moderation/` |
| Shared Kernel | Pagination, Timestamps | `internal/domain/shared/` |

## DDD Patterns

### Entities (Have Identity)

```go
// Entity with encapsulated business logic
type User struct {
    id           UserID      // Value Object for ID
    email        Email       // Value Object
    username     Username    // Value Object
    role         Role
    status       UserStatus
    createdAt    time.Time
}

// Factory enforces invariants - ALWAYS use this
func NewUser(email Email, username Username, hash PasswordHash) (*User, error) {
    if email.IsEmpty() {
        return nil, ErrEmailRequired
    }
    return &User{
        id:        NewUserID(),
        email:     email,
        username:  username,
        role:      RoleUser,
        status:    UserStatusPending,
        createdAt: time.Now().UTC(),
    }, nil
}
```

### Value Objects (Immutable, Compared by Value)

```go
// Value Object with validation in constructor
type Email struct {
    value string
}

func NewEmail(value string) (Email, error) {
    value = strings.TrimSpace(strings.ToLower(value))
    if value == "" {
        return Email{}, ErrEmailEmpty
    }
    if !emailRegex.MatchString(value) {
        return Email{}, ErrEmailInvalid
    }
    return Email{value: value}, nil
}

func (e Email) String() string      { return e.value }
func (e Email) IsEmpty() bool       { return e.value == "" }
func (e Email) Equals(o Email) bool { return e.value == o.value }
```

### Aggregates (Consistency Boundaries)

```go
// Image is an Aggregate Root - all access goes through it
type Image struct {
    id         ImageID
    ownerID    UserID
    metadata   ImageMetadata  // Value Object
    visibility Visibility
    variants   []ImageVariant // Owned entities
    events     []DomainEvent  // Collected for publishing
}

// Modifications go through aggregate root
func (i *Image) AddVariant(v ImageVariant) error {
    if i.status == ImageStatusDeleted {
        return ErrImageDeleted
    }
    for _, existing := range i.variants {
        if existing.Size == v.Size {
            return ErrVariantExists
        }
    }
    i.variants = append(i.variants, v)
    i.events = append(i.events, ImageVariantAdded{ImageID: i.id, Size: v.Size})
    return nil
}
```

### Repository Interfaces (Domain Layer)

```go
// Defined in domain layer - NO infrastructure imports
type UserRepository interface {
    NextID() UserID
    FindByID(ctx context.Context, id UserID) (*User, error)
    FindByEmail(ctx context.Context, email Email) (*User, error)
    Save(ctx context.Context, user *User) error
    Delete(ctx context.Context, id UserID) error
}
```

### Domain Events

```go
type DomainEvent interface {
    OccurredAt() time.Time
    EventType() string
}

type ImageUploaded struct {
    ImageID    ImageID
    OwnerID    UserID
    OccurredOn time.Time
}
```

## Project Structure

```
goimg-datalayer/
├── api/openapi/              # OpenAPI 3.1 spec (source of truth)
├── cmd/
│   ├── api/                  # HTTP server entrypoint
│   ├── worker/               # Background worker entrypoint
│   └── migrate/              # Migration CLI
├── internal/
│   ├── domain/               # NO EXTERNAL DEPS
│   │   ├── identity/         # User, Role, Session, OAuth
│   │   ├── gallery/          # Image, Album, Tag
│   │   ├── moderation/       # Report, Review, Ban
│   │   └── shared/           # Pagination, timestamps
│   ├── application/          # Commands, queries, services
│   │   ├── identity/commands/
│   │   ├── identity/queries/
│   │   ├── gallery/commands/
│   │   └── ...
│   ├── infrastructure/       # External implementations
│   │   ├── persistence/postgres/
│   │   ├── persistence/redis/
│   │   ├── storage/          # S3, local FS
│   │   ├── security/         # JWT, OAuth, ClamAV
│   │   └── messaging/        # Event publishing
│   └── interfaces/http/      # Presentation layer
│       ├── handlers/
│       ├── middleware/
│       └── dto/
├── tests/
├── claude/                   # Agent guides
└── docker/
```

## Layer Dependencies

| Layer | Can Import | Cannot Import |
|-------|------------|---------------|
| Domain | stdlib only | application, infrastructure, interfaces |
| Application | domain | infrastructure, interfaces |
| Infrastructure | domain, application | interfaces |
| Interfaces | domain, application, infrastructure | - |

## Quick Reference

- **Entity**: Has ID, mutable, lifecycle
- **Value Object**: No ID, immutable, compared by value
- **Aggregate**: Cluster with single root, consistency boundary
- **Repository**: Persistence abstraction, interface in domain
- **Domain Service**: Cross-aggregate logic
- **Domain Event**: Something that happened, publish after save
