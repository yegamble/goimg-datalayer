package testhelpers

import (
	"time"

	"github.com/google/uuid"

	"github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
)

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
	ValidUserID    = identity.NewUserID()
	ValidSessionID = uuid.New()
	ValidFamilyID  = uuid.New().String()
)

func ValidUser() *identity.User {
	email, _ := identity.NewEmail(ValidEmail)
	username, _ := identity.NewUsername(ValidUsername)
	passwordHash, _ := identity.NewPasswordHash(ValidPassword)

	user, _ := identity.NewUser(email, username, passwordHash)
	user.ClearEvents()
	return user
}

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
		0,
		time.Now().UTC(),
		time.Now().UTC(),
		identity.UserTypeRegistered,
		nil,
		nil,
		false,
		nil,
	)
	return user
}

func ValidUserWithPassword(password string) *identity.User {
	email, _ := identity.NewEmail(ValidEmail)
	username, _ := identity.NewUsername(ValidUsername)
	passwordHash, _ := identity.NewPasswordHash(password)

	user, _ := identity.NewUser(email, username, passwordHash)
	_ = user.Activate()
	user.ClearEvents()
	return user
}

func ValidActiveUser() *identity.User {
	user := ValidUser()
	_ = user.Activate()
	user.ClearEvents()
	return user
}

func ValidAdminUser() *identity.User {
	user := ValidActiveUser()
	_ = user.ChangeRole(identity.RoleAdmin)
	user.ClearEvents()
	return user
}

func ValidSuspendedUser() *identity.User {
	user := ValidActiveUser()
	_ = user.Suspend("Test suspension")
	user.ClearEvents()
	return user
}

func ValidDeletedUser() *identity.User {
	email, _ := identity.NewEmail(ValidEmail)
	username, _ := identity.NewUsername(ValidPassword)
	passwordHash, _ := identity.NewPasswordHash(ValidPassword)

	user := identity.ReconstructUser(
		identity.NewUserID(),
		email,
		username,
		passwordHash,
		identity.RoleUser,
		identity.StatusDeleted,
		ValidDisplayName,
		ValidBio,
		0,
		time.Now().UTC(),
		time.Now().UTC(),
		identity.UserTypeRegistered,
		nil,
		nil,
		false,
		nil,
	)
	return user
}

func ValidActiveUserWithIDAndUsername(userID identity.UserID, emailStr, usernameStr string) *identity.User {
	email, _ := identity.NewEmail(emailStr)
	username, _ := identity.NewUsername(usernameStr)
	passwordHash, _ := identity.NewPasswordHash(ValidPassword)

	user := identity.ReconstructUser(
		userID,
		email,
		username,
		passwordHash,
		identity.RoleUser,
		identity.StatusActive,
		usernameStr,
		"",
		0,
		time.Now().UTC(),
		time.Now().UTC(),
		identity.UserTypeRegistered,
		nil,
		nil,
		false,
		nil,
	)
	return user
}

func ValidEmailVO() identity.Email {
	email, _ := identity.NewEmail(ValidEmail)
	return email
}

func ValidUsernameVO() identity.Username {
	username, _ := identity.NewUsername(ValidUsername)
	return username
}

func ValidPasswordHashVO() identity.PasswordHash {
	hash, _ := identity.NewPasswordHash(ValidPassword)
	return hash
}

func ValidTokenPair() (string, string) {
	return "valid.access.token", "valid.refresh.token"
}

func ValidJWTClaims() *services.JWTClaims {
	now := time.Now().UTC()
	return &services.JWTClaims{
		UserID:    ValidUserID.String(),
		Email:     ValidEmail,
		Role:      string(identity.RoleUser),
		SessionID: ValidSessionID.String(),
		TokenType: "access",
		JTI:       uuid.New().String(),
		ExpiresAt: now.Add(15 * time.Minute),
	}
}

func ExpiredJWTClaims() *services.JWTClaims {
	now := time.Now().UTC()
	return &services.JWTClaims{
		UserID:    ValidUserID.String(),
		Email:     ValidEmail,
		Role:      string(identity.RoleUser),
		SessionID: ValidSessionID.String(),
		TokenType: "access",
		JTI:       uuid.New().String(),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
}

func ValidRefreshTokenMetadata() *services.RefreshTokenMetadata {
	now := time.Now().UTC()
	return &services.RefreshTokenMetadata{
		TokenHash:  "test-token-hash",
		UserID:     ValidUserID.String(),
		SessionID:  ValidSessionID.String(),
		FamilyID:   ValidFamilyID,
		IssuedAt:   now,
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		IP:         ValidIPAddress,
		UserAgent:  ValidUserAgent,
		ParentHash: "",
		Used:       false,
	}
}

func ExpiredRefreshTokenMetadata() *services.RefreshTokenMetadata {
	now := time.Now().UTC()
	return &services.RefreshTokenMetadata{
		TokenHash:  "expired-token-hash",
		UserID:     ValidUserID.String(),
		SessionID:  ValidSessionID.String(),
		FamilyID:   ValidFamilyID,
		IssuedAt:   now.Add(-8 * 24 * time.Hour),
		ExpiresAt:  now.Add(-1 * time.Hour),
		IP:         ValidIPAddress,
		UserAgent:  ValidUserAgent,
		ParentHash: "",
		Used:       false,
	}
}

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

func ValidSession() services.Session {
	now := time.Now().UTC()
	return services.Session{
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

func AlternateEmail() identity.Email {
	email, _ := identity.NewEmail("alternate@example.com")
	return email
}

func AlternateUsername() identity.Username {
	username, _ := identity.NewUsername("alternateuser")
	return username
}

func InvalidEmails() []string {
	return []string{
		"",
		"notanemail",
		"@example.com",
		"user@",
		"user name@test.com",
		"user@mailinator.com",
	}
}

func InvalidUsernames() []string {
	return []string{
		"",
		"ab",
		"user@",
		"user ",
		"admin",
		"system",
	}
}

func InvalidPasswords() []string {
	return []string{
		"",
		"short",
		"nodigit",
		"NOUPPER",
		"nolower1",
		"NoSpecial1",
	}
}

func WeakPasswords() []string {
	return []string{
		"Password1!",
		"Welcome123!",
		"Test1234!",
		"Qwerty123!",
	}
}
