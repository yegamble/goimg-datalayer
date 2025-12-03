package testhelpers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/redis"
	jwtpkg "github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// Test constants for consistent fixture data.
const (
	ValidEmail       = "test@example.com"
	ValidUsername    = "testuser"
	ValidPassword    = "SecureP@ssw0rd123"
	ValidDisplayName = "Test User"
	ValidBio         = "This is a test bio"
	ValidIPAddress   = "192.168.1.1"
	ValidUserAgent   = "Mozilla/5.0 (Test Browser)"
)

var (
	// ValidUserID is a reusable user ID for tests.
	ValidUserID = identity.NewUserID()
	// ValidSessionID is a reusable session ID for tests.
	ValidSessionID = uuid.New()
	// ValidFamilyID is a reusable family ID for tests.
	ValidFamilyID = uuid.New().String()
)

// ValidUser returns a valid user entity for testing.
func ValidUser() *identity.User {
	email, _ := identity.NewEmail(ValidEmail)
	username, _ := identity.NewUsername(ValidUsername)
	passwordHash, _ := identity.NewPasswordHash(ValidPassword)

	user, _ := identity.NewUser(email, username, passwordHash)
	user.ClearEvents() // Clear creation event for cleaner tests
	return user
}

// ValidUserWithID returns a valid user with a specific ID.
func ValidUserWithID(userID identity.UserID) *identity.User {
	email, _ := identity.NewEmail(ValidEmail)
	username, _ := identity.NewUsername(ValidUsername)
	passwordHash, _ := identity.NewPasswordHash(ValidPassword)

	user := identity.ReconstructUser(
		userID,
		email,
		username,
		passwordHash,
		identity.RoleUser,
		identity.StatusActive,
		ValidDisplayName,
		ValidBio,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	return user
}

// ValidActiveUser returns a user with active status.
func ValidActiveUser() *identity.User {
	user := ValidUser()
	_ = user.Activate()
	user.ClearEvents()
	return user
}

// ValidAdminUser returns a user with admin role.
func ValidAdminUser() *identity.User {
	user := ValidActiveUser()
	_ = user.ChangeRole(identity.RoleAdmin)
	user.ClearEvents()
	return user
}

// ValidSuspendedUser returns a suspended user.
func ValidSuspendedUser() *identity.User {
	user := ValidActiveUser()
	_ = user.Suspend("Test suspension")
	user.ClearEvents()
	return user
}

// ValidEmail returns a valid Email value object.
func ValidEmailVO() identity.Email {
	email, _ := identity.NewEmail(ValidEmail)
	return email
}

// ValidUsername returns a valid Username value object.
func ValidUsernameVO() identity.Username {
	username, _ := identity.NewUsername(ValidUsername)
	return username
}

// ValidPasswordHash returns a valid PasswordHash value object.
func ValidPasswordHashVO() identity.PasswordHash {
	hash, _ := identity.NewPasswordHash(ValidPassword)
	return hash
}

// ValidTokenPair returns valid access and refresh tokens for testing.
func ValidTokenPair() (accessToken string, refreshToken string) {
	return "valid.access.token", "valid.refresh.token"
}

// ValidJWTClaims returns valid JWT claims for testing.
func ValidJWTClaims() *jwtpkg.Claims {
	now := time.Now().UTC()
	return &jwtpkg.Claims{
		UserID:    ValidUserID.String(),
		Email:     ValidEmail,
		Role:      string(identity.RoleUser),
		SessionID: ValidSessionID.String(),
		TokenType: jwtpkg.TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "goimg-api",
			Subject:   ValidUserID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}
}

// ExpiredJWTClaims returns expired JWT claims for testing.
func ExpiredJWTClaims() *jwtpkg.Claims {
	now := time.Now().UTC()
	return &jwtpkg.Claims{
		UserID:    ValidUserID.String(),
		Email:     ValidEmail,
		Role:      string(identity.RoleUser),
		SessionID: ValidSessionID.String(),
		TokenType: jwtpkg.TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "goimg-api",
			Subject:   ValidUserID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ID:        uuid.New().String(),
		},
	}
}

