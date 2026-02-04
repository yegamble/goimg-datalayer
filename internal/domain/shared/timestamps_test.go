package shared_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNow(t *testing.T) {
	t.Parallel()

	now := shared.Now()
	assert.False(t, now.IsZero())
	assert.Equal(t, time.UTC, now.Location())
	assert.WithinDuration(t, time.Now().UTC(), now, time.Second)
}

func TestParseISO8601(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "RFC3339",
			input:   "2023-10-25T12:00:00Z",
			want:    time.Date(2023, 10, 25, 12, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "RFC3339Nano",
			input:   "2023-10-25T12:00:00.123456Z",
			want:    time.Date(2023, 10, 25, 12, 0, 0, 123456000, time.UTC),
			wantErr: false,
		},
		{
			name:    "RFC3339 with offset",
			input:   "2023-10-25T14:00:00+02:00",
			want:    time.Date(2023, 10, 25, 12, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "2023/10/25",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := shared.ParseISO8601(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, time.UTC, got.Location())
		})
	}
}

func TestFormatISO8601(t *testing.T) {
	t.Parallel()

	ts := time.Date(2023, 10, 25, 12, 0, 0, 0, time.UTC)
	expected := "2023-10-25T12:00:00Z"

	assert.Equal(t, expected, shared.FormatISO8601(ts))

	// Test non-UTC conversion
	loc, _ := time.LoadLocation("America/New_York")
	tsNY := ts.In(loc)
	assert.Equal(t, expected, shared.FormatISO8601(tsNY))
}
