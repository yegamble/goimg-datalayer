// Package helpers provides test fixtures and factory functions.
package helpers

import (
	"github.com/google/uuid"
)

// TestUserID returns a consistent UUID for testing.
func TestUserID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// TestEmail returns a consistent test email.
func TestEmail() string {
	return "test@example.com"
}

// TestUsername returns a consistent test username.
func TestUsername() string {
	return "testuser"
}

// RandomUUID generates a random UUID for tests.
func RandomUUID() uuid.UUID {
	return uuid.New()
}
