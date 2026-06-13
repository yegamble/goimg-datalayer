package community

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupActivityID(t *testing.T) {
	id := NewGroupActivityID()
	assert.NotEmpty(t, id.String())

	parsed, err := ParseGroupActivityID(id.String())
	assert.NoError(t, err)
	assert.True(t, id.Equals(parsed))

	_, err = ParseGroupActivityID("invalid")
	assert.Error(t, err)

	mustParsed := MustParseGroupActivityID(id.String())
	assert.True(t, id.Equals(mustParsed))
}
