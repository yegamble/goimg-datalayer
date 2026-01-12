package community

import (
	"fmt"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

const (
	// MaxDescriptionLength is the maximum length for group descriptions.
	MaxDescriptionLength = 1000
)

// Group is the aggregate root for the community bounded context.
// It manages group lifecycle, settings, membership counts, and emits domain events.
// Groups can be public, private, or invite-only, and contain shared image pools and albums.
type Group struct {
	id           GroupID
	name         GroupName
	slug         GroupSlug
	description  string
	groupType    GroupType
	ownerID      identity.UserID
	settings     GroupSettings
	memberCount  int
	imageCount   int
	albumCount   int
	coverImageID *gallery.ImageID
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
}

// NewGroup creates a new Group aggregate with the given owner, name, slug, and type.
// The owner is automatically counted as the first member.
// Returns an error if validation fails.
func NewGroup(ownerID identity.UserID, name GroupName, slug GroupSlug, groupType GroupType) (*Group, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf("%w: owner ID is required", shared.ErrInvalidInput)
	}

	if name.IsEmpty() {
		return nil, ErrGroupNameRequired
	}

	if slug.IsEmpty() {
		return nil, ErrGroupSlugRequired
	}

	if !groupType.IsValid() {
		return nil, ErrInvalidGroupType
	}

	now := time.Now().UTC()
	group := &Group{
		id:           NewGroupID(),
		name:         name,
		slug:         slug,
		description:  "",
		groupType:    groupType,
		ownerID:      ownerID,
		settings:     DefaultGroupSettings(),
		memberCount:  1, // Owner is the first member
		imageCount:   0,
		albumCount:   0,
		coverImageID: nil,
		createdAt:    now,
		updatedAt:    now,
		events:       []shared.DomainEvent{},
	}

	group.addEvent(&GroupCreated{
		BaseEvent: shared.NewBaseEvent("community.group.created", group.id.String()),
		GroupID:   group.id,
		OwnerID:   group.ownerID,
		Name:      group.name,
		Slug:      group.slug,
		GroupType: group.groupType,
	})

	return group, nil
}

// ReconstructGroup reconstitutes a Group from persistence without validation or events.
// Use this only when loading from the database.
func ReconstructGroup(
	id GroupID,
	name GroupName,
	slug GroupSlug,
	description string,
	groupType GroupType,
	ownerID identity.UserID,
	settings GroupSettings,
	memberCount, imageCount, albumCount int,
	coverImageID *gallery.ImageID,
	createdAt, updatedAt time.Time,
) *Group {
	return &Group{
		id:           id,
		name:         name,
		slug:         slug,
		description:  description,
		groupType:    groupType,
		ownerID:      ownerID,
		settings:     settings,
		memberCount:  memberCount,
		imageCount:   imageCount,
		albumCount:   albumCount,
		coverImageID: coverImageID,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       []shared.DomainEvent{},
	}
}

// Getters

// ID returns the unique identifier of the group.
func (g *Group) ID() GroupID {
	return g.id
}

// Name returns the group name.
func (g *Group) Name() GroupName {
	return g.name
}

// Slug returns the URL-safe slug.
func (g *Group) Slug() GroupSlug {
	return g.slug
}

// Description returns the group description.
func (g *Group) Description() string {
	return g.description
}

// GroupType returns the group type (public, private, invite-only).
func (g *Group) GroupType() GroupType {
	return g.groupType
}

// OwnerID returns the ID of the group owner.
func (g *Group) OwnerID() identity.UserID {
	return g.ownerID
}

// Settings returns the group settings.
func (g *Group) Settings() GroupSettings {
	return g.settings
}

// MemberCount returns the number of members in the group.
func (g *Group) MemberCount() int {
	return g.memberCount
}

// ImageCount returns the number of images in the group pool.
func (g *Group) ImageCount() int {
	return g.imageCount
}

// AlbumCount returns the number of albums in the group.
func (g *Group) AlbumCount() int {
	return g.albumCount
}

// CoverImageID returns the ID of the cover image, or nil if not set.
func (g *Group) CoverImageID() *gallery.ImageID {
	return g.coverImageID
}

// CreatedAt returns when the group was created.
func (g *Group) CreatedAt() time.Time {
	return g.createdAt
}

// UpdatedAt returns when the group was last modified.
func (g *Group) UpdatedAt() time.Time {
	return g.updatedAt
}

// Events returns the domain events that have occurred.
func (g *Group) Events() []shared.DomainEvent {
	return g.events
}

// ClearEvents clears all pending domain events.
func (g *Group) ClearEvents() {
	g.events = []shared.DomainEvent{}
}

// Behavior Methods

