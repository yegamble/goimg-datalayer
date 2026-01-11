package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestDigestFrequency_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		frequency DigestFrequency
		want      bool
	}{
		{
			name:      "immediate is valid",
			frequency: DigestImmediate,
			want:      true,
		},
		{
			name:      "hourly is valid",
			frequency: DigestHourly,
			want:      true,
		},
		{
			name:      "daily is valid",
			frequency: DigestDaily,
			want:      true,
		},
		{
			name:      "weekly is valid",
			frequency: DigestWeekly,
			want:      true,
		},
		{
			name:      "empty is invalid",
			frequency: DigestFrequency(""),
			want:      false,
		},
		{
			name:      "random string is invalid",
			frequency: DigestFrequency("monthly"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.frequency.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDigestFrequency_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		frequency DigestFrequency
		want      string
	}{
		{
			name:      "immediate",
			frequency: DigestImmediate,
			want:      "immediate",
		},
		{
			name:      "hourly",
			frequency: DigestHourly,
			want:      "hourly",
		},
		{
			name:      "daily",
			frequency: DigestDaily,
			want:      "daily",
		},
		{
			name:      "weekly",
			frequency: DigestWeekly,
			want:      "weekly",
		},
		{
			name:      "empty",
			frequency: DigestFrequency(""),
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.frequency.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewNotificationPreferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		emailEnabled    bool
		emailTypes      map[shared.NotificationType]bool
		digestFrequency DigestFrequency
		wantFrequency   DigestFrequency
	}{
		{
			name:            "valid preferences with all fields",
			emailEnabled:    true,
			emailTypes:      map[shared.NotificationType]bool{shared.NotificationTypeNewFollower: true},
			digestFrequency: DigestHourly,
			wantFrequency:   DigestHourly,
		},
		{
			name:            "nil email types map creates empty map",
			emailEnabled:    false,
			emailTypes:      nil,
			digestFrequency: DigestDaily,
			wantFrequency:   DigestDaily,
		},
		{
			name:            "invalid frequency defaults to daily",
			emailEnabled:    true,
			emailTypes:      map[shared.NotificationType]bool{},
			digestFrequency: DigestFrequency("invalid"),
			wantFrequency:   DigestDaily,
		},
		{
			name:            "empty frequency defaults to daily",
			emailEnabled:    false,
			emailTypes:      map[shared.NotificationType]bool{},
			digestFrequency: DigestFrequency(""),
			wantFrequency:   DigestDaily,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefs := NewNotificationPreferences(tt.emailEnabled, tt.emailTypes, tt.digestFrequency)

			assert.Equal(t, tt.emailEnabled, prefs.EmailEnabled())
			assert.Equal(t, tt.wantFrequency, prefs.DigestFrequency())
			assert.NotNil(t, prefs.EmailTypes())

			if tt.emailTypes != nil {
				assert.Equal(t, len(tt.emailTypes), len(prefs.EmailTypes()))
			}
		})
	}
}

func TestNotificationPreferences_EmailEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "email enabled",
			enabled: true,
		},
		{
			name:    "email disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefs := NewNotificationPreferences(tt.enabled, nil, DigestDaily)
			assert.Equal(t, tt.enabled, prefs.EmailEnabled())
		})
	}
}

func TestNotificationPreferences_EmailTypes(t *testing.T) {
	t.Parallel()

	t.Run("returns copy of email types map", func(t *testing.T) {
		t.Parallel()

		originalTypes := map[shared.NotificationType]bool{
			shared.NotificationTypeNewFollower: true,
			shared.NotificationTypeNewPhotos:   false,
		}

		prefs := NewNotificationPreferences(true, originalTypes, DigestDaily)
		emailTypes := prefs.EmailTypes()

		// Verify we got a copy with same content
		assert.Equal(t, len(originalTypes), len(emailTypes))
		for k, v := range originalTypes {
			assert.Equal(t, v, emailTypes[k])
		}

		// Verify modifications don't affect original
		emailTypes[shared.NotificationTypeNewFollower] = false
		assert.NotEqual(t, emailTypes[shared.NotificationTypeNewFollower], prefs.EmailTypes()[shared.NotificationTypeNewFollower])
	})

	t.Run("empty map returns empty copy", func(t *testing.T) {
		t.Parallel()

		prefs := NewNotificationPreferences(false, map[shared.NotificationType]bool{}, DigestDaily)
		emailTypes := prefs.EmailTypes()

		assert.NotNil(t, emailTypes)
		assert.Equal(t, 0, len(emailTypes))
	})
}

func TestNotificationPreferences_DigestFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		frequency DigestFrequency
	}{
		{
			name:      "immediate frequency",
			frequency: DigestImmediate,
		},
		{
			name:      "hourly frequency",
			frequency: DigestHourly,
		},
		{
			name:      "daily frequency",
			frequency: DigestDaily,
		},
		{
			name:      "weekly frequency",
			frequency: DigestWeekly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefs := NewNotificationPreferences(true, nil, tt.frequency)
			assert.Equal(t, tt.frequency, prefs.DigestFrequency())
		})
	}
}

