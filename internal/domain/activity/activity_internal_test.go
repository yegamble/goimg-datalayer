package activity

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"testing"
	"time"
)

func TestActivity_AddEvent_Internal(t *testing.T) {
	t.Parallel()

	actorID := identity.NewUserID()
	targetID := uuid.New()

	a, err := NewActivity(actorID, ActivityTypeImageUploaded, targetID, TargetTypeImage, nil)
	require.NoError(t, err)

	a.addEvent(nil)
	assert.Len(t, a.Events(), 2)
}

func TestActivity_Metadata(t *testing.T) {
	a := Activity{
		metadata: map[string]string{"foo": "bar"},
	}
	m := a.Metadata()
	assert.Equal(t, "bar", m["foo"])
}

func TestReconstructActivity_NilMetadata(t *testing.T) {
	t.Parallel()
	act := ReconstructActivity(
		NewActivityID(),
		identity.NewUserID(),
		ActivityTypeImageUploaded,
		uuid.New(),
		TargetTypeImage,
		nil,
		time.Now(),
	)
	assert.NotNil(t, act.Metadata())
}
