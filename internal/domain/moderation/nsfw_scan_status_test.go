package moderation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/moderation"
)

func TestParseNSFWScanStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected moderation.NSFWScanStatus
		wantErr  bool
	}{
		{
			name:     "pending status",
			input:    "pending",
			expected: moderation.ScanStatusPending,
			wantErr:  false,
		},
		{
			name:     "scanning status",
			input:    "scanning",
			expected: moderation.ScanStatusScanning,
			wantErr:  false,
		},
		{
			name:     "completed status",
			input:    "completed",
			expected: moderation.ScanStatusCompleted,
			wantErr:  false,
		},
		{
			name:     "failed status",
			input:    "failed",
			expected: moderation.ScanStatusFailed,
			wantErr:  false,
		},
		{
			name:     "invalid status",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := moderation.ParseNSFWScanStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, moderation.ErrInvalidNSFWScanStatus)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, status)
			}
		})
	}
}

func TestNSFWScanStatus_String(t *testing.T) {
	tests := []struct {
		status   moderation.NSFWScanStatus
		expected string
	}{
		{moderation.ScanStatusPending, "pending"},
		{moderation.ScanStatusScanning, "scanning"},
		{moderation.ScanStatusCompleted, "completed"},
		{moderation.ScanStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestNSFWScanStatus_IsValid(t *testing.T) {
	tests := []struct {
		status moderation.NSFWScanStatus
		valid  bool
	}{
		{moderation.ScanStatusPending, true},
		{moderation.ScanStatusScanning, true},
		{moderation.ScanStatusCompleted, true},
		{moderation.ScanStatusFailed, true},
		{moderation.NSFWScanStatus("invalid"), false},
		{moderation.NSFWScanStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

func TestNSFWScanStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status     moderation.NSFWScanStatus
		isTerminal bool
	}{
		{moderation.ScanStatusPending, false},
		{moderation.ScanStatusScanning, false},
		{moderation.ScanStatusCompleted, true},
		{moderation.ScanStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isTerminal, tt.status.IsTerminal())
		})
	}
}

func TestNSFWScanStatus_IsActive(t *testing.T) {
	tests := []struct {
		status   moderation.NSFWScanStatus
		isActive bool
	}{
		{moderation.ScanStatusPending, true},
		{moderation.ScanStatusScanning, true},
		{moderation.ScanStatusCompleted, false},
		{moderation.ScanStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isActive, tt.status.IsActive())
		})
	}
}
