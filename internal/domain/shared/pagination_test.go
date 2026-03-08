package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestPagination(t *testing.T) {
	p, err := shared.NewPagination(1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, p.Page())
	assert.Equal(t, 10, p.PerPage())
	assert.Equal(t, 0, p.Offset())
	assert.Equal(t, 10, p.Limit())

    p = p.WithTotal(25)
    assert.Equal(t, int64(25), p.Total())
    assert.Equal(t, 3, p.TotalPages())
    assert.True(t, p.HasNext())
    assert.False(t, p.HasPrev())
}

func TestPaginationErrors(t *testing.T) {
    _, err := shared.NewPagination(0, 10)
    assert.ErrorIs(t, err, shared.ErrInvalidInput)

    _, err = shared.NewPagination(1, 0)
    assert.ErrorIs(t, err, shared.ErrInvalidInput)

    _, err = shared.NewPagination(1, 101)
    assert.ErrorIs(t, err, shared.ErrInvalidInput)
}

func TestPaginationDefault(t *testing.T) {
    p := shared.DefaultPagination()
    assert.Equal(t, 1, p.Page())
    assert.Equal(t, 20, p.PerPage())
}
func TestPaginationMore(t *testing.T) {
    p, _ := shared.NewPagination(1, 10)
    p = p.WithTotal(-5)
    assert.Equal(t, int64(0), p.Total())
    assert.Equal(t, 0, p.TotalPages())
    assert.False(t, p.HasNext())

    p2, _ := shared.NewPagination(2, 10)
    p2 = p2.WithTotal(25)
    assert.True(t, p2.HasNext())
}
