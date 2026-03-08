package activity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/activity"
)

func TestActivityID(t *testing.T) {
	id := activity.NewActivityID()
	assert.NotEmpty(t, id.String())
}
