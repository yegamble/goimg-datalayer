package moderation_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestParseReportStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    moderation.ReportStatus
		wantErr error
	}{
		{
			name:    "pending",
			input:   "pending",
			want:    moderation.StatusPending,
			wantErr: nil,
		},
		{
			name:    "reviewing",
			input:   "reviewing",
			want:    moderation.StatusReviewing,
			wantErr: nil,
		},
		{
			name:    "resolved",
			input:   "resolved",
			want:    moderation.StatusResolved,
			wantErr: nil,
		},
		{
			name:    "dismissed",
			input:   "dismissed",
			want:    moderation.StatusDismissed,
			wantErr: nil,
		},
		{
			name:    "invalid",
			input:   "invalid",
			want:    "",
			wantErr: moderation.ErrInvalidReportStatus,
		},
		{
			name:    "empty",
			input:   "",
			want:    "",
			wantErr: moderation.ErrInvalidReportStatus,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, err := moderation.ParseReportStatus(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				assert.Empty(t, status)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, status)
			}
		})
	}
}

func TestReportStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status moderation.ReportStatus
		want   string
	}{
		{
			name:   "pending",
			status: moderation.StatusPending,
			want:   "pending",
		},
		{
			name:   "reviewing",
			status: moderation.StatusReviewing,
			want:   "reviewing",
		},
		{
			name:   "resolved",
			status: moderation.StatusResolved,
			want:   "resolved",
		},
		{
			name:   "dismissed",
			status: moderation.StatusDismissed,
			want:   "dismissed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestReportStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status moderation.ReportStatus
		want   bool
	}{
		{
			name:   "pending is valid",
			status: moderation.StatusPending,
			want:   true,
		},
		{
			name:   "reviewing is valid",
			status: moderation.StatusReviewing,
			want:   true,
		},
		{
			name:   "resolved is valid",
			status: moderation.StatusResolved,
			want:   true,
		},
		{
			name:   "dismissed is valid",
			status: moderation.StatusDismissed,
			want:   true,
		},
		{
			name:   "invalid status",
			status: moderation.ReportStatus("invalid"),
			want:   false,
		},
		{
			name:   "empty status",
			status: moderation.ReportStatus(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestReportStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status moderation.ReportStatus
		want   bool
	}{
		{
			name:   "pending is not terminal",
			status: moderation.StatusPending,
			want:   false,
		},
		{
			name:   "reviewing is not terminal",
			status: moderation.StatusReviewing,
			want:   false,
		},
		{
			name:   "resolved is terminal",
			status: moderation.StatusResolved,
			want:   true,
		},
		{
			name:   "dismissed is terminal",
			status: moderation.StatusDismissed,
			want:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.IsTerminal())
		})
	}
}
