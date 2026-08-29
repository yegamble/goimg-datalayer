package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"

	defaultAccessTTL = 15 * time.Minute
	minKeySize       = 4096
)

type Config struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	Issuer         string
}

func DefaultConfig() Config {
	return Config{
		PrivateKeyPath: "",
		PublicKeyPath:  "",
		AccessTTL:      defaultAccessTTL,
		RefreshTTL:     7 * 24 * time.Hour,
		Issuer:         "goimg-api",
	}
}

type Claims struct {
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	SessionID     string    `json:"session_id"`
	TokenType     TokenType `json:"token_type"`
	TwoFAVerified bool      `json:"twofa_verified,omitempty"`
	EmailVerified bool      `json:"email_verified,omitempty"`
	jwt.RegisteredClaims
}

type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	config     Config
}

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

	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	if privateKey.N.BitLen() < minKeySize {
		return nil, fmt.Errorf("private key must be at least %d bits (got %d bits)", minKeySize, privateKey.N.BitLen())
	}

	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
		config:     cfg,
	}, nil
}

//nolint:dupl // Access and refresh token generation are intentionally similar but distinct
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
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

func (s *Service) GenerateElevatedAccessToken(userID, email, role, sessionID string) (string, error) {
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
		UserID:        userID,
		Email:         email,
		Role:          role,
		SessionID:     sessionID,
		TokenType:     TokenTypeAccess,
		TwoFAVerified: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign elevated access token: %w", err)
	}

	return signedToken, nil
}

//nolint:dupl // Access and refresh token generation are intentionally similar but distinct
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
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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

	if claims.Issuer != s.config.Issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", s.config.Issuer, claims.Issuer)
	}

	return claims, nil
}

func (s *Service) ExtractTokenID(tokenString string) (string, error) {
	if tokenString == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

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

func (s *Service) GetTokenExpiration(tokenString string) (time.Time, error) {
	if tokenString == "" {
		return time.Time{}, fmt.Errorf("token cannot be empty")
	}

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

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	// #nosec G304 - path is securely provided by application configuration
	keyData, err := os.ReadFile(filepath.Clean(path))
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

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

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

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	// #nosec G304 - path is securely provided by application configuration
	keyData, err := os.ReadFile(filepath.Clean(path))
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

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA public key")
		}
		return rsaKey, nil
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
}
