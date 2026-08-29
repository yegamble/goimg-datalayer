package community

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupImageID(t *testing.T) {
	id := NewGroupImageID()
	assert.False(t, id.IsZero())
	assert.NotEmpty(t, id.String())
	assert.NotEqual(t, uuid.Nil, id.UUID())

	parsed, err := ParseGroupImageID(id.String())
	assert.NoError(t, err)
	assert.True(t, id.Equals(parsed))

	_, err = ParseGroupImageID("invalid")
	assert.Error(t, err)

	mustParsed := MustParseGroupImageID(id.String())
	assert.True(t, id.Equals(mustParsed))
}
