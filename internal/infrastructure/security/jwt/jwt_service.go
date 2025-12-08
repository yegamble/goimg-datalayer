package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType represents the type of JWT token.
type TokenType string

const (
	// TokenTypeAccess represents an access token used for API authentication.
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh represents a refresh token used for obtaining new access tokens.
	TokenTypeRefresh TokenType = "refresh"

	// JWT configuration defaults and constraints.
	defaultAccessTTL = 15 * time.Minute   // Default access token TTL
	minKeySize       = 4096               // Minimum RSA key size in bits (OWASP 2024)
)

// Config holds JWT service configuration.
type Config struct {
	PrivateKeyPath string        // Path to RSA private key file (PEM format)
	PublicKeyPath  string        // Path to RSA public key file (PEM format)
	AccessTTL      time.Duration // Access token time-to-live (default: 15 minutes)
	RefreshTTL     time.Duration // Refresh token time-to-live (default: 7 days)
	Issuer         string        // Token issuer (default: "goimg-api")
}

// DefaultConfig returns a Config with secure defaults.
func DefaultConfig() Config {
	return Config{
		PrivateKeyPath: "",
		PublicKeyPath:  "",
		AccessTTL:      defaultAccessTTL,
		RefreshTTL:     7 * 24 * time.Hour, // 7 days
		Issuer:         "goimg-api",
	}
}

// Claims represents the JWT claims for goimg tokens.
type Claims struct {
	UserID    string    `json:"user_id"`    // User UUID
	Email     string    `json:"email"`      // User email
	Role      string    `json:"role"`       // User role (user, moderator, admin)
	SessionID string    `json:"session_id"` // Session UUID for token family tracking
	TokenType TokenType `json:"token_type"` // Type of token (access or refresh)
	jwt.RegisteredClaims
}

// Service handles JWT token generation and validation using RS256 algorithm.
type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	config     Config
}

// NewService creates a new JWT service with the given configuration.
// It loads RSA key pairs from the specified paths and validates them.
func NewService(cfg Config) (*Service, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("jwt issuer cannot be empty")
	}

	if cfg.AccessTTL <= 0 {
		return nil, fmt.Errorf("jwt access TTL must be positive")
	}

	if cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("jwt refresh TTL must be positive")
	}

	if cfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("jwt private key path cannot be empty")
	}

	if cfg.PublicKeyPath == "" {
		return nil, fmt.Errorf("jwt public key path cannot be empty")
	}

	// Load private key
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Load public key
	publicKey, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	// Validate key size (must be at least 4096 bits for security)
	if privateKey.N.BitLen() < minKeySize {
		return nil, fmt.Errorf("private key must be at least %d bits (got %d bits)", minKeySize, privateKey.N.BitLen())
	}

	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
		config:     cfg,
	}, nil
}

// GenerateAccessToken generates a new access token for the given user.
func (s *Service) GenerateAccessToken(userID, email, role, sessionID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id cannot be empty")
	}

	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	if role == "" {
		return "", fmt.Errorf("role cannot be empty")
	}

	if sessionID == "" {
		return "", fmt.Errorf("session id cannot be empty")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.config.AccessTTL)

	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(), // Unique token ID (jti) for blacklisting
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a new refresh token for the given user.
func (s *Service) GenerateRefreshToken(userID, email, role, sessionID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id cannot be empty")
	}

	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	if role == "" {
		return "", fmt.Errorf("role cannot be empty")
	}

	if sessionID == "" {
		return "", fmt.Errorf("session id cannot be empty")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.config.RefreshTTL)

	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(), // Unique token ID (jti)
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns its claims.
// Returns an error if the token is invalid, expired, or has an invalid signature.
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Verify issuer
	if claims.Issuer != s.config.Issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", s.config.Issuer, claims.Issuer)
	}

	return claims, nil
}

// ExtractTokenID extracts the JWT ID (jti) from a token without full validation.
// This is useful for blacklist checks before full validation.
func (s *Service) ExtractTokenID(tokenString string) (string, error) {
	if tokenString == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Parse without validation to extract claims
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", fmt.Errorf("invalid claims type")
	}

	if claims.ID == "" {
		return "", fmt.Errorf("token has no ID")
	}

	return claims.ID, nil
}

// GetTokenExpiration extracts the expiration time from a token without full validation.
// This is useful for determining blacklist TTL.
func (s *Service) GetTokenExpiration(tokenString string) (time.Time, error) {
	if tokenString == "" {
		return time.Time{}, fmt.Errorf("token cannot be empty")
	}

	// Parse without validation to extract claims
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid claims type")
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, fmt.Errorf("token has no expiration")
	}

	return claims.ExpiresAt.Time, nil
}

// loadPrivateKey loads an RSA private key from a PEM file.
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	//nolint:gosec // G304: File path comes from trusted configuration, not user input
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected key type: %s", block.Type)
	}

	// Try parsing as PKCS#1 first
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Try parsing as PKCS#8
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return rsaKey, nil
}

// loadPublicKey loads an RSA public key from a PEM file.
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	//nolint:gosec // G304: File path comes from trusted configuration, not user input
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "RSA PUBLIC KEY" && block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected key type: %s", block.Type)
	}

	// Try parsing as PKIX first
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA public key")
		}
		return rsaKey, nil
	}

	// Try parsing as PKCS#1
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
}
