package moderation

import (
	"fmt"

	"github.com/google/uuid"
)

// ReportID is a value object representing a unique report identifier.
// It wraps a UUID to provide type safety and prevent mixing with other ID types.
type ReportID struct {
	value uuid.UUID
}

// NewReportID creates a new ReportID with a generated UUID.
func NewReportID() ReportID {
	return ReportID{value: uuid.New()}
}

// ParseReportID parses a string into a ReportID.
// Returns an error if the string is not a valid UUID.
func ParseReportID(s string) (ReportID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ReportID{}, fmt.Errorf("invalid report id: %w", err)
	}
	return ReportID{value: id}, nil
}

// MustParseReportID parses a string into a ReportID and panics on error.
// Only use in tests or when the input is guaranteed to be valid.
func MustParseReportID(s string) ReportID {
	id, err := ParseReportID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the ReportID.
func (id ReportID) String() string {
	return id.value.String()
}

// IsZero returns true if this is the zero value (nil UUID).
func (id ReportID) IsZero() bool {
	return id.value == uuid.Nil
}

// Equals returns true if this ReportID equals the other ReportID.
func (id ReportID) Equals(other ReportID) bool {
	return id.value == other.value
}
