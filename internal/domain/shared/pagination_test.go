package shared

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPagination(t *testing.T) {
	p := DefaultPagination()
	assert.Equal(t, 1, p.Page())
	assert.Equal(t, 20, p.PerPage())

	p2, err := NewPagination(2, 50)
	assert.NoError(t, err)
	assert.Equal(t, 2, p2.Page())
	assert.Equal(t, 50, p2.PerPage())
	assert.Equal(t, 50, p2.Offset())
	assert.Equal(t, 50, p2.Limit())

	p3 := p2.WithTotal(150)
	assert.Equal(t, int64(150), p3.Total())
	assert.Equal(t, 3, p3.TotalPages())
	assert.True(t, p3.HasNext())
	assert.True(t, p3.HasPrev())

	p4 := p2.WithTotal(101)
	assert.Equal(t, 3, p4.TotalPages())

	p5 := p2.WithTotal(-10)
	assert.Equal(t, int64(0), p5.Total())
	assert.False(t, p5.HasNext())

	_, err = NewPagination(0, 10)
	assert.Error(t, err)

	_, err = NewPagination(1, 0)
	assert.Error(t, err)

	_, err = NewPagination(1, 101)
	assert.Error(t, err)
}
