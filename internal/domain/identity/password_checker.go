package identity

import "context"

// PasswordSecurityChecker verifies if passwords have been compromised in known data breaches.
// Implementations should use privacy-preserving techniques (e.g., k-anonymity) to avoid
// transmitting the full password or hash to external services.
//
// This is a domain service interface because password breach checking:
//   - Enforces a business rule (prevent compromised passwords)
//   - Requires external data (breach database)
//   - Doesn't naturally belong to any single entity
//
// Implementations live in the infrastructure layer (e.g., HIBP API client).
type PasswordSecurityChecker interface {
	// IsCompromised checks if a password has appeared in known data breaches.
	// It returns true if the password is compromised, false otherwise.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - password: The plaintext password to check (never logged or stored)
	//
	// Returns:
	//   - bool: true if password is compromised, false if safe
	//   - error: non-nil if the check cannot be performed (API failure, network error, etc.)
	//
	// Security considerations:
	//   - Implementations MUST NOT transmit the full password or hash
	//   - Use k-anonymity (send only hash prefix, e.g., first 5 chars of SHA-1)
	//   - Fail open: if the check fails, allow registration but log the error
	//   - Never log the password itself
	IsCompromised(ctx context.Context, password string) (bool, error)
}
