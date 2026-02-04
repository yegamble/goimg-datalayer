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
		wantErr     bool
		wantPage    int
		wantPerPage int
	}{
		{
			name:        "valid pagination",
			page:        1,
			perPage:     20,
			wantErr:     false,
			wantPage:    1,
			wantPerPage: 20,
		},
		{
			name:        "valid pagination with custom values",
			page:        5,
			perPage:     50,
			wantErr:     false,
			wantPage:    5,
			wantPerPage: 50,
		},
		{
			name:        "valid pagination at min perPage",
			page:        1,
			perPage:     shared.MinPerPage,
			wantErr:     false,
			wantPage:    1,
			wantPerPage: shared.MinPerPage,
		},
		{
			name:        "valid pagination at max perPage",
			page:        1,
			perPage:     shared.MaxPerPage,
			wantErr:     false,
			wantPage:    1,
			wantPerPage: shared.MaxPerPage,
		},
		{
			name:    "invalid page zero",
			page:    0,
			perPage: 20,
			wantErr: true,
		},
		{
			name:    "invalid negative page",
			page:    -1,
			perPage: 20,
			wantErr: true,
		},
		{
			name:    "invalid perPage zero",
			page:    1,
			perPage: 0,
			wantErr: true,
		},
		{
			name:    "invalid negative perPage",
			page:    1,
			perPage: -5,
			wantErr: true,
		},
		{
			name:    "invalid perPage exceeds max",
			page:    1,
			perPage: shared.MaxPerPage + 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := shared.NewPagination(tt.page, tt.perPage)

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, shared.ErrInvalidInput)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPage, p.Page())
			assert.Equal(t, tt.wantPerPage, p.PerPage())
			assert.Equal(t, int64(0), p.Total())
		})
	}
}

func TestDefaultPagination(t *testing.T) {
	t.Parallel()

	p := shared.DefaultPagination()

	assert.Equal(t, shared.DefaultPage, p.Page())
	assert.Equal(t, shared.DefaultPerPage, p.PerPage())
	assert.Equal(t, int64(0), p.Total())
}

func TestPagination_WithTotal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		page      int
		perPage   int
		total     int64
		wantTotal int64
	}{
		{
			name:      "positive total",
			page:      1,
			perPage:   20,
			total:     100,
			wantTotal: 100,
		},
		{
			name:      "zero total",
			page:      1,
			perPage:   20,
			total:     0,
			wantTotal: 0,
		},
		{
			name:      "negative total normalized to zero",
			page:      1,
			perPage:   20,
			total:     -5,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)
			p = p.WithTotal(tt.total)

			assert.Equal(t, tt.wantTotal, p.Total())
			assert.Equal(t, tt.page, p.Page())
			assert.Equal(t, tt.perPage, p.PerPage())
		})
	}
}

func TestPagination_Offset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		page       int
		perPage    int
		wantOffset int
	}{
		{
			name:       "first page",
			page:       1,
			perPage:    20,
			wantOffset: 0,
		},
		{
			name:       "second page",
			page:       2,
			perPage:    20,
			wantOffset: 20,
		},
		{
			name:       "fifth page",
			page:       5,
			perPage:    20,
			wantOffset: 80,
		},
		{
			name:       "first page with custom perPage",
			page:       1,
			perPage:    50,
			wantOffset: 0,
		},
		{
			name:       "third page with custom perPage",
			page:       3,
			perPage:    50,
			wantOffset: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)

			assert.Equal(t, tt.wantOffset, p.Offset())
		})
	}
}

func TestPagination_Limit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		page      int
		perPage   int
		wantLimit int
	}{
		{
			name:      "default perPage",
			page:      1,
			perPage:   20,
			wantLimit: 20,
		},
		{
			name:      "custom perPage",
			page:      1,
			perPage:   50,
			wantLimit: 50,
		},
		{
			name:      "min perPage",
			page:      1,
			perPage:   shared.MinPerPage,
			wantLimit: shared.MinPerPage,
		},
		{
			name:      "max perPage",
			page:      1,
			perPage:   shared.MaxPerPage,
			wantLimit: shared.MaxPerPage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)

			assert.Equal(t, tt.wantLimit, p.Limit())
		})
	}
}

func TestPagination_TotalPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		page           int
		perPage        int
		total          int64
		wantTotalPages int
	}{
		{
			name:           "zero total",
			page:           1,
			perPage:        20,
			total:          0,
			wantTotalPages: 0,
		},
		{
			name:           "exact multiple",
			page:           1,
			perPage:        20,
			total:          100,
			wantTotalPages: 5,
		},
		{
			name:           "with remainder",
			page:           1,
			perPage:        20,
			total:          95,
			wantTotalPages: 5,
		},
		{
			name:           "less than one page",
			page:           1,
			perPage:        20,
			total:          15,
			wantTotalPages: 1,
		},
		{
			name:           "exactly one page",
			page:           1,
			perPage:        20,
			total:          20,
			wantTotalPages: 1,
		},
		{
			name:           "one more than one page",
			page:           1,
			perPage:        20,
			total:          21,
			wantTotalPages: 2,
		},
		{
			name:           "large dataset",
			page:           1,
			perPage:        20,
			total:          1000,
			wantTotalPages: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)
			p = p.WithTotal(tt.total)

			assert.Equal(t, tt.wantTotalPages, p.TotalPages())
		})
	}
}

func TestPagination_HasNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        int
		perPage     int
		total       int64
		wantHasNext bool
	}{
		{
			name:        "zero total",
			page:        1,
			perPage:     20,
			total:       0,
			wantHasNext: false,
		},
		{
			name:        "first page with more pages",
			page:        1,
			perPage:     20,
			total:       100,
			wantHasNext: true,
		},
		{
			name:        "middle page",
			page:        3,
			perPage:     20,
			total:       100,
			wantHasNext: true,
		},
		{
			name:        "last page",
			page:        5,
			perPage:     20,
			total:       100,
			wantHasNext: false,
		},
		{
			name:        "beyond last page",
			page:        10,
			perPage:     20,
			total:       100,
			wantHasNext: false,
		},
		{
			name:        "single page",
			page:        1,
			perPage:     20,
			total:       15,
			wantHasNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)
			p = p.WithTotal(tt.total)

			assert.Equal(t, tt.wantHasNext, p.HasNext())
		})
	}
}

func TestPagination_HasPrev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        int
		perPage     int
		wantHasPrev bool
	}{
		{
			name:        "first page",
			page:        1,
			perPage:     20,
			wantHasPrev: false,
		},
		{
			name:        "second page",
			page:        2,
			perPage:     20,
			wantHasPrev: true,
		},
		{
			name:        "middle page",
			page:        5,
			perPage:     20,
			wantHasPrev: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, _ := shared.NewPagination(tt.page, tt.perPage)

			assert.Equal(t, tt.wantHasPrev, p.HasPrev())
		})
	}
}

func TestPagination_Immutability(t *testing.T) {
	t.Parallel()

	original, _ := shared.NewPagination(1, 20)
	modified := original.WithTotal(100)

	// Verify original is unchanged
	assert.Equal(t, int64(0), original.Total())

	// Verify modified has new total
	assert.Equal(t, int64(100), modified.Total())

	// Verify other fields are preserved
	assert.Equal(t, original.Page(), modified.Page())
	assert.Equal(t, original.PerPage(), modified.PerPage())
}
