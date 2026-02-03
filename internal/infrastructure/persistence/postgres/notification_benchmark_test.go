package postgres

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/domain/notification"
)

func BenchmarkNotificationRow_toDomain(b *testing.B) {
	// Setup test data
	meta := map[string]string{
		"key1":     "value1",
		"key2":     "value2",
		"key3":     "value3",
		"long_key": "some reasonably long string value to simulate real metadata",
	}
	metaBytes, _ := json.Marshal(meta)

	row := notificationRow{
		ID:               notification.NewNotificationID().String(),
		RecipientID:      identity.NewUserID().String(),
		NotificationType: notification.TypeNewFollower.String(),
		Title:            "New Follower",
		Body:             "You have a new follower",
		Metadata:         metaBytes,
		CreatedAt:        time.Now(),
		ReadAt:           sql.NullTime{Valid: false},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := row.toDomain()
		if err != nil {
			b.Fatalf("toDomain failed: %v", err)
		}
	}
}

func BenchmarkNotificationRow_toDomain_EmptyMetadata(b *testing.B) {
	row := notificationRow{
		ID:               notification.NewNotificationID().String(),
		RecipientID:      identity.NewUserID().String(),
		NotificationType: notification.TypeNewFollower.String(),
		Title:            "New Follower",
		Body:             "You have a new follower",
		Metadata:         []byte{}, // Empty
		CreatedAt:        time.Now(),
		ReadAt:           sql.NullTime{Valid: false},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := row.toDomain()
		if err != nil {
			b.Fatalf("toDomain failed: %v", err)
		}
	}
}
