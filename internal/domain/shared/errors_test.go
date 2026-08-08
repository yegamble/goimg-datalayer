package shared_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "resource not found", shared.ErrNotFound.Error())
	assert.Equal(t, "resource already exists", shared.ErrAlreadyExists.Error())
	assert.Equal(t, "invalid input", shared.ErrInvalidInput.Error())
	assert.Equal(t, "unauthorized", shared.ErrUnauthorized.Error())
	assert.Equal(t, "forbidden", shared.ErrForbidden.Error())

	// Test that they behave correctly with errors.Is
	err := shared.ErrNotFound
	assert.True(t, errors.Is(err, shared.ErrNotFound))
	assert.False(t, errors.Is(err, shared.ErrForbidden))
}
