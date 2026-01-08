package moderation

import "fmt"

// NSFWProvider represents the NSFW detection API provider.
type NSFWProvider string

const (
	// ProviderSightEngine uses the SightEngine API for NSFW detection.
	ProviderSightEngine NSFWProvider = "sightengine"
	// ProviderModerateContent uses the ModerateContent API as a fallback.
	ProviderModerateContent NSFWProvider = "moderatecontent"
)

// ParseNSFWProvider creates an NSFWProvider from a string value.
// Returns an error if the string is not a valid provider.
func ParseNSFWProvider(s string) (NSFWProvider, error) {
	provider := NSFWProvider(s)
	if !provider.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidNSFWProvider, s)
	}
	return provider, nil
}

// String returns the string representation of the NSFWProvider.
func (p NSFWProvider) String() string {
	return string(p)
}

// IsValid returns true if the NSFWProvider is a valid provider value.
func (p NSFWProvider) IsValid() bool {
	switch p {
	case ProviderSightEngine, ProviderModerateContent:
		return true
	default:
		return false
	}
}
