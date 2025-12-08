package fixtures

// JWTTestKeys provides RSA key pairs for JWT testing.
// WARNING: DO NOT USE THESE KEYS IN PRODUCTION!
// These are test-only keys with weak security for fast test execution.

const (
	// TestPrivateKeyPEM is a 2048-bit RSA private key for JWT signing in tests.
	// DO NOT USE IN PRODUCTION!
	//nolint:gosec // G101: This is a test fixture, not production credentials
	TestPrivateKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/vLBnJ7RG8tJRY7iIBn5nU0AHGN0HtqOV
/aWqhFr7Np2lNb/b6TJNF3LAeqJm7qEqbh5fqiQWdqTJXI1l0b8mR2nV6KQVT1Yo
U4GUj9hy+TcKfqh4fQeOLQkIl3DhDpKxH7KStXpEW9LSrRgL0KLZjcxKvC3LmQJw
8GHz2q2TqRJ9KXKxqJKCw0rLCEQqZQDt0D8c3KUFqGjGjN2Fz6Q9cLEPxfYh0aDq
u2wRJqQb+RrNBh1xYzVGGN6W8xT0JYHmMCLgBLDiQxS/uKxfqVqCXqf9MhTPh4Ky
YPZEPywJBSvQKRB9iFcBxB0RQD6Bv3XBw8OvowIDAQABAoIBADvPH2Ls1FgBt7FU
M8KQfqBqF7FhK5YQCzGUGOIJJt8n4DQ5l3pTMjxeqLKDU3fZFE7ZQ8HmDLb3qLM2
k5VyJn9j9TRqxNKGQxLv6GWlEfqLDCJLpHm7xvOQBJSIZZXKMRqLLKPBNNJW7+b3
oVQxXVfLU0QBQMxQ7XLhhGnCQ7PvqLFsF7yjnQsYJa/OQQJdYLJhQwKVpKW4ZOKX
aKMKGLPBNJHQQXJLZvDLqEXKGAJVhRZ7nCELQrQwLQKBQVGZLvF3XmgXNKBHFAr9
XPZwDLvRF5KTqQKBgQDvPHOoO4hMjxp7NMhv9vKfEPqLQd+FQgLqTgFhQXlK7pTc
aB0JBH3MPmqJvYJLKZQcXMQbCPMQOQKBgQDgLZXdPjFMHmYEpSvD1OQvxLqhQJGZ
6KQhPg7KBQJBANYxjc8VTXAKGvJ7x5F3eCF0w0M0QJC1FhLXqT0g7fQCQQDLLQg5
JVz8Cy8QJxMFYqGLhCQF3sF8Rw2xF3v9Ah7MAkEA0oI+KL7hgPQxLv3XKBq5fQc=
-----END RSA PRIVATE KEY-----`

	// TestPublicKeyPEM is the corresponding 2048-bit RSA public key for JWT verification in tests.
	TestPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z3VS5JJcds3xfn/vLBn
J7RG8tJRY7iIBn5nU0AHGN0HtqOV/aWqhFr7Np2lNb/b6TJNF3LAeqJm7qEqbh5f
qiQWdqTJXI1l0b8mR2nV6KQVT1YoU4GUj9hy+TcKfqh4fQeOLQkIl3DhDpKxH7KS
tXpEW9LSrRgL0KLZjcxKvC3LmQJw8GHz2q2TqRJ9KXKxqJKCw0rLCEQqZQDt0D8c
3KUFqGjGjN2Fz6Q9cLEPxfYh0aDqu2wRJqQb+RrNBh1xYzVGGN6W8xT0JYHmMCLg
BLDiQxS/uKxfqVqCXqf9MhTPh4KyYPZEPywJBSvQKRB9iFcBxB0RQD6Bv3XBw8Ov
owIDAQAB
-----END PUBLIC KEY-----`

	// TestJWTIssuer is the issuer claim for test JWTs.
	TestJWTIssuer = "goimg-test"

	// TestJWTAudience is the audience claim for test JWTs.
	TestJWTAudience = "goimg-api-test"

	// TestAccessTokenDuration is the default duration for test access tokens (15 minutes).
	TestAccessTokenDuration = 15 * 60 // 15 minutes in seconds

	// TestRefreshTokenDuration is the default duration for test refresh tokens (7 days).
	TestRefreshTokenDuration = 7 * 24 * 60 * 60 // 7 days in seconds
)

// JWTClaimsFixture provides test JWT claims.
type JWTClaimsFixture struct {
	UserID   string
	Email    string
	Username string
	Role     string
}

// ValidClaims returns valid JWT claims for testing.
func ValidClaims() *JWTClaimsFixture {
	return &JWTClaimsFixture{
		UserID:   "00000000-0000-0000-0000-000000000001",
		Email:    "test@example.com",
		Username: "testuser",
		Role:     "user",
	}
}

// AdminClaims returns JWT claims with admin role.
func AdminClaims() *JWTClaimsFixture {
	return &JWTClaimsFixture{
		UserID:   "00000000-0000-0000-0000-000000000002",
		Email:    "admin@example.com",
		Username: "admin",
		Role:     "admin",
	}
}

// ModeratorClaims returns JWT claims with moderator role.
func ModeratorClaims() *JWTClaimsFixture {
	return &JWTClaimsFixture{
		UserID:   "00000000-0000-0000-0000-000000000003",
		Email:    "moderator@example.com",
		Username: "moderator",
		Role:     "moderator",
	}
}
