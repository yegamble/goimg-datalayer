package moderation_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestParseReviewAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    moderation.ReviewAction
		wantErr error
	}{
		{
			name:    "dismiss",
			input:   "dismiss",
			want:    moderation.ActionDismiss,
			wantErr: nil,
		},
		{
			name:    "warn",
			input:   "warn",
			want:    moderation.ActionWarn,
			wantErr: nil,
		},
		{
			name:    "remove",
			input:   "remove",
			want:    moderation.ActionRemove,
			wantErr: nil,
		},
		{
			name:    "ban",
			input:   "ban",
			want:    moderation.ActionBan,
			wantErr: nil,
		},
		{
			name:    "invalid",
			input:   "invalid",
			want:    "",
			wantErr: moderation.ErrInvalidReviewAction,
		},
		{
			name:    "empty",
			input:   "",
			want:    "",
			wantErr: moderation.ErrInvalidReviewAction,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			action, err := moderation.ParseReviewAction(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				assert.Empty(t, action)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, action)
			}
		})
	}
}

func TestReviewAction_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action moderation.ReviewAction
		want   string
	}{
		{
			name:   "dismiss",
			action: moderation.ActionDismiss,
			want:   "dismiss",
		},
		{
			name:   "warn",
			action: moderation.ActionWarn,
			want:   "warn",
		},
		{
			name:   "remove",
			action: moderation.ActionRemove,
			want:   "remove",
		},
		{
			name:   "ban",
			action: moderation.ActionBan,
			want:   "ban",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.action.String())
		})
	}
}

func TestReviewAction_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action moderation.ReviewAction
		want   bool
	}{
		{
			name:   "dismiss is valid",
			action: moderation.ActionDismiss,
			want:   true,
		},
		{
			name:   "warn is valid",
			action: moderation.ActionWarn,
			want:   true,
		},
		{
			name:   "remove is valid",
			action: moderation.ActionRemove,
			want:   true,
		},
		{
			name:   "ban is valid",
			action: moderation.ActionBan,
			want:   true,
		},
		{
			name:   "invalid action",
			action: moderation.ReviewAction("invalid"),
			want:   false,
		},
		{
			name:   "empty action",
			action: moderation.ReviewAction(""),
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.action.IsValid())
		})
	}
}
