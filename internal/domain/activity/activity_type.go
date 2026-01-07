package activity

import "fmt"

// ActivityType represents the type of activity that occurred.
type ActivityType string

const (
	// ActivityTypeImageUploaded represents an image upload activity.
	ActivityTypeImageUploaded ActivityType = "image_uploaded"

	// ActivityTypeImageLiked represents an image like activity.
	ActivityTypeImageLiked ActivityType = "image_liked"

	// ActivityTypeImageCommented represents an image comment activity.
	ActivityTypeImageCommented ActivityType = "image_commented"

	// ActivityTypeUserFollowed represents a user follow activity.
	ActivityTypeUserFollowed ActivityType = "user_followed"

	// ActivityTypeAlbumCreated represents an album creation activity.
	ActivityTypeAlbumCreated ActivityType = "album_created"
)

// Valid returns true if the activity type is valid.
func (t ActivityType) Valid() bool {
	switch t {
	case ActivityTypeImageUploaded,
		ActivityTypeImageLiked,
		ActivityTypeImageCommented,
		ActivityTypeUserFollowed,
		ActivityTypeAlbumCreated:
		return true
	default:
		return false
	}
}

// String returns the string representation of the activity type.
func (t ActivityType) String() string {
	return string(t)
}

// ParseActivityType creates an ActivityType from a string.
func ParseActivityType(s string) (ActivityType, error) {
	t := ActivityType(s)
	if !t.Valid() {
		return "", fmt.Errorf("invalid activity type: %s", s)
	}
	return t, nil
}
