package secrets

import (
	"context"
	"fmt"
)

// SecretProvider defines the interface for retrieving secrets from various sources.
// Implementations can load secrets from environment variables, Docker Secrets,
// AWS Secrets Manager, HashiCorp Vault, or other secret management systems.
type SecretProvider interface {
	// GetSecret retrieves a secret by name.
	// Returns the secret value or an error if not found or inaccessible.
	GetSecret(ctx context.Context, name string) (string, error)

	// GetSecretWithDefault retrieves a secret by name, returning a default value if not found.
	// This is useful for optional secrets like REDIS_PASSWORD in development.
	GetSecretWithDefault(ctx context.Context, name, defaultValue string) string

	// MustGetSecret retrieves a secret by name and panics if not found.
	// Use this for required secrets during application initialization.
	MustGetSecret(ctx context.Context, name string) string

	// ProviderName returns the name of the provider for logging/debugging.
	ProviderName() string
}

// SecretConfig holds configuration for the secret provider.
type SecretConfig struct {
	// Provider specifies which provider to use: "env", "docker", "vault", etc.
	Provider string

	// DockerSecretsPath is the filesystem path where Docker Secrets are mounted.
	// Default: /run/secrets/
	DockerSecretsPath string

	// FailFast determines whether to panic on missing required secrets during initialization.
	// Recommended: true for production to catch configuration errors early.
	FailFast bool
}

// NewProvider creates a SecretProvider based on the configuration.
// It selects the appropriate provider implementation based on the Provider field.
func NewProvider(config SecretConfig) (SecretProvider, error) {
	switch config.Provider {
	case "env", "environment":
		return NewEnvProvider(), nil
	case "docker", "docker-secrets":
		path := config.DockerSecretsPath
		if path == "" {
			path = "/run/secrets"
		}
		return NewDockerSecretsProvider(path), nil
	default:
		return nil, fmt.Errorf("unknown secret provider: %s (supported: env, docker)", config.Provider)
	}
}

// SecretName constants for all secrets used in the application.
// This provides type safety and prevents typos when requesting secrets.
const (
	// Authentication & Authorization.
	SecretJWT = "JWT_SECRET"

	// Database.
	SecretDBPassword = "DB_PASSWORD"

	// Redis
	SecretRedisPassword = "REDIS_PASSWORD"

	// Object Storage (S3-compatible)
	SecretS3AccessKey = "S3_ACCESS_KEY"
	SecretS3SecretKey = "S3_SECRET_KEY"

	// IPFS Pinning Services
	SecretIPFSPinataJWT       = "IPFS_PINATA_JWT"
	SecretIPFSInfuraProjectID = "IPFS_INFURA_PROJECT_ID"
	SecretIPFSInfuraSecret    = "IPFS_INFURA_PROJECT_SECRET"

	// OAuth2 Providers
	SecretOAuthGoogleClientID     = "OAUTH_GOOGLE_CLIENT_ID"
	SecretOAuthGoogleClientSecret = "OAUTH_GOOGLE_CLIENT_SECRET"
	SecretOAuthGitHubClientID     = "OAUTH_GITHUB_CLIENT_ID"
	SecretOAuthGitHubClientSecret = "OAUTH_GITHUB_CLIENT_SECRET"

	// SMTP Email
	SecretSMTPPassword = "SMTP_PASSWORD"

	// Monitoring (Grafana)
	SecretGrafanaAdminPassword = "GRAFANA_ADMIN_PASSWORD"

	// Backup Encryption
	SecretBackupS3AccessKey = "BACKUP_S3_ACCESS_KEY"
	SecretBackupS3SecretKey = "BACKUP_S3_SECRET_KEY"
)

// RequiredSecrets returns the list of secrets that MUST be present for the application to start.
// Missing required secrets will cause the application to fail fast during initialization.
func RequiredSecrets() []string {
	return []string{
		SecretJWT,
		SecretDBPassword,
		// Note: REDIS_PASSWORD is optional in development (Redis without auth)
		// but should be required in production
	}
}

// OptionalSecrets returns the list of secrets that are optional.
// These secrets enable additional features but are not required for core functionality.
func OptionalSecrets() []string {
	return []string{
		SecretRedisPassword,
		SecretS3AccessKey,
		SecretS3SecretKey,
		SecretIPFSPinataJWT,
		SecretIPFSInfuraProjectID,
		SecretIPFSInfuraSecret,
		SecretOAuthGoogleClientID,
		SecretOAuthGoogleClientSecret,
		SecretOAuthGitHubClientID,
		SecretOAuthGitHubClientSecret,
		SecretSMTPPassword,
		SecretGrafanaAdminPassword,
		SecretBackupS3AccessKey,
		SecretBackupS3SecretKey,
	}
}
