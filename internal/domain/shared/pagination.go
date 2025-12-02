package shared

import "fmt"

const (
	// DefaultPage is the default page number (1-indexed).
	DefaultPage = 1

	// DefaultPerPage is the default number of items per page.
	DefaultPerPage = 20

	// MinPerPage is the minimum allowed items per page.
	MinPerPage = 1

	// MaxPerPage is the maximum allowed items per page.
	MaxPerPage = 100
)

// Pagination is an immutable value object that encapsulates pagination parameters.
// Pages are 1-indexed (first page is 1, not 0).
type Pagination struct {
	page    int
	perPage int
	total   int64
}

// NewPagination creates a new Pagination with validation.
// page must be >= 1 (defaults to 1 if invalid).
// perPage must be >= 1 and <= 100 (defaults to 20 if invalid).
func NewPagination(page, perPage int) (Pagination, error) {
	// Validate and normalize page
	if page < 1 {
		return Pagination{}, fmt.Errorf("%w: page must be >= 1, got %d", ErrInvalidInput, page)
	}

	// Validate perPage range
	if perPage < MinPerPage {
		return Pagination{}, fmt.Errorf("%w: perPage must be >= %d, got %d", ErrInvalidInput, MinPerPage, perPage)
	}
	if perPage > MaxPerPage {
		return Pagination{}, fmt.Errorf("%w: perPage must be <= %d, got %d", ErrInvalidInput, MaxPerPage, perPage)
	}

	return Pagination{
		page:    page,
		perPage: perPage,
		total:   0, // Total is unknown until set via WithTotal
	}, nil
}

// DefaultPagination returns a Pagination with default values (page 1, 20 per page).
func DefaultPagination() Pagination {
	return Pagination{
		page:    DefaultPage,
		perPage: DefaultPerPage,
		total:   0,
	}
}

// Page returns the current page number (1-indexed).
func (p Pagination) Page() int {
	return p.page
}

// PerPage returns the number of items per page.
func (p Pagination) PerPage() int {
	return p.perPage
}

// Total returns the total number of items across all pages.
func (p Pagination) Total() int64 {
	return p.total
}

// WithTotal returns a new Pagination with the total count set.
// This is typically called after querying the database for the total count.
func (p Pagination) WithTotal(total int64) Pagination {
	if total < 0 {
		total = 0
	}
	return Pagination{
		page:    p.page,
		perPage: p.perPage,
		total:   total,
	}
}

// Offset returns the offset for database queries (0-indexed).
// Formula: (page - 1) * perPage.
func (p Pagination) Offset() int {
	return (p.page - 1) * p.perPage
}

// Limit returns the limit for database queries.
// This is equivalent to PerPage.
func (p Pagination) Limit() int {
	return p.perPage
}

// TotalPages returns the total number of pages based on the total count.
// Returns 0 if total is not set.
func (p Pagination) TotalPages() int {
	if p.total == 0 {
		return 0
	}
	pages := int(p.total) / p.perPage
	if int(p.total)%p.perPage > 0 {
		pages++
	}
	return pages
}

// HasNext returns true if there is a next page available.
func (p Pagination) HasNext() bool {
	if p.total == 0 {
		return false
	}
	return p.page < p.TotalPages()
}

// HasPrev returns true if there is a previous page available.
func (p Pagination) HasPrev() bool {
	return p.page > 1
}
