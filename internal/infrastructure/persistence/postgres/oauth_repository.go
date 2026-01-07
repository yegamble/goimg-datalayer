package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/yegamble/goimg-datalayer/internal/domain/identity"
)

// SQL queries for OAuth account operations.
const (
	sqlInsertOAuthAccount = `
		INSERT INTO oauth_accounts (
			id, user_id, provider, provider_user_id, email, display_name, avatar_url,
			access_token_encrypted, refresh_token_encrypted, token_expires_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	sqlUpdateOAuthAccount = `
		UPDATE oauth_accounts
		SET email = $3,
		    display_name = $4,
		    avatar_url = $5,
		    access_token_encrypted = $6,
		    refresh_token_encrypted = $7,
		    token_expires_at = $8,
		    updated_at = $9
		WHERE id = $1 AND user_id = $2
	`

	sqlSelectOAuthAccountByID = `
		SELECT id, user_id, provider, provider_user_id, email, display_name, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM oauth_accounts
		WHERE id = $1
	`

	sqlSelectOAuthAccountByProviderAndUserID = `
		SELECT id, user_id, provider, provider_user_id, email, display_name, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM oauth_accounts
		WHERE provider = $1 AND provider_user_id = $2
	`

	sqlSelectOAuthAccountsByUserID = `
		SELECT id, user_id, provider, provider_user_id, email, display_name, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM oauth_accounts
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	sqlSelectOAuthAccountByUserIDAndProvider = `
		SELECT id, user_id, provider, provider_user_id, email, display_name, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM oauth_accounts
		WHERE user_id = $1 AND provider = $2
	`

	sqlDeleteOAuthAccount = `
		DELETE FROM oauth_accounts
		WHERE id = $1
	`

	sqlCheckOAuthAccountExists = `
		SELECT EXISTS(
			SELECT 1 FROM oauth_accounts
			WHERE provider = $1 AND provider_user_id = $2
		)
	`
)

// oauthAccountRow represents an OAuth account row in the database.
type oauthAccountRow struct {
	ID                    string         `db:"id"`
	UserID                string         `db:"user_id"`
	Provider              string         `db:"provider"`
	ProviderUserID        string         `db:"provider_user_id"`
	Email                 string         `db:"email"`
	DisplayName           sql.NullString `db:"display_name"`
	AvatarURL             sql.NullString `db:"avatar_url"`
	AccessTokenEncrypted  []byte         `db:"access_token_encrypted"`
	RefreshTokenEncrypted []byte         `db:"refresh_token_encrypted"`
	TokenExpiresAt        sql.NullTime   `db:"token_expires_at"`
	CreatedAt             time.Time      `db:"created_at"`
	UpdatedAt             time.Time      `db:"updated_at"`
}

// toDomain converts a database row to a domain OAuth account.
func (r *oauthAccountRow) toDomain() (*identity.OAuthAccount, error) {
	id, err := identity.ParseOAuthAccountID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth account id: %w", err)
	}

	userID, err := identity.ParseUserID(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	provider, err := identity.ParseOAuthProvider(r.Provider)
	if err != nil {
		return nil, fmt.Errorf("parse provider: %w", err)
	}

	providerUserID, err := identity.NewProviderUserID(r.ProviderUserID)
	if err != nil {
		return nil, fmt.Errorf("parse provider user id: %w", err)
	}

	email, err := identity.NewEmail(r.Email)
	if err != nil {
		return nil, fmt.Errorf("parse email: %w", err)
	}

	var displayName string
	if r.DisplayName.Valid {
		displayName = r.DisplayName.String
	}

	var avatarURL string
	if r.AvatarURL.Valid {
		avatarURL = r.AvatarURL.String
	}

	var tokenExpiresAt *time.Time
	if r.TokenExpiresAt.Valid {
		tokenExpiresAt = &r.TokenExpiresAt.Time
	}

	return identity.ReconstructOAuthAccount(
		id,
		userID,
		provider,
		providerUserID,
		email,
		displayName,
		avatarURL,
		r.AccessTokenEncrypted,
		r.RefreshTokenEncrypted,
		tokenExpiresAt,
		r.CreatedAt,
		r.UpdatedAt,
	), nil
}

// fromDomain converts a domain OAuth account to a database row.
func oauthAccountFromDomain(account *identity.OAuthAccount) *oauthAccountRow {
	row := &oauthAccountRow{
		ID:                    account.ID().String(),
		UserID:                account.UserID().String(),
		Provider:              account.Provider().String(),
		ProviderUserID:        account.ProviderUserID().String(),
		Email:                 account.Email().String(),
		AccessTokenEncrypted:  account.EncryptedAccessToken(),
		RefreshTokenEncrypted: account.EncryptedRefreshToken(),
		CreatedAt:             account.CreatedAt(),
		UpdatedAt:             account.UpdatedAt(),
	}

	if account.DisplayName() != "" {
		row.DisplayName = sql.NullString{String: account.DisplayName(), Valid: true}
	}

	if account.AvatarURL() != "" {
		row.AvatarURL = sql.NullString{String: account.AvatarURL(), Valid: true}
	}

	if account.TokenExpiresAt() != nil {
		row.TokenExpiresAt = sql.NullTime{Time: *account.TokenExpiresAt(), Valid: true}
	}

	return row
}

