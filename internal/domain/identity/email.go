package identity

import (
	"regexp"
	"strings"
)

// Email is a value object representing a validated email address.
type Email struct {
	value string
}

var (
	// emailRegex validates basic email format following RFC 5322 simplified pattern.
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// disposableEmailDomains contains common disposable email providers.
	// In production, this should be loaded from a comprehensive external list.
	disposableEmailDomains = map[string]bool{
		"mailinator.com":    true,
		"tempmail.com":      true,
		"guerrillamail.com": true,
		"10minutemail.com":  true,
		"throwaway.email":   true,
		"temp-mail.org":     true,
		"yopmail.com":       true,
		"fakeinbox.com":     true,
		"trashmail.com":     true,
		"getnada.com":       true,
	}
)

// NewEmail creates a new Email value object after validating the input.
// The email is normalized: trimmed, lowercased.
func NewEmail(value string) (Email, error) {
	// Normalize: trim whitespace and lowercase
	value = strings.TrimSpace(strings.ToLower(value))

	// Validate
	if value == "" {
		return Email{}, ErrEmailEmpty
	}

	if len(value) > 255 {
		return Email{}, ErrEmailTooLong
	}

	if !emailRegex.MatchString(value) {
		return Email{}, ErrEmailInvalid
	}

	// Check for disposable email domains
	domain := extractDomain(value)
	if disposableEmailDomains[domain] {
		return Email{}, ErrEmailDisposable
	}

	return Email{value: value}, nil
}

// String returns the string representation of the email address.
func (e Email) String() string {
	return e.value
}

// Domain returns the domain part of the email address.
func (e Email) Domain() string {
	return extractDomain(e.value)
}

// IsEmpty returns true if the email is the zero value.
func (e Email) IsEmpty() bool {
	return e.value == ""
}

// Equals returns true if this Email equals the other Email.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// extractDomain extracts the domain part from an email address.
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