// UpdateDescription changes the group description.
func (g *Group) UpdateDescription(description string) error {
	description = strings.TrimSpace(description)
	if len(description) > MaxDescriptionLength {
		return fmt.Errorf("%w: got %d characters", ErrGroupDescTooLong, len(description))
	}

	if g.description == description {
		return nil // No change
	}

	g.description = description
	g.updatedAt = time.Now().UTC()

	g.addEvent(&GroupDescriptionUpdated{
		BaseEvent:      shared.NewBaseEvent("community.group.description_updated", g.id.String()),
		GroupID:        g.id,
		NewDescription: description,
	})

	return nil
}

// UpdateSettings changes the group settings.
func (g *Group) UpdateSettings(settings GroupSettings) error {
	if g.settings.Equals(settings) {
		return nil // No change
	}

	oldSettings := g.settings
	g.settings = settings
	g.updatedAt = time.Now().UTC()

	g.addEvent(&GroupSettingsUpdated{
		BaseEvent:   shared.NewBaseEvent("community.group.settings_updated", g.id.String()),
		GroupID:     g.id,
		OldSettings: oldSettings,
		NewSettings: settings,
	})

	return nil
}

// SetCoverImage sets the cover image for the group.
// Pass nil to remove the cover image.
func (g *Group) SetCoverImage(imageID *gallery.ImageID) {
	// Check if actually changing
	if g.coverImageID == nil && imageID == nil {
		return // Both nil
	}
	if g.coverImageID != nil && imageID != nil && g.coverImageID.Equals(*imageID) {
		return // Same image
	}

	g.coverImageID = imageID
	g.updatedAt = time.Now().UTC()

	var eventImageID gallery.ImageID
	if imageID != nil {
		eventImageID = *imageID
	}

	g.addEvent(&GroupCoverImageChanged{
		BaseEvent:    shared.NewBaseEvent("community.group.cover_image_changed", g.id.String()),
		GroupID:      g.id,
		CoverImageID: eventImageID,
	})
}

// IncrementMemberCount increments the member count.
// This is typically called when a new member joins.
func (g *Group) IncrementMemberCount() {
	g.memberCount++
	g.updatedAt = time.Now().UTC()
}

// DecrementMemberCount decrements the member count.
// This is typically called when a member leaves or is removed.
func (g *Group) DecrementMemberCount() {
	if g.memberCount > 0 {
		g.memberCount--
		g.updatedAt = time.Now().UTC()
	}
}

// IncrementImageCount increments the image count.
// This is typically called when an image is shared to the group.
func (g *Group) IncrementImageCount() {
	g.imageCount++
	g.updatedAt = time.Now().UTC()
}

// DecrementImageCount decrements the image count.
// This is typically called when an image is removed from the group.
func (g *Group) DecrementImageCount() {
	if g.imageCount > 0 {
		g.imageCount--
		g.updatedAt = time.Now().UTC()
	}
}

// IncrementAlbumCount increments the album count.
// This is typically called when a group album is created.
func (g *Group) IncrementAlbumCount() {
	g.albumCount++
	g.updatedAt = time.Now().UTC()
}

// DecrementAlbumCount decrements the album count.
// This is typically called when a group album is deleted.
func (g *Group) DecrementAlbumCount() {
	if g.albumCount > 0 {
		g.albumCount--
		g.updatedAt = time.Now().UTC()
	}
}

// CanAcceptNewMembers returns true if the group can accept new members.
// Checks against member limit if configured.
func (g *Group) CanAcceptNewMembers() bool {
	if !g.settings.HasMemberLimit() {
		return true // No limit
	}
	return g.memberCount < g.settings.MaxMembers()
}

// IsOwnedBy returns true if the group is owned by the given user.
func (g *Group) IsOwnedBy(userID identity.UserID) bool {
	return g.ownerID.Equals(userID)
}

// IsPublic returns true if the group is public.
func (g *Group) IsPublic() bool {
	return g.groupType == GroupTypePublic
}

// IsPrivate returns true if the group is private.
func (g *Group) IsPrivate() bool {
	return g.groupType == GroupTypePrivate
}

// IsInviteOnly returns true if the group is invite-only.
func (g *Group) IsInviteOnly() bool {
	return g.groupType == GroupTypeInviteOnly
}

// IsDiscoverable returns true if the group can be found via search/browse.
func (g *Group) IsDiscoverable() bool {
	return g.groupType.IsDiscoverable()
}

// AllowsInstantJoin returns true if users can join without approval.
func (g *Group) AllowsInstantJoin() bool {
	return g.groupType.AllowsInstantJoin()
}

// Helper Methods

// addEvent appends a domain event to the events slice.
func (g *Group) addEvent(event shared.DomainEvent) {
	g.events = append(g.events, event)
}
