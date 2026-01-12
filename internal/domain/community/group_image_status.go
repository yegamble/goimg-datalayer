package community

// GroupImageStatus represents the approval status of an image shared to a group.
type GroupImageStatus string

const (
	// GroupImageStatusPending indicates the image is awaiting moderation.
	GroupImageStatusPending GroupImageStatus = "pending"

	// GroupImageStatusApproved indicates the image has been approved by a moderator.
	GroupImageStatusApproved GroupImageStatus = "approved"

	// GroupImageStatusRejected indicates the image has been rejected by a moderator.
	GroupImageStatusRejected GroupImageStatus = "rejected"
)

// String returns the string representation of the GroupImageStatus.
func (s GroupImageStatus) String() string {
	return string(s)
}

// IsValid returns true if the status is a valid GroupImageStatus value.
func (s GroupImageStatus) IsValid() bool {
	switch s {
	case GroupImageStatusPending, GroupImageStatusApproved, GroupImageStatusRejected:
		return true
	default:
		return false
	}
}

// IsPending returns true if the status is pending.
func (s GroupImageStatus) IsPending() bool {
	return s == GroupImageStatusPending
}

// IsApproved returns true if the status is approved.
func (s GroupImageStatus) IsApproved() bool {
	return s == GroupImageStatusApproved
}

// IsRejected returns true if the status is rejected.
func (s GroupImageStatus) IsRejected() bool {
	return s == GroupImageStatusRejected
}

// AllGroupImageStatuses returns all valid group image statuses.
func AllGroupImageStatuses() []GroupImageStatus {
	return []GroupImageStatus{
		GroupImageStatusPending,
		GroupImageStatusApproved,
		GroupImageStatusRejected,
	}
}