// OAuthAccountRepository handles persistence of OAuth accounts.
type OAuthAccountRepository struct {
	db *sqlx.DB
}

// NewOAuthAccountRepository creates a new OAuth account repository.
func NewOAuthAccountRepository(db *sqlx.DB) *OAuthAccountRepository {
	return &OAuthAccountRepository{db: db}
}

// FindByID retrieves an OAuth account by its unique ID.
func (r *OAuthAccountRepository) FindByID(ctx context.Context, id identity.OAuthAccountID) (*identity.OAuthAccount, error) {
	var row oauthAccountRow
	err := r.db.GetContext(ctx, &row, sqlSelectOAuthAccountByID, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrOAuthAccountNotFound
		}
		return nil, fmt.Errorf("failed to find OAuth account by id: %w", err)
	}

	account, err := row.toDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert OAuth account to domain: %w", err)
	}

	return account, nil
}

// FindByProviderAndUserID retrieves an OAuth account by provider and provider user ID.
func (r *OAuthAccountRepository) FindByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (*identity.OAuthAccount, error) {
	var row oauthAccountRow
	err := r.db.GetContext(ctx, &row, sqlSelectOAuthAccountByProviderAndUserID,
		provider.String(), providerUserID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrOAuthAccountNotFound
		}
		return nil, fmt.Errorf("failed to find OAuth account by provider and user id: %w", err)
	}

	account, err := row.toDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert OAuth account to domain: %w", err)
	}

	return account, nil
}

// FindByUserID retrieves all OAuth accounts for a user.
func (r *OAuthAccountRepository) FindByUserID(ctx context.Context, userID identity.UserID) ([]*identity.OAuthAccount, error) {
	var rows []oauthAccountRow
	err := r.db.SelectContext(ctx, &rows, sqlSelectOAuthAccountsByUserID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find OAuth accounts by user id: %w", err)
	}

	accounts := make([]*identity.OAuthAccount, 0, len(rows))
	for _, row := range rows {
		account, err := row.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert OAuth account to domain: %w", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// FindByUserIDAndProvider retrieves a specific OAuth account for a user and provider.
func (r *OAuthAccountRepository) FindByUserIDAndProvider(
	ctx context.Context,
	userID identity.UserID,
	provider identity.OAuthProvider,
) (*identity.OAuthAccount, error) {
	var row oauthAccountRow
	err := r.db.GetContext(ctx, &row, sqlSelectOAuthAccountByUserIDAndProvider,
		userID.String(), provider.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, identity.ErrOAuthProviderNotLinked
		}
		return nil, fmt.Errorf("failed to find OAuth account by user id and provider: %w", err)
	}

	account, err := row.toDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert OAuth account to domain: %w", err)
	}

	return account, nil
}

// Save persists an OAuth account to the repository.
// If the account already exists (same ID), it is updated; otherwise, it is created.
func (r *OAuthAccountRepository) Save(ctx context.Context, account *identity.OAuthAccount) error {
	row := oauthAccountFromDomain(account)

	// Try to insert first
	_, err := r.db.ExecContext(
		ctx,
		sqlInsertOAuthAccount,
		row.ID,
		row.UserID,
		row.Provider,
		row.ProviderUserID,
		row.Email,
		row.DisplayName,
		row.AvatarURL,
		row.AccessTokenEncrypted,
		row.RefreshTokenEncrypted,
		row.TokenExpiresAt,
		row.CreatedAt,
		row.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				if pqErr.Constraint == "uq_oauth_provider_user" {
					return identity.ErrOAuthAccountExists
				}
			}
		}

		// If not a constraint violation, try update
		result, updateErr := r.db.ExecContext(
			ctx,
			sqlUpdateOAuthAccount,
			row.ID,
			row.UserID,
			row.Email,
			row.DisplayName,
			row.AvatarURL,
			row.AccessTokenEncrypted,
			row.RefreshTokenEncrypted,
			row.TokenExpiresAt,
			row.UpdatedAt,
		)

		if updateErr != nil {
			return fmt.Errorf("failed to save OAuth account: insert error: %w, update error: %v", err, updateErr)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return identity.ErrOAuthAccountNotFound
		}

		return nil
	}

	return nil
}

// Delete removes an OAuth account from the repository.
func (r *OAuthAccountRepository) Delete(ctx context.Context, id identity.OAuthAccountID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteOAuthAccount, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete OAuth account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return identity.ErrOAuthAccountNotFound
	}

	return nil
}

// ExistsByProviderAndUserID checks if an OAuth account exists for the provider and provider user ID.
func (r *OAuthAccountRepository) ExistsByProviderAndUserID(
	ctx context.Context,
	provider identity.OAuthProvider,
	providerUserID identity.ProviderUserID,
) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, sqlCheckOAuthAccountExists,
		provider.String(), providerUserID.String())
	if err != nil {
		return false, fmt.Errorf("failed to check OAuth account existence: %w", err)
	}
	return exists, nil
}