// ValidRefreshTokenMetadata returns valid refresh token metadata for testing.
func ValidRefreshTokenMetadata() *jwtpkg.RefreshTokenMetadata {
	now := time.Now().UTC()
	return &jwtpkg.RefreshTokenMetadata{
		TokenHash:  "test-token-hash",
		UserID:     ValidUserID.String(),
		SessionID:  ValidSessionID.String(),
		FamilyID:   ValidFamilyID,
		IssuedAt:   now,
		ExpiresAt:  now.Add(7 * 24 * time.Hour), // 7 days
		IP:         ValidIPAddress,
		UserAgent:  ValidUserAgent,
		ParentHash: "",
		Used:       false,
	}
}

// ExpiredRefreshTokenMetadata returns expired refresh token metadata.
func ExpiredRefreshTokenMetadata() *jwtpkg.RefreshTokenMetadata {
	now := time.Now().UTC()
	return &jwtpkg.RefreshTokenMetadata{
		TokenHash:  "expired-token-hash",
		UserID:     ValidUserID.String(),
		SessionID:  ValidSessionID.String(),
		FamilyID:   ValidFamilyID,
		IssuedAt:   now.Add(-8 * 24 * time.Hour),
		ExpiresAt:  now.Add(-1 * time.Hour), // Expired 1 hour ago
		IP:         ValidIPAddress,
		UserAgent:  ValidUserAgent,
		ParentHash: "",
		Used:       false,
	}
}

// ValidPostgresSession returns a valid Postgres session for testing.
func ValidPostgresSession() *postgres.Session {
	now := time.Now().UTC()
	return &postgres.Session{
		ID:               ValidSessionID,
		UserID:           ValidUserID,
		RefreshTokenHash: "test-refresh-token-hash",
		IPAddress:        ValidIPAddress,
		UserAgent:        ValidUserAgent,
		ExpiresAt:        now.Add(7 * 24 * time.Hour),
		CreatedAt:        now,
		RevokedAt:        nil,
	}
}

// ValidRedisSession returns a valid Redis session for testing.
func ValidRedisSession() redis.Session {
	now := time.Now().UTC()
	return redis.Session{
		SessionID: ValidSessionID.String(),
		UserID:    ValidUserID.String(),
		Email:     ValidEmail,
		Role:      string(identity.RoleUser),
		IP:        ValidIPAddress,
		UserAgent: ValidUserAgent,
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
}

// AlternateEmail returns an alternate email for testing uniqueness constraints.
func AlternateEmail() identity.Email {
	email, _ := identity.NewEmail("alternate@example.com")
	return email
}

// AlternateUsername returns an alternate username for testing uniqueness constraints.
func AlternateUsername() identity.Username {
	username, _ := identity.NewUsername("alternateuser")
	return username
}

// InvalidEmail returns various invalid email strings for testing validation.
func InvalidEmails() []string {
	return []string{
		"",                    // empty
		"notanemail",          // missing @
		"@example.com",        // missing local part
		"user@",               // missing domain
		"user name@test.com",  // spaces
		"user@mailinator.com", // disposable
	}
}

// InvalidUsernames returns various invalid username strings for testing validation.
func InvalidUsernames() []string {
	return []string{
		"",       // empty
		"ab",     // too short
		"user@",  // invalid character
		"user ",  // space
		"admin",  // reserved
		"system", // reserved
	}
}

// InvalidPasswords returns various invalid password strings for testing validation.
func InvalidPasswords() []string {
	return []string{
		"",         // empty
		"short",    // too short
		"nodigit",  // missing digit
		"NOUPPER",  // missing uppercase
		"nolower1", // missing lowercase
		"NoSpecial1", // missing special character
	}
}

// WeakPasswords returns passwords that pass validation but are weak.
func WeakPasswords() []string {
	return []string{
		"Password1!",    // common pattern
		"Welcome123!",   // common pattern
		"Test1234!",     // sequential
		"Qwerty123!",    // keyboard pattern
	}
}
