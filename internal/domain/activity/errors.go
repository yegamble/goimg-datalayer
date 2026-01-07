package activity

import "errors"

var (
	// ErrActorRequired is returned when the actor ID is not provided.
	ErrActorRequired = errors.New("actor ID is required")

	// ErrInvalidActivityType is returned when the activity type is invalid.
	ErrInvalidActivityType = errors.New("invalid activity type")

	// ErrTargetRequired is returned when the target ID is not provided.
	ErrTargetRequired = errors.New("target ID is required")

	// ErrInvalidTargetType is returned when the target type is invalid.
	ErrInvalidTargetType = errors.New("invalid target type")

	// ErrActivityNotFound is returned when an activity is not found.
	ErrActivityNotFound = errors.New("activity not found")
)
