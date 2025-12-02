package moderation

import "fmt"

// ReviewAction represents the action taken by a moderator when reviewing a report.
type ReviewAction string

const (
	// ActionDismiss indicates the report was dismissed as invalid or unfounded.
	ActionDismiss ReviewAction = "dismiss"
	// ActionWarn indicates the user was given a warning.
	ActionWarn ReviewAction = "warn"
	// ActionRemove indicates the content was removed.
	ActionRemove ReviewAction = "remove"
	// ActionBan indicates the user was banned.
	ActionBan ReviewAction = "ban"
)

// ParseReviewAction creates a ReviewAction from a string value.
// Returns an error if the string is not a valid review action.
func ParseReviewAction(s string) (ReviewAction, error) {
	action := ReviewAction(s)
	if !action.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidReviewAction, s)
	}
	return action, nil
}

// String returns the string representation of the ReviewAction.
func (a ReviewAction) String() string {
	return string(a)
}

// IsValid returns true if the ReviewAction is a valid action value.
func (a ReviewAction) IsValid() bool {
	switch a {
	case ActionDismiss, ActionWarn, ActionRemove, ActionBan:
		return true
	default:
		return false
	}
}
