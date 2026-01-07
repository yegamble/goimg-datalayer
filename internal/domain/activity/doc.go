// Package activity implements the Activity bounded context.
//
// This context is responsible for tracking user activities and generating activity feeds.
// Activities represent actions performed by users that are relevant to their followers,
// such as uploading images, liking photos, commenting, following other users, or creating albums.
//
// # Domain Model
//
// The Activity aggregate root represents a single activity in the system:
//   - ActivityID: Unique identifier for the activity
//   - ActorID: The user who performed the action
//   - ActivityType: The type of action (image_uploaded, image_liked, etc.)
//   - TargetID: The ID of the entity the action was performed on
//   - TargetType: The type of target entity (image, user, album, comment)
//   - Metadata: Additional contextual information (flexible key-value pairs)
//   - CreatedAt: Timestamp when the activity occurred
//
// # Business Rules
//
//  1. Activities are immutable once created
//  2. ActorID must be a valid user ID
//  3. ActivityType must be one of the predefined types
//  4. TargetID must reference a valid entity
//  5. Activities can be queried by actor or by a user's followed users (feed)
//  6. Old activities can be deleted for cleanup (retention policies)
//
// # Use Cases
//
//   - Record user activities (via application layer)
//   - Retrieve a user's activity history
//   - Generate activity feeds for users (showing activities from followed users)
//   - Clean up old activities based on retention policies
//
// # Integration Points
//
//   - Identity context: References UserID for actors
//   - Gallery context: References image/album IDs as targets
//   - Social context: References follow relationships for feed generation
package activity
