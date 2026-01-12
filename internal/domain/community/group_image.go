package community

import (
	"fmt"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// GroupImage represents an image shared to a group's image pool.
// Images may require approval before being visible to group members.
type GroupImage struct {
	id         GroupImageID
	groupID    GroupID
	imageID    gallery.ImageID
	sharedBy   identity.UserID
	status     GroupImageStatus
	reviewedBy *identity.UserID
	sharedAt   time.Time
	reviewedAt *time.Time
}

// NewGroupImage creates a new GroupImage with pending status.
// The image will need approval if the group has require_approval enabled.
func NewGroupImage(
	groupID GroupID,
	imageID gallery.ImageID,
	sharedBy identity.UserID,
	requiresApproval bool,
) (*GroupImage, error) {
	if groupID.IsZero() {
		return nil, fmt.Errorf("group id cannot be empty")
	}

	if imageID.IsZero() {
		return nil, fmt.Errorf("image id cannot be empty")
	}

	if sharedBy.IsZero() {
		return nil, fmt.Errorf("shared by user id cannot be empty")
	}

	// If approval is not required, auto-approve the image
	status := GroupImageStatusPending
	if !requiresApproval {
		status = GroupImageStatusApproved
	}

	return &GroupImage{
		id:       NewGroupImageID(),
		groupID:  groupID,
		imageID:  imageID,
		sharedBy: sharedBy,
		status:   status,
		sharedAt: time.Now().UTC(),
	}, nil
}

// ReconstructGroupImage reconstitutes a GroupImage from persistence.
// This bypasses validation and should only be used by the infrastructure layer.
func ReconstructGroupImage(
	id GroupImageID,
	groupID GroupID,
	imageID gallery.ImageID,
	sharedBy identity.UserID,
	status GroupImageStatus,
	reviewedBy *identity.UserID,
	sharedAt time.Time,
	reviewedAt *time.Time,
) *GroupImage {
	return &GroupImage{
		id:         id,
		groupID:    groupID,
		imageID:    imageID,
		sharedBy:   sharedBy,
		status:     status,
		reviewedBy: reviewedBy,
		sharedAt:   sharedAt,
		reviewedAt: reviewedAt,
	}
}

// ID returns the group image's unique identifier.
func (gi *GroupImage) ID() GroupImageID {
	return gi.id
}

// GroupID returns the group this image belongs to.
func (gi *GroupImage) GroupID() GroupID {
	return gi.groupID
}

// ImageID returns the ID of the shared image.
func (gi *GroupImage) ImageID() gallery.ImageID {
	return gi.imageID
}

// SharedBy returns the user who shared the image.
func (gi *GroupImage) SharedBy() identity.UserID {
	return gi.sharedBy
}

// Status returns the current approval status.
func (gi *GroupImage) Status() GroupImageStatus {
	return gi.status
}

// ReviewedBy returns the user who reviewed the image (if reviewed).
func (gi *GroupImage) ReviewedBy() *identity.UserID {
	return gi.reviewedBy
}

// SharedAt returns when the image was shared.
func (gi *GroupImage) SharedAt() time.Time {
	return gi.sharedAt
}

// ReviewedAt returns when the image was reviewed (if reviewed).
func (gi *GroupImage) ReviewedAt() *time.Time {
	return gi.reviewedAt
}

// IsPending returns true if the image is awaiting moderation.
func (gi *GroupImage) IsPending() bool {
	return gi.status.IsPending()
}

// IsApproved returns true if the image has been approved.
func (gi *GroupImage) IsApproved() bool {
	return gi.status.IsApproved()
}

// IsRejected returns true if the image has been rejected.
func (gi *GroupImage) IsRejected() bool {
	return gi.status.IsRejected()
}

// Approve approves the image for display in the group.
// Returns an error if the image is not in pending status.
func (gi *GroupImage) Approve(reviewerID identity.UserID) error {
	if reviewerID.IsZero() {
		return fmt.Errorf("reviewer id cannot be empty")
	}

	if !gi.status.IsPending() {
		return fmt.Errorf("cannot approve image: current status is %s", gi.status)
	}

	now := time.Now().UTC()
	gi.status = GroupImageStatusApproved
	gi.reviewedBy = &reviewerID
	gi.reviewedAt = &now

	return nil
}

// Reject rejects the image from being displayed in the group.
// Returns an error if the image is not in pending status.
func (gi *GroupImage) Reject(reviewerID identity.UserID) error {
	if reviewerID.IsZero() {
		return fmt.Errorf("reviewer id cannot be empty")
	}

	if !gi.status.IsPending() {
		return fmt.Errorf("cannot reject image: current status is %s", gi.status)
	}

	now := time.Now().UTC()
	gi.status = GroupImageStatusRejected
	gi.reviewedBy = &reviewerID
	gi.reviewedAt = &now

	return nil
}
