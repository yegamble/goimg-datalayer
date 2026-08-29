package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNotificationType_Methods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      shared.NotificationType
		requires  bool
		isAdmin   bool
		isValid   bool
	}{
		{shared.NotificationTypeNewFollower, false, false, true},
		{shared.NotificationTypeAccountSuspended, true, false, true},
		{shared.NotificationTypeMalwareDetected, true, false, true},
		{shared.NotificationType("invalid_type"), false, false, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.requires, tt.name.RequiresEmail())
			assert.Equal(t, tt.isAdmin, tt.name.IsAdminOnly())
			assert.Equal(t, tt.isValid, tt.name.IsValid())
			assert.Equal(t, string(tt.name), tt.name.String())
		})
	}
}
