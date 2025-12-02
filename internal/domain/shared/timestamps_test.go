package shared_test

import (
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNow(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC()
	result := shared.Now()
	after := time.Now().UTC()

	// Verify result is between before and after
	if result.Before(before) || result.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", result, before, after)
	}

	// Verify result is in UTC
	if result.Location() != time.UTC {
		t.Errorf("Now() location = %v, want UTC", result.Location())
	}
}

func TestParseISO8601(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		verify  func(t *testing.T, result time.Time)
	}{
		{
			name:    "RFC3339 format",
			input:   "2023-12-01T15:04:05Z",
			wantErr: false,
			verify: func(t *testing.T, result time.Time) {
				t.Helper()
				expected := time.Date(2023, 12, 1, 15, 4, 5, 0, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("result = %v, want %v", result, expected)
				}
			},
		},
		{
			name:    "RFC3339 with timezone offset",
			input:   "2023-12-01T15:04:05+05:00",
			wantErr: false,
			verify: func(t *testing.T, result time.Time) {
				t.Helper()
				// Should be converted to UTC (10:04:05 UTC)
				expected := time.Date(2023, 12, 1, 10, 4, 5, 0, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("result = %v, want %v", result, expected)
				}
				if result.Location() != time.UTC {
					t.Errorf("location = %v, want UTC", result.Location())
				}
			},
		},
		{
			name:    "RFC3339 with negative timezone offset",
			input:   "2023-12-01T15:04:05-05:00",
			wantErr: false,
			verify: func(t *testing.T, result time.Time) {
				t.Helper()
				// Should be converted to UTC (20:04:05 UTC)
				expected := time.Date(2023, 12, 1, 20, 4, 5, 0, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("result = %v, want %v", result, expected)
				}
				if result.Location() != time.UTC {
					t.Errorf("location = %v, want UTC", result.Location())
				}
			},
		},
		{
			name:    "RFC3339Nano format",
			input:   "2023-12-01T15:04:05.123456789Z",
			wantErr: false,
			verify: func(t *testing.T, result time.Time) {
				t.Helper()
				expected := time.Date(2023, 12, 1, 15, 4, 5, 123456789, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("result = %v, want %v", result, expected)
				}
			},
		},
		{
			name:    "RFC3339Nano with timezone",
			input:   "2023-12-01T15:04:05.123456789+02:00",
			wantErr: false,
			verify: func(t *testing.T, result time.Time) {
				t.Helper()
				// Should be converted to UTC (13:04:05.123456789 UTC)
				expected := time.Date(2023, 12, 1, 13, 4, 5, 123456789, time.UTC)
				if !result.Equal(expected) {
					t.Errorf("result = %v, want %v", result, expected)
				}
				if result.Location() != time.UTC {
					t.Errorf("location = %v, want UTC", result.Location())
				}
			},
		},
		{
			name:    "invalid format",
			input:   "2023-12-01 15:04:05",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid date",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "partial date",
			input:   "2023-12-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := shared.ParseISO8601(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseISO8601() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseISO8601() unexpected error = %v", err)
				return
			}

			if tt.verify != nil {
				tt.verify(t, result)
			}
		})
	}
}

func TestFormatISO8601(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "UTC time",
			input: time.Date(2023, 12, 1, 15, 4, 5, 0, time.UTC),
			want:  "2023-12-01T15:04:05Z",
		},
		{
			name:  "time with timezone offset converted to UTC",
			input: time.Date(2023, 12, 1, 15, 4, 5, 0, time.FixedZone("Test", 5*3600)),
			want:  "2023-12-01T10:04:05Z", // Converted to UTC (15:04 - 5:00 = 10:04)
		},
		{
			name:  "zero time",
			input: time.Time{},
			want:  "0001-01-01T00:00:00Z",
		},
		{
			name:  "time with nanoseconds truncated",
			input: time.Date(2023, 12, 1, 15, 4, 5, 123456789, time.UTC),
			want:  "2023-12-01T15:04:05Z", // Nanoseconds not included in RFC3339
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := shared.FormatISO8601(tt.input)

			if result != tt.want {
				t.Errorf("FormatISO8601() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestFormatISO8601_ParseISO8601_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
	}{
		{
			name:  "UTC time",
			input: time.Date(2023, 12, 1, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "different date",
			input: time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
		},
		{
			name:  "midnight",
			input: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Format then parse
			formatted := shared.FormatISO8601(tt.input)
			parsed, err := shared.ParseISO8601(formatted)

			if err != nil {
				t.Errorf("ParseISO8601() error = %v", err)
				return
			}

			// Note: Nanoseconds are lost in RFC3339 format, so truncate to seconds
			expectedTruncated := tt.input.Truncate(time.Second)
			parsedTruncated := parsed.Truncate(time.Second)

			if !parsedTruncated.Equal(expectedTruncated) {
				t.Errorf("round trip: got %v, want %v", parsedTruncated, expectedTruncated)
			}

			// Verify still in UTC
			if parsed.Location() != time.UTC {
				t.Errorf("parsed location = %v, want UTC", parsed.Location())
			}
		})
	}
}

func TestTimestampFunctions_AlwaysReturnUTC(t *testing.T) {
	t.Parallel()

	// Test Now() returns UTC
	now := shared.Now()
	if now.Location() != time.UTC {
		t.Errorf("Now() location = %v, want UTC", now.Location())
	}

	// Test ParseISO8601 returns UTC
	parsed, err := shared.ParseISO8601("2023-12-01T15:04:05+05:00")
	if err != nil {
		t.Fatalf("ParseISO8601() error = %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("ParseISO8601() location = %v, want UTC", parsed.Location())
	}

	// Test FormatISO8601 converts to UTC
	nonUTC := time.Date(2023, 12, 1, 15, 4, 5, 0, time.FixedZone("Test", 3600))
	formatted := shared.FormatISO8601(nonUTC)
	// Should end with Z indicating UTC
	if len(formatted) == 0 || formatted[len(formatted)-1] != 'Z' {
		t.Errorf("FormatISO8601() = %v, want to end with Z", formatted)
	}
}
