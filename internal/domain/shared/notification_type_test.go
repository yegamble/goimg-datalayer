package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_Methods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ              shared.NotificationType
		wantRequiresEmail bool
		wantIsAdminOnly   bool
		wantIsValid       bool
	}{
		{shared.NotificationTypeNewFollower, false, false, true},
		{shared.NotificationTypeAccountSuspended, true, false, true},
		{shared.NotificationTypeAbuseReport, false, true, true},
		{"invalid_type", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			assert.Equal(t, tt.wantRequiresEmail, tt.typ.RequiresEmail())
			assert.Equal(t, tt.wantIsAdminOnly, tt.typ.IsAdminOnly())
			assert.Equal(t, tt.wantIsValid, tt.typ.IsValid())
		})
	}
}
