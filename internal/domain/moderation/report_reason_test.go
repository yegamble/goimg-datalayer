package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

//nolint:funlen // Table-driven test with comprehensive test cases
func TestParseReportReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    moderation.ReportReason
		wantErr error
	}{
		{
			name:    "spam",
			input:   "spam",
			want:    moderation.ReasonSpam,
			wantErr: nil,
		},
		{
			name:    "inappropriate",
			input:   "inappropriate",
			want:    moderation.ReasonInappropriate,
			wantErr: nil,
		},
		{
			name:    "copyright",
			input:   "copyright",
			want:    moderation.ReasonCopyright,
			wantErr: nil,
		},
		{
			name:    "harassment",
			input:   "harassment",
			want:    moderation.ReasonHarassment,
			wantErr: nil,
		},
		{
			name:    "other",
			input:   "other",
			want:    moderation.ReasonOther,
			wantErr: nil,
		},
		{
			name:    "invalid",
			input:   "invalid",
			want:    "",
			wantErr: moderation.ErrInvalidReportReason,
		},
		{
			name:    "empty",
			input:   "",
			want:    "",
			wantErr: moderation.ErrInvalidReportReason,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reason, err := moderation.ParseReportReason(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, reason)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, reason)
			}
		})
	}
}

func TestReportReason_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reason moderation.ReportReason
		want   string
	}{
		{
			name:   "spam",
			reason: moderation.ReasonSpam,
			want:   "spam",
		},
		{
			name:   "inappropriate",
			reason: moderation.ReasonInappropriate,
			want:   "inappropriate",
		},
		{
			name:   "copyright",
			reason: moderation.ReasonCopyright,
			want:   "copyright",
		},
		{
			name:   "harassment",
			reason: moderation.ReasonHarassment,
			want:   "harassment",
		},
		{
			name:   "other",
			reason: moderation.ReasonOther,
			want:   "other",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.reason.String())
		})
	}
}

func TestReportReason_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reason moderation.ReportReason
		want   bool
	}{
		{
			name:   "spam is valid",
			reason: moderation.ReasonSpam,
			want:   true,
		},
		{
			name:   "inappropriate is valid",
			reason: moderation.ReasonInappropriate,
			want:   true,
		},
		{
			name:   "copyright is valid",
			reason: moderation.ReasonCopyright,
			want:   true,
		},
		{
			name:   "harassment is valid",
			reason: moderation.ReasonHarassment,
			want:   true,
		},
		{
			name:   "other is valid",
			reason: moderation.ReasonOther,
			want:   true,
		},
		{
			name:   "invalid reason",
			reason: moderation.ReportReason("invalid"),
			want:   false,
		},
		{
			name:   "empty reason",
			reason: moderation.ReportReason(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.reason.IsValid())
		})
	}
}
