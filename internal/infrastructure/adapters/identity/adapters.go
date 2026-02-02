package identity

import (
	"context"
	"time"

	"github.com/google/uuid"

	appidentity "github.com/yegamble/goimg-datalayer/internal/application/identity"
	appservices "github.com/yegamble/goimg-datalayer/internal/application/identity/services"
	domidentity "github.com/yegamble/goimg-datalayer/internal/domain/identity"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/persistence/postgres"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/jwt"
)

// JWTServiceAdapter adapts jwt.Service to appservices.JWTService interface (for Login/Logout)
type JWTServiceAdapter struct {
	Service *jwt.Service
}

func (a *JWTServiceAdapter) GenerateAccessToken(userID, email, role, sessionID string) (string, error) {
	return a.Service.GenerateAccessToken(userID, email, role, sessionID)
}

func (a *JWTServiceAdapter) GenerateElevatedAccessToken(userID, email, role, sessionID string) (string, error) {
	return a.Service.GenerateElevatedAccessToken(userID, email, role, sessionID)
}

func (a *JWTServiceAdapter) GenerateRefreshToken(userID, email, role, sessionID string) (string, error) {
	return a.Service.GenerateRefreshToken(userID, email, role, sessionID)
}

func (a *JWTServiceAdapter) ValidateToken(tokenString string) (*appservices.JWTClaims, error) {
	claims, err := a.Service.ValidateToken(tokenString)
	if err != nil {
		return nil, err
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

func (a *JWTServiceAdapter) ExtractTokenID(tokenString string) (string, error) {
	return a.Service.ExtractTokenID(tokenString)
}

func (a *JWTServiceAdapter) GetTokenExpiration(tokenString string) (time.Time, error) {
	return a.Service.GetTokenExpiration(tokenString)
}

// JWTServiceAdapterIdentity adapts jwt.Service to appidentity.JWTService interface (for Guest Session)
type JWTServiceAdapterIdentity struct {
	Service *jwt.Service
}

func (a *JWTServiceAdapterIdentity) GenerateAccessToken(claims *appidentity.TokenClaims) (string, error) {
	return a.Service.GenerateAccessToken(
		claims.UserID.String(),
		claims.Email,
		claims.Role,
		claims.SessionID.String(),
	)
}

func (a *JWTServiceAdapterIdentity) GenerateRefreshToken(claims *appidentity.TokenClaims) (string, error) {
	return a.Service.GenerateRefreshToken(
		claims.UserID.String(),
		claims.Email,
		claims.Role,
		claims.SessionID.String(),
	)
}

func (a *JWTServiceAdapterIdentity) ValidateToken(token string) (*appidentity.TokenClaims, error) {
	claims, err := a.Service.ValidateToken(token)
	if err != nil {
		return nil, err
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

func (a *JWTServiceAdapterIdentity) ExtractTokenID(token string) (string, error) {
	return a.Service.ExtractTokenID(token)
}

func (a *JWTServiceAdapterIdentity) GetTokenExpiration(token string) (time.Time, error) {
	return a.Service.GetTokenExpiration(token)
}

// RefreshTokenServiceAdapter adapts jwt.RefreshTokenService to appservices.RefreshTokenService
type RefreshTokenServiceAdapter struct {
	Service *jwt.RefreshTokenService
}

func (a *RefreshTokenServiceAdapter) GenerateToken(
	ctx context.Context, userID, sessionID, familyID, parentHash, ip, userAgent string,
) (string, *appservices.RefreshTokenMetadata, error) {
	token, meta, err := a.Service.GenerateToken(ctx, userID, sessionID, familyID, parentHash, ip, userAgent)
	if err != nil {
		return "", nil, err
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

func (a *RefreshTokenServiceAdapter) ValidateToken(ctx context.Context, token string) (*appservices.RefreshTokenMetadata, error) {
	meta, err := a.Service.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
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

func (a *RefreshTokenServiceAdapter) MarkAsUsed(ctx context.Context, token string) error {
	return a.Service.MarkAsUsed(ctx, token)
}

func (a *RefreshTokenServiceAdapter) RevokeToken(ctx context.Context, token string) error {
	return a.Service.RevokeToken(ctx, token)
}

func (a *RefreshTokenServiceAdapter) RevokeFamily(ctx context.Context, familyID string) error {
	return a.Service.RevokeFamily(ctx, familyID)
}

func (a *RefreshTokenServiceAdapter) DetectAnomalies(metadata *appservices.RefreshTokenMetadata, currentIP, currentUserAgent string) bool {
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

// SessionStoreAdapter adapts postgres.SessionRepository to appservices.SessionStore
type SessionStoreAdapter struct {
	Repo *postgres.SessionRepository
}

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
	return a.Repo.Create(ctx, pgSession)
}

func (a *SessionStoreAdapter) Get(ctx context.Context, sessionID string) (*appservices.Session, error) {
	id, _ := uuid.Parse(sessionID)
	sess, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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

func (a *SessionStoreAdapter) Exists(ctx context.Context, sessionID string) (bool, error) {
	id, _ := uuid.Parse(sessionID)
	_, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (a *SessionStoreAdapter) Revoke(ctx context.Context, sessionID string) error {
	id, _ := uuid.Parse(sessionID)
	return a.Repo.Revoke(ctx, id)
}

func (a *SessionStoreAdapter) RevokeAll(ctx context.Context, userID string) error {
	uid, _ := domidentity.ParseUserID(userID)
	_, err := a.Repo.RevokeAllForUser(ctx, uid)
	return err
}

func (a *SessionStoreAdapter) GetUserSessions(ctx context.Context, userID string) ([]*appservices.Session, error) {
	uid, _ := domidentity.ParseUserID(userID)
	sessList, err := a.Repo.GetByUserID(ctx, uid)
	if err != nil {
		return nil, err
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

// SessionStoreAdapterIdentity adapts postgres.SessionRepository to appidentity.SessionStore
type SessionStoreAdapterIdentity struct {
	Repo *postgres.SessionRepository
}

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
	return a.Repo.Create(ctx, pgSession)
}

func (a *SessionStoreAdapterIdentity) Get(ctx context.Context, sessionID uuid.UUID) (*appidentity.Session, error) {
	sess, err := a.Repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
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

func (a *SessionStoreAdapterIdentity) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*appidentity.Session, error) {
	domUserID, _ := domidentity.ParseUserID(userID.String())
	sessList, err := a.Repo.GetByUserID(ctx, domUserID)
	if err != nil {
		return nil, err
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

func (a *SessionStoreAdapterIdentity) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	return a.Repo.Revoke(ctx, sessionID)
}

func (a *SessionStoreAdapterIdentity) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	domUserID, _ := domidentity.ParseUserID(userID.String())
	_, err := a.Repo.RevokeAllForUser(ctx, domUserID)
	return err
}

func (a *SessionStoreAdapterIdentity) Exists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	_, err := a.Repo.GetByID(ctx, sessionID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// TokenBlacklistAdapter for appservices.TokenBlacklist
type TokenBlacklistAdapter struct {
	Service *jwt.TokenBlacklist
}

func (a *TokenBlacklistAdapter) Add(ctx context.Context, tokenID string, expiresAt time.Time) error {
	return a.Service.Add(ctx, tokenID, expiresAt)
}

func (a *TokenBlacklistAdapter) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	return a.Service.IsBlacklisted(ctx, tokenID)
}

func (a *TokenBlacklistAdapter) Remove(ctx context.Context, tokenID string) error {
	return a.Service.Remove(ctx, tokenID)
}
