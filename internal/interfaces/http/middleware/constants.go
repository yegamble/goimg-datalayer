package middleware

// HTTP middleware configuration constants.
const (
	// Logging constants.
	httpStatusServerError = 500 // HTTP 5xx status codes (server errors)
	httpStatusClientError = 400 // HTTP 4xx status codes (client errors)
	httpStatusRedirect    = 300 // HTTP 3xx status codes (redirects)
)
