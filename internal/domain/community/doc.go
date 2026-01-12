// Package community implements the community bounded context for groups and collaborative features.
//
// This bounded context handles:
//   - Group creation and management (public, private, invite-only)
//   - Group membership with role-based access control (Owner, Admin, Member)
//   - Member invitations and join workflows
//   - Group image pools and shared albums
//   - Group activity tracking
//
// The Group aggregate enforces business rules around membership, permissions,
// and content sharing within communities. This is inspired by Flickr groups
// and addresses Chevereto's most requested feature: shared/collaborative albums.
//
// Key Aggregates:
//   - Group: The root aggregate managing group lifecycle, settings, and member counts
//   - GroupMembership: Entity tracking member roles and status within a group
//
// This package follows Domain-Driven Design (DDD) principles:
//   - No infrastructure dependencies (no database, HTTP, external services)
//   - All business logic encapsulated in entities and value objects
//   - Domain events emitted for important state changes
//   - Repository interfaces defined here, implemented in infrastructure layer
package community
