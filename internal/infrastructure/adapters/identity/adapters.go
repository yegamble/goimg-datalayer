// Package identity provides adapter implementations for the identity bounded context.
package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	appservices "github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	domidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// JWTServiceAdapter adapts jwt.Service to appservices.JWTService interface (for Login/Logout).
type JWTServiceAdapter struct {
	Service *jwt.Service
}

// GenerateAccessToken generates a new JWT access token.
func (a *JWTServiceAdapter) GenerateAccessToken(userID, email, role, sessionID string) (string, error) {
	token, err := a.Service.GenerateAccessToken(userID, email, role, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}
	return token, nil
}

// GenerateElevatedAccessToken generates a new elevated JWT access token.
func (a *JWTServiceAdapter) GenerateElevatedAccessToken(userID, email, role, sessionID string) (string, error) {
	token, err := a.Service.GenerateElevatedAccessToken(userID, email, role, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to generate elevated access token: %w", err)
	}
	return token, nil
}

// GenerateRefreshToken generates a new JWT refresh token.
func (a *JWTServiceAdapter) GenerateRefreshToken(userID, email, role, sessionID string) (string, error) {
	token, err := a.Service.GenerateRefreshToken(userID, email, role, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return token, nil
}

// ValidateToken validates the given JWT token string and returns the claims.
func (a *JWTServiceAdapter) ValidateToken(tokenString string) (*appservices.JWTClaims, error) {
	claims, err := a.Service.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	return &appservices.JWTClaims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID,
		TokenType: string(claims.TokenType),
		JTI:       claims.ID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

// ExtractTokenID extracts the JTI from the token string without validation.
func (a *JWTServiceAdapter) ExtractTokenID(tokenString string) (string, error) {
	id, err := a.Service.ExtractTokenID(tokenString)
	if err != nil {
		return "", fmt.Errorf("failed to extract token ID: %w", err)
	}
	return id, nil
}

// GetTokenExpiration extracts the expiration time from the token string.
func (a *JWTServiceAdapter) GetTokenExpiration(tokenString string) (time.Time, error) {
	exp, err := a.Service.GetTokenExpiration(tokenString)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get token expiration: %w", err)
	}
	return exp, nil
}

// JWTServiceAdapterIdentity adapts jwt.Service to appidentity.JWTService interface (for Guest Session).
type JWTServiceAdapterIdentity struct {
	Service *jwt.Service
}

// GenerateAccessToken generates a new JWT access token from claims.
func (a *JWTServiceAdapterIdentity) GenerateAccessToken(claims *appidentity.TokenClaims) (string, error) {
	token, err := a.Service.GenerateAccessToken(
		claims.UserID.String(),
		claims.Email,
		claims.Role,
		claims.SessionID.String(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token from claims: %w", err)
	}
	return token, nil
}

// GenerateRefreshToken generates a new JWT refresh token from claims.
func (a *JWTServiceAdapterIdentity) GenerateRefreshToken(claims *appidentity.TokenClaims) (string, error) {
	token, err := a.Service.GenerateRefreshToken(
		claims.UserID.String(),
		claims.Email,
		claims.Role,
		claims.SessionID.String(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token from claims: %w", err)
	}
	return token, nil
}

// ValidateToken validates the given token string and returns the identity claims.
func (a *JWTServiceAdapterIdentity) ValidateToken(token string) (*appidentity.TokenClaims, error) {
	claims, err := a.Service.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	uid, _ := uuid.Parse(claims.UserID)
	sid, _ := uuid.Parse(claims.SessionID)

	return &appidentity.TokenClaims{
		UserID:    uid,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: sid,
		TokenType: string(claims.TokenType),
	}, nil
}

// ExtractTokenID extracts the JTI from the token string.
func (a *JWTServiceAdapterIdentity) ExtractTokenID(token string) (string, error) {
	id, err := a.Service.ExtractTokenID(token)
	if err != nil {
		return "", fmt.Errorf("failed to extract token ID: %w", err)
	}
	return id, nil
}

// GetTokenExpiration extracts the expiration time from the token string.
func (a *JWTServiceAdapterIdentity) GetTokenExpiration(token string) (time.Time, error) {
	exp, err := a.Service.GetTokenExpiration(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get token expiration: %w", err)
	}
	return exp, nil
}

// RefreshTokenServiceAdapter adapts jwt.RefreshTokenService to appservices.RefreshTokenService.
type RefreshTokenServiceAdapter struct {
	Service *jwt.RefreshTokenService
}

// GenerateToken generates a new refresh token and stores its metadata.
func (a *RefreshTokenServiceAdapter) GenerateToken(
	ctx context.Context, userID, sessionID, familyID, parentHash, ip, userAgent string,
) (string, *appservices.RefreshTokenMetadata, error) {
	token, meta, err := a.Service.GenerateToken(ctx, userID, sessionID, familyID, parentHash, ip, userAgent)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return token, &appservices.RefreshTokenMetadata{
		TokenHash:  meta.TokenHash,
		UserID:     meta.UserID,
		SessionID:  meta.SessionID,
		FamilyID:   meta.FamilyID,
		IssuedAt:   meta.IssuedAt,
		ExpiresAt:  meta.ExpiresAt,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
		ParentHash: meta.ParentHash,
		Used:       meta.Used,
	}, nil
}

// ValidateToken validates the refresh token against stored metadata.
func (a *RefreshTokenServiceAdapter) ValidateToken(
	ctx context.Context, token string,
) (*appservices.RefreshTokenMetadata, error) {
	meta, err := a.Service.ValidateToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}
	return &appservices.RefreshTokenMetadata{
		TokenHash:  meta.TokenHash,
		UserID:     meta.UserID,
		SessionID:  meta.SessionID,
		FamilyID:   meta.FamilyID,
		IssuedAt:   meta.IssuedAt,
		ExpiresAt:  meta.ExpiresAt,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
		ParentHash: meta.ParentHash,
		Used:       meta.Used,
	}, nil
}

// MarkAsUsed marks the refresh token as used.
func (a *RefreshTokenServiceAdapter) MarkAsUsed(ctx context.Context, token string) error {
	if err := a.Service.MarkAsUsed(ctx, token); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	return nil
}

// RevokeToken revokes a specific refresh token.
func (a *RefreshTokenServiceAdapter) RevokeToken(ctx context.Context, token string) error {
	if err := a.Service.RevokeToken(ctx, token); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// RevokeFamily revokes all refresh tokens in a family.
func (a *RefreshTokenServiceAdapter) RevokeFamily(ctx context.Context, familyID string) error {
	if err := a.Service.RevokeFamily(ctx, familyID); err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}
	return nil
}

// DetectAnomalies checks for suspicious activity based on token usage.
func (a *RefreshTokenServiceAdapter) DetectAnomalies(
	metadata *appservices.RefreshTokenMetadata, currentIP, currentUserAgent string,
) bool {
	infraMeta := &jwt.RefreshTokenMetadata{
		TokenHash:  metadata.TokenHash,
		UserID:     metadata.UserID,
		SessionID:  metadata.SessionID,
		FamilyID:   metadata.FamilyID,
		IssuedAt:   metadata.IssuedAt,
		ExpiresAt:  metadata.ExpiresAt,
		IP:         metadata.IP,
		UserAgent:  metadata.UserAgent,
		ParentHash: metadata.ParentHash,
		Used:       metadata.Used,
	}
	return a.Service.DetectAnomalies(infraMeta, currentIP, currentUserAgent)
}

// SessionStoreAdapter adapts postgres.SessionRepository to appservices.SessionStore.
type SessionStoreAdapter struct {
	Repo *postgres.SessionRepository
}

// Create creates a new session.
func (a *SessionStoreAdapter) Create(ctx context.Context, session appservices.Session) error {
	sessID, _ := uuid.Parse(session.SessionID)
	userID, _ := domidentity.ParseUserID(session.UserID)

	pgSession := &postgres.Session{
		ID:        sessID,
		UserID:    userID,
		IPAddress: session.IP,
		UserAgent: session.UserAgent,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
	if err := a.Repo.Create(ctx, pgSession); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID.
func (a *SessionStoreAdapter) Get(ctx context.Context, sessionID string) (*appservices.Session, error) {
	id, _ := uuid.Parse(sessionID)
	sess, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &appservices.Session{
		SessionID: sess.ID.String(),
		UserID:    sess.UserID.String(),
		IP:        sess.IPAddress,
		UserAgent: sess.UserAgent,
		CreatedAt: sess.CreatedAt,
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

// Exists checks if a session exists.
func (a *SessionStoreAdapter) Exists(ctx context.Context, sessionID string) (bool, error) {
	id, _ := uuid.Parse(sessionID)
	_, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		// Check for specific not found error if available, otherwise assume error
		if errors.Is(err, sql.ErrNoRows) { // Assuming ErrNoRows or similar
			return false, nil
		}
		// Return error for DB failures instead of masking as false
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return true, nil
}

// Revoke revokes a session by ID.
func (a *SessionStoreAdapter) Revoke(ctx context.Context, sessionID string) error {
	id, _ := uuid.Parse(sessionID)
	if err := a.Repo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

// RevokeAll revokes all sessions for a user.
func (a *SessionStoreAdapter) RevokeAll(ctx context.Context, userID string) error {
	uid, _ := domidentity.ParseUserID(userID)
	if _, err := a.Repo.RevokeAllForUser(ctx, uid); err != nil {
		return fmt.Errorf("failed to revoke all sessions: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all active sessions for a user.
func (a *SessionStoreAdapter) GetUserSessions(ctx context.Context, userID string) ([]*appservices.Session, error) {
	uid, _ := domidentity.ParseUserID(userID)
	sessList, err := a.Repo.GetByUserID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	result := make([]*appservices.Session, len(sessList))
	for i, sess := range sessList {
		result[i] = &appservices.Session{
			SessionID: sess.ID.String(),
			UserID:    sess.UserID.String(),
			IP:        sess.IPAddress,
			UserAgent: sess.UserAgent,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		}
	}
	return result, nil
}

// SessionStoreAdapterIdentity adapts postgres.SessionRepository to appidentity.SessionStore.
type SessionStoreAdapterIdentity struct {
	Repo *postgres.SessionRepository
}

// Create creates a new session.
func (a *SessionStoreAdapterIdentity) Create(ctx context.Context, session *appidentity.Session) error {
	domUserID, _ := domidentity.ParseUserID(session.UserID.String())
	pgSession := &postgres.Session{
		ID:        session.ID,
		UserID:    domUserID,
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
	if err := a.Repo.Create(ctx, pgSession); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID.
func (a *SessionStoreAdapterIdentity) Get(ctx context.Context, sessionID uuid.UUID) (*appidentity.Session, error) {
	sess, err := a.Repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &appidentity.Session{
		ID:        sess.ID,
		UserID:    sess.UserID.UUID(),
		IPAddress: sess.IPAddress,
		UserAgent: sess.UserAgent,
		CreatedAt: sess.CreatedAt,
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

// GetUserSessions retrieves all active sessions for a user.
func (a *SessionStoreAdapterIdentity) GetUserSessions(
	ctx context.Context, userID uuid.UUID,
) ([]*appidentity.Session, error) {
	domUserID, _ := domidentity.ParseUserID(userID.String())
	sessList, err := a.Repo.GetByUserID(ctx, domUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	result := make([]*appidentity.Session, len(sessList))
	for i, sess := range sessList {
		result[i] = &appidentity.Session{
			ID:        sess.ID,
			UserID:    sess.UserID.UUID(),
			IPAddress: sess.IPAddress,
			UserAgent: sess.UserAgent,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
		}
	}
	return result, nil
}

// Revoke revokes a session by ID.
func (a *SessionStoreAdapterIdentity) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	if err := a.Repo.Revoke(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

// RevokeAll revokes all sessions for a user.
func (a *SessionStoreAdapterIdentity) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	domUserID, _ := domidentity.ParseUserID(userID.String())
	if _, err := a.Repo.RevokeAllForUser(ctx, domUserID); err != nil {
		return fmt.Errorf("failed to revoke all sessions: %w", err)
	}
	return nil
}

// Exists checks if a session exists.
func (a *SessionStoreAdapterIdentity) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	_, err := a.Repo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return true, nil
}

// TokenBlacklistAdapter for appservices.TokenBlacklist.
type TokenBlacklistAdapter struct {
	Service *jwt.TokenBlacklist
}

// Add adds a token to the blacklist.
func (a *TokenBlacklistAdapter) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	if err := a.Service.Add(ctx, tokenID, expiresAt); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// IsBlacklisted checks if a token is blacklisted.
func (a *TokenBlacklistAdapter) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	blacklisted, err := a.Service.IsBlacklisted(ctx, tokenID)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return blacklisted, nil
}

// Remove removes a token from the blacklist.
func (a *TokenBlacklistAdapter) Remove(ctx context.Context, tokenID string) error {
	if err := a.Service.Remove(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to remove token from blacklist: %w", err)
	}
	return nil
}
