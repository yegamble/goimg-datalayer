package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

func TestNewPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        int
		perPage     int
		expectPage  int
		expectPer   int
		expectError bool
	}{
		{"Valid", 1, 20, 1, 20, false},
		{"Page < 1", 0, 20, 0, 0, true},
		{"PerPage < Min", 1, 0, 0, 0, true},
		{"PerPage > Max", 1, 101, 0, 0, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := shared.NewPagination(tt.page, tt.perPage)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectPage, p.Page())
				assert.Equal(t, tt.expectPer, p.PerPage())
			}
		})
	}
}

func TestDefaultPagination(t *testing.T) {
	t.Parallel()
	p := shared.DefaultPagination()
	assert.Equal(t, 1, p.Page())
	assert.Equal(t, 20, p.PerPage())
	assert.Equal(t, int64(0), p.Total())
}

func TestPagination_Calculations(t *testing.T) {
	t.Parallel()

	p, _ := shared.NewPagination(2, 10)

	assert.Equal(t, 10, p.Offset())
	assert.Equal(t, 10, p.Limit())

	p = p.WithTotal(105)
	assert.Equal(t, int64(105), p.Total())
	assert.Equal(t, 11, p.TotalPages())
	assert.True(t, p.HasNext())
	assert.True(t, p.HasPrev())

	// Last page
	pLast, _ := shared.NewPagination(11, 10)
	pLast = pLast.WithTotal(105)
	assert.False(t, pLast.HasNext())
	assert.True(t, pLast.HasPrev())

	// First page
	pFirst, _ := shared.NewPagination(1, 10)
	pFirst = pFirst.WithTotal(105)
	assert.True(t, pFirst.HasNext())
	assert.False(t, pFirst.HasPrev())

	// Empty total
	pEmpty, _ := shared.NewPagination(1, 10)
	assert.Equal(t, 0, pEmpty.TotalPages())
	assert.False(t, pEmpty.HasNext())
}
