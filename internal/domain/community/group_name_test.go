package community

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
		wantVal string
	}{
		{
			name:    "valid name",
			input:   "Landscape Photography",
			wantErr: nil,
			wantVal: "Landscape Photography",
		},
		{
			name:    "minimum length",
			input:   "Art",
			wantErr: nil,
			wantVal: "Art",
		},
		{
			name:    "maximum length",
			input:   strings.Repeat("a", 100),
			wantErr: nil,
			wantVal: strings.Repeat("a", 100),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrGroupNameRequired,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrGroupNameRequired,
		},
		{
			name:    "too short",
			input:   "AB",
			wantErr: ErrGroupNameTooShort,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 101),
			wantErr: ErrGroupNameTooLong,
		},
		{
			name:    "trimmed whitespace",
			input:   "  Photography  ",
			wantErr: nil,
			wantVal: "Photography",
		},
		{
			name:    "collapsed spaces",
			input:   "Landscape    Photography",
			wantErr: nil,
			wantVal: "Landscape Photography",
		},
		{
			name:    "multiple spaces collapsed",
			input:   "Nature   and   Wildlife",
			wantErr: nil,
			wantVal: "Nature and Wildlife",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, err := NewGroupName(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.True(t, name.IsEmpty())
			} else {
				require.NoError(t, err)
				assert.False(t, name.IsEmpty())
				assert.Equal(t, tt.wantVal, name.String())
			}
		})
	}
}

func TestGroupName_Equals(t *testing.T) {
	t.Parallel()

	name1, _ := NewGroupName("Photography")
	name2, _ := NewGroupName("Photography")
	name3, _ := NewGroupName("Art")

	assert.True(t, name1.Equals(name2))
	assert.False(t, name1.Equals(name3))
}
