package shared_test

import (
	"errors"
	"testing"

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
				if err == nil {
					t.Errorf("NewPagination() expected error, got nil")
				}
				if !errors.Is(err, shared.ErrInvalidInput) {
					t.Errorf("NewPagination() error = %v, want wrapped ErrInvalidInput", err)
				}
				return
			}

			if err != nil {
				t.Errorf("NewPagination() unexpected error = %v", err)
				return
			}

			if p.Page() != tt.wantPage {
				t.Errorf("Page() = %v, want %v", p.Page(), tt.wantPage)
			}
			if p.PerPage() != tt.wantPerPage {
				t.Errorf("PerPage() = %v, want %v", p.PerPage(), tt.wantPerPage)
			}
			if p.Total() != 0 {
				t.Errorf("Total() = %v, want 0", p.Total())
			}
		})
	}
}

func TestDefaultPagination(t *testing.T) {
	t.Parallel()

	p := shared.DefaultPagination()

	if p.Page() != shared.DefaultPage {
		t.Errorf("Page() = %v, want %v", p.Page(), shared.DefaultPage)
	}
	if p.PerPage() != shared.DefaultPerPage {
		t.Errorf("PerPage() = %v, want %v", p.PerPage(), shared.DefaultPerPage)
	}
	if p.Total() != 0 {
		t.Errorf("Total() = %v, want 0", p.Total())
	}
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

			if p.Total() != tt.wantTotal {
				t.Errorf("Total() = %v, want %v", p.Total(), tt.wantTotal)
			}

			// Verify original pagination values are preserved
			if p.Page() != tt.page {
				t.Errorf("Page() = %v, want %v", p.Page(), tt.page)
			}
			if p.PerPage() != tt.perPage {
				t.Errorf("PerPage() = %v, want %v", p.PerPage(), tt.perPage)
			}
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

			if p.Offset() != tt.wantOffset {
				t.Errorf("Offset() = %v, want %v", p.Offset(), tt.wantOffset)
			}
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

			if p.Limit() != tt.wantLimit {
				t.Errorf("Limit() = %v, want %v", p.Limit(), tt.wantLimit)
			}
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

			if p.TotalPages() != tt.wantTotalPages {
				t.Errorf("TotalPages() = %v, want %v", p.TotalPages(), tt.wantTotalPages)
			}
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

			if p.HasNext() != tt.wantHasNext {
				t.Errorf("HasNext() = %v, want %v", p.HasNext(), tt.wantHasNext)
			}
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

			if p.HasPrev() != tt.wantHasPrev {
				t.Errorf("HasPrev() = %v, want %v", p.HasPrev(), tt.wantHasPrev)
			}
		})
	}
}

func TestPagination_Immutability(t *testing.T) {
	t.Parallel()

	original, _ := shared.NewPagination(1, 20)
	modified := original.WithTotal(100)

	// Verify original is unchanged
	if original.Total() != 0 {
		t.Errorf("original Total() = %v, want 0 (immutability violated)", original.Total())
	}

	// Verify modified has new total
	if modified.Total() != 100 {
		t.Errorf("modified Total() = %v, want 100", modified.Total())
	}

	// Verify other fields are preserved
	if modified.Page() != original.Page() {
		t.Errorf("modified Page() = %v, want %v", modified.Page(), original.Page())
	}
	if modified.PerPage() != original.PerPage() {
		t.Errorf("modified PerPage() = %v, want %v", modified.PerPage(), original.PerPage())
	}
}
