package handlers

// HTTP handler pagination and limits.
const (
	defaultLimit   = 20   // Default pagination limit
	maxLimit       = 100  // Maximum pagination limit
	maxFileSize    = 50   // Maximum file size in MB
	msToSecDivisor = 1000 // Divisor to convert milliseconds to seconds
	contextTimeout = 30   // Default context timeout in seconds

	// defaultPerPage is the default number of items returned per page in list/search endpoints.
	defaultPerPage = 20
	// maxPerPage is the maximum number of items allowed per page.
	maxPerPage = 100
)
