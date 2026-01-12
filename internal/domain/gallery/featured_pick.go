package gallery

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

const (
	// MaxFeaturingReason is the maximum length for featuring reason notes.
	MaxFeaturingReason = 500
)

// FeaturedPick represents an admin-curated featured image.
// This is an entity within the gallery bounded context that manages
// which images are featured on the explore page with scheduling support.
type FeaturedPick struct {
	id            FeaturedPickID
	imageID       ImageID
	featuredBy    identity.UserID
	reason        string     // Optional reason/notes for featuring
	displayOrder  int        // Display priority (0 = highest)
	featuredFrom  time.Time
	featuredUntil *time.Time // nil = no expiration
	createdAt     time.Time
	updatedAt     time.Time
}

// NewFeaturedPick creates a new FeaturedPick with immediate featuring.
// The image is featured from now with no expiration.
func NewFeaturedPick(
	imageID ImageID,
	featuredBy identity.UserID,
	reason string,
	displayOrder int,
) (*FeaturedPick, error) {
	return NewFeaturedPickWithSchedule(
		imageID,
		featuredBy,
		reason,
		displayOrder,
		time.Now().UTC(),
		nil, // No expiration
	)
}

// NewFeaturedPickWithSchedule creates a new FeaturedPick with scheduling.
// This allows admins to schedule when an image should be featured.
func NewFeaturedPickWithSchedule(
	imageID ImageID,
	featuredBy identity.UserID,
	reason string,
	displayOrder int,
	featuredFrom time.Time,
	featuredUntil *time.Time,
) (*FeaturedPick, error) {
	if imageID.IsZero() {
		return nil, fmt.Errorf("%w: image ID is required", ErrInvalidMetadata)
	}
	if featuredBy.IsZero() {
		return nil, fmt.Errorf("%w: featured_by user ID is required", ErrInvalidMetadata)
	}
	if displayOrder < 0 {
		return nil, fmt.Errorf("%w: display order must be non-negative", ErrInvalidMetadata)
	}
	if len(reason) > MaxFeaturingReason {
		return nil, ErrReasonTooLong
	}
	if featuredUntil != nil && featuredUntil.Before(featuredFrom) {
		return nil, fmt.Errorf("%w: featured_until must be after featured_from", ErrInvalidMetadata)
	}

	now := time.Now().UTC()
	return &FeaturedPick{
		id:            NewFeaturedPickID(),
		imageID:       imageID,
		featuredBy:    featuredBy,
		reason:        reason,
		displayOrder:  displayOrder,
		featuredFrom:  featuredFrom.UTC(),
		featuredUntil: featuredUntil,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// ReconstructFeaturedPick reconstitutes a FeaturedPick from persistence.
// Use this only when loading from the database.
func ReconstructFeaturedPick(
	id FeaturedPickID,
	imageID ImageID,
	featuredBy identity.UserID,
	reason string,
	displayOrder int,
	featuredFrom time.Time,
	featuredUntil *time.Time,
	createdAt, updatedAt time.Time,
) *FeaturedPick {
	return &FeaturedPick{
		id:            id,
		imageID:       imageID,
		featuredBy:    featuredBy,
		reason:        reason,
		displayOrder:  displayOrder,
		featuredFrom:  featuredFrom,
		featuredUntil: featuredUntil,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

// ID returns the FeaturedPick's unique identifier.
func (f *FeaturedPick) ID() FeaturedPickID { return f.id }

// ImageID returns the ID of the featured image.
func (f *FeaturedPick) ImageID() ImageID { return f.imageID }

// FeaturedBy returns the ID of the admin who featured this image.
func (f *FeaturedPick) FeaturedBy() identity.UserID { return f.featuredBy }

// Reason returns the optional reason/notes for featuring.
func (f *FeaturedPick) Reason() string { return f.reason }

// DisplayOrder returns the display priority (0 = highest).
func (f *FeaturedPick) DisplayOrder() int { return f.displayOrder }

// FeaturedFrom returns the start date/time for featuring.
func (f *FeaturedPick) FeaturedFrom() time.Time { return f.featuredFrom }

// FeaturedUntil returns the end date/time for featuring (nil = no expiration).
func (f *FeaturedPick) FeaturedUntil() *time.Time { return f.featuredUntil }

// CreatedAt returns when the featured pick was created.
func (f *FeaturedPick) CreatedAt() time.Time { return f.createdAt }

// UpdatedAt returns when the featured pick was last updated.
func (f *FeaturedPick) UpdatedAt() time.Time { return f.updatedAt }

// IsActive returns true if the featured pick is currently active.
func (f *FeaturedPick) IsActive() bool {
	now := time.Now().UTC()
	if now.Before(f.featuredFrom) {
		return false // Not started yet
	}
	if f.featuredUntil != nil && now.After(*f.featuredUntil) {
		return false // Expired
	}
	return true
}

// UpdateSchedule updates the featuring schedule.
// This allows admins to extend or shorten the featuring period.
func (f *FeaturedPick) UpdateSchedule(featuredFrom time.Time, featuredUntil *time.Time) error {
	if featuredUntil != nil && featuredUntil.Before(featuredFrom) {
		return fmt.Errorf("%w: featured_until must be after featured_from", ErrInvalidMetadata)
	}

	f.featuredFrom = featuredFrom.UTC()
	f.featuredUntil = featuredUntil
	f.updatedAt = time.Now().UTC()

	return nil
}

// UpdateDisplayOrder updates the display priority.
func (f *FeaturedPick) UpdateDisplayOrder(order int) error {
	if order < 0 {
		return fmt.Errorf("%w: display order must be non-negative", ErrInvalidMetadata)
	}

	f.displayOrder = order
	f.updatedAt = time.Now().UTC()

	return nil
}

// Expire immediately expires the featured pick.
// This is equivalent to setting featured_until to now.
func (f *FeaturedPick) Expire() {
	now := time.Now().UTC()
	f.featuredUntil = &now
	f.updatedAt = now
}