func TestNotificationPreferences_ShouldEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		emailEnabled bool
		emailTypes   map[shared.NotificationType]bool
		notifType    shared.NotificationType
		want         bool
	}{
		{
			name:         "account status always sends email regardless of preferences",
			emailEnabled: false,
			emailTypes:   map[shared.NotificationType]bool{shared.NotificationTypeAccountSuspended: false},
			notifType:    shared.NotificationTypeAccountSuspended,
			want:         true,
		},
		{
			name:         "account banned always sends email",
			emailEnabled: false,
			emailTypes:   nil,
			notifType:    shared.NotificationTypeAccountBanned,
			want:         true,
		},
		{
			name:         "account reinstated always sends email",
			emailEnabled: false,
			emailTypes:   nil,
			notifType:    shared.NotificationTypeAccountReinstated,
			want:         true,
		},
		{
			name:         "email globally disabled returns false for normal types",
			emailEnabled: false,
			emailTypes:   nil,
			notifType:    shared.NotificationTypeNewFollower,
			want:         false,
		},
		{
			name:         "email enabled with no per-type preference defaults to true",
			emailEnabled: true,
			emailTypes:   map[shared.NotificationType]bool{},
			notifType:    shared.NotificationTypeNewFollower,
			want:         true,
		},
		{
			name:         "email enabled with explicit type enabled",
			emailEnabled: true,
			emailTypes:   map[shared.NotificationType]bool{shared.NotificationTypeNewFollower: true},
			notifType:    shared.NotificationTypeNewFollower,
			want:         true,
		},
		{
			name:         "email enabled with explicit type disabled",
			emailEnabled: true,
			emailTypes:   map[shared.NotificationType]bool{shared.NotificationTypeNewFollower: false},
			notifType:    shared.NotificationTypeNewFollower,
			want:         false,
		},
		{
			name:         "email enabled for different type than requested",
			emailEnabled: true,
			emailTypes:   map[shared.NotificationType]bool{shared.NotificationTypeNewPhotos: true},
			notifType:    shared.NotificationTypeNewFollower,
			want:         true, // Defaults to true if not explicitly set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefs := NewNotificationPreferences(tt.emailEnabled, tt.emailTypes, DigestDaily)
			got := prefs.ShouldEmail(tt.notifType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationPreferences_EnableEmail(t *testing.T) {
	t.Parallel()

	prefs := NewNotificationPreferences(false, nil, DigestDaily)
	assert.False(t, prefs.EmailEnabled())

	prefs.EnableEmail()
	assert.True(t, prefs.EmailEnabled())

	// Enabling again should be idempotent
	prefs.EnableEmail()
	assert.True(t, prefs.EmailEnabled())
}

func TestNotificationPreferences_DisableEmail(t *testing.T) {
	t.Parallel()

	prefs := NewNotificationPreferences(true, nil, DigestDaily)
	assert.True(t, prefs.EmailEnabled())

	prefs.DisableEmail()
	assert.False(t, prefs.EmailEnabled())

	// Disabling again should be idempotent
	prefs.DisableEmail()
	assert.False(t, prefs.EmailEnabled())
}

func TestNotificationPreferences_SetEmailTypePreference(t *testing.T) {
	t.Parallel()

	t.Run("set preference on empty map", func(t *testing.T) {
		t.Parallel()

		prefs := NewNotificationPreferences(true, nil, DigestDaily)
		prefs.SetEmailTypePreference(shared.NotificationTypeNewFollower, true)

		types := prefs.EmailTypes()
		assert.True(t, types[shared.NotificationTypeNewFollower])
	})

	t.Run("override existing preference", func(t *testing.T) {
		t.Parallel()

		prefs := NewNotificationPreferences(true, map[shared.NotificationType]bool{
			shared.NotificationTypeNewFollower: true,
		}, DigestDaily)

		prefs.SetEmailTypePreference(shared.NotificationTypeNewFollower, false)

		types := prefs.EmailTypes()
		assert.False(t, types[shared.NotificationTypeNewFollower])
	})

	t.Run("set multiple preferences", func(t *testing.T) {
		t.Parallel()

		prefs := NewNotificationPreferences(true, nil, DigestDaily)

		prefs.SetEmailTypePreference(shared.NotificationTypeNewFollower, true)
		prefs.SetEmailTypePreference(shared.NotificationTypeNewPhotos, false)
		prefs.SetEmailTypePreference(shared.NotificationTypeAbuseReport, true)

		types := prefs.EmailTypes()
		assert.True(t, types[shared.NotificationTypeNewFollower])
		assert.False(t, types[shared.NotificationTypeNewPhotos])
		assert.True(t, types[shared.NotificationTypeAbuseReport])
	})

	t.Run("initialize nil map if needed", func(t *testing.T) {
		t.Parallel()

		// Create prefs with manually set nil map (edge case)
		prefs := NotificationPreferences{
			emailEnabled:    true,
			emailTypes:      nil,
			digestFrequency: DigestDaily,
		}

		prefs.SetEmailTypePreference(shared.NotificationTypeNewFollower, true)

		types := prefs.EmailTypes()
		assert.NotNil(t, types)
		assert.True(t, types[shared.NotificationTypeNewFollower])
	})
}

func TestNotificationPreferences_SetDigestFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		initialFrequency DigestFrequency
		newFrequency     DigestFrequency
		wantFrequency    DigestFrequency
	}{
		{
			name:             "valid frequency change",
			initialFrequency: DigestDaily,
			newFrequency:     DigestHourly,
			wantFrequency:    DigestHourly,
		},
		{
			name:             "invalid frequency ignored",
			initialFrequency: DigestDaily,
			newFrequency:     DigestFrequency("invalid"),
			wantFrequency:    DigestDaily,
		},
		{
			name:             "empty frequency ignored",
			initialFrequency: DigestWeekly,
			newFrequency:     DigestFrequency(""),
			wantFrequency:    DigestWeekly,
		},
		{
			name:             "change from immediate to weekly",
			initialFrequency: DigestImmediate,
			newFrequency:     DigestWeekly,
			wantFrequency:    DigestWeekly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefs := NewNotificationPreferences(true, nil, tt.initialFrequency)
			prefs.SetDigestFrequency(tt.newFrequency)

			assert.Equal(t, tt.wantFrequency, prefs.DigestFrequency())
		})
	}
}
