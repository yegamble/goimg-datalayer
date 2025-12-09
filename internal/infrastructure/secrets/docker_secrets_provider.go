package secrets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// DockerSecretsProvider loads secrets from Docker Secrets.
// Docker Secrets are mounted as files in /run/secrets/ (by default).
//
// Docker Swarm and Kubernetes both support mounting secrets as files,
// making this provider suitable for production container orchestration.
//
// Each secret is stored in a separate file named after the secret:
//   - /run/secrets/JWT_SECRET
//   - /run/secrets/DB_PASSWORD
//   - etc.
//
// Benefits over environment variables:
//  1. Secrets are not visible in `docker inspect` or process listings
//  2. Secrets are not logged or included in error messages
//  3. Access can be restricted with filesystem permissions
//  4. Secrets can be rotated without rebuilding containers
//  5. Better integration with orchestration platforms (Swarm, Kubernetes)
type DockerSecretsProvider struct {
	secretsPath string
	cache       map[string]string // In-memory cache for performance
}

// NewDockerSecretsProvider creates a new Docker Secrets provider.
// The secretsPath parameter specifies where secrets are mounted (default: /run/secrets).
func NewDockerSecretsProvider(secretsPath string) *DockerSecretsProvider {
	if secretsPath == "" {
		secretsPath = "/run/secrets"
	}

	log.Info().
		Str("provider", "docker-secrets").
		Str("path", secretsPath).
		Msg("Initialized Docker Secrets provider")

	return &DockerSecretsProvider{
		secretsPath: secretsPath,
		cache:       make(map[string]string),
	}
}

// GetSecret retrieves a secret from a Docker Secret file.
// Returns an error if the file does not exist or cannot be read.
func (p *DockerSecretsProvider) GetSecret(_ context.Context, name string) (string, error) {
	// Check cache first
	if value, ok := p.cache[name]; ok {
		log.Debug().
			Str("secret", name).
			Str("provider", "docker-secrets").
			Msg("Retrieved secret from cache")
		return value, nil
	}

	// Read from filesystem
	secretFile := filepath.Join(p.secretsPath, name)

	//nolint:gosec // G304: File path from trusted configuration source (secretsPath is validated at initialization)
	data, err := os.ReadFile(secretFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("secret %s not found at %s", name, secretFile)
		}
		return "", fmt.Errorf("failed to read secret %s: %w", name, err)
	}

	// Trim whitespace (Docker Secrets often have trailing newlines)
	value := strings.TrimSpace(string(data))

	if value == "" {
		return "", fmt.Errorf("secret %s is empty at %s", name, secretFile)
	}

	// Cache the value
	p.cache[name] = value

	log.Debug().
		Str("secret", name).
		Str("provider", "docker-secrets").
		Str("path", secretFile).
		Msg("Retrieved secret from Docker Secrets file")

	return value, nil
}

// GetSecretWithDefault retrieves a secret from Docker Secrets,
// returning the default value if not found.
func (p *DockerSecretsProvider) GetSecretWithDefault(ctx context.Context, name, defaultValue string) string {
	value, err := p.GetSecret(ctx, name)
	if err != nil {
		if defaultValue != "" {
			log.Debug().
				Str("secret", name).
				Str("provider", "docker-secrets").
				Err(err).
				Msg("Secret not found, using default value")
		}
		return defaultValue
	}
	return value
}

// MustGetSecret retrieves a secret from Docker Secrets and panics if not found.
// Use this during application initialization for required secrets.
func (p *DockerSecretsProvider) MustGetSecret(ctx context.Context, name string) string {
	value, err := p.GetSecret(ctx, name)
	if err != nil {
		log.Error().
			Str("secret", name).
			Str("provider", "docker-secrets").
			Err(err).
			Msg("Required secret not found in Docker Secrets")
		panic(fmt.Sprintf("required secret %s not found: %v", name, err))
	}
	return value
}

// ProviderName returns the name of this provider.
func (p *DockerSecretsProvider) ProviderName() string {
	return "docker-secrets"
}

// ValidateRequiredSecrets checks that all required secrets are present.
// Returns an error listing all missing secrets, or nil if all are present.
// This should be called during application initialization.
func (p *DockerSecretsProvider) ValidateRequiredSecrets(_ context.Context) error {
	missing := make([]string, 0)

	for _, name := range RequiredSecrets() {
		secretFile := filepath.Join(p.secretsPath, name)
		if _, err := os.Stat(secretFile); os.IsNotExist(err) {
			missing = append(missing, name)
		} else if err != nil {
			log.Warn().
				Str("secret", name).
				Str("path", secretFile).
				Err(err).
				Msg("Error checking secret file")
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required Docker Secrets: %v", missing)
	}

	log.Info().
		Int("count", len(RequiredSecrets())).
		Str("provider", "docker-secrets").
		Msg("All required secrets validated")

	return nil
}

// ListAvailableSecrets returns information about which secrets are available.
// This is useful for debugging configuration issues.
// WARNING: Never log the actual secret values!
func (p *DockerSecretsProvider) ListAvailableSecrets(_ context.Context) map[string]bool {
	secrets := make(map[string]bool)

	// Check required secrets
	for _, name := range RequiredSecrets() {
		secretFile := filepath.Join(p.secretsPath, name)
		_, err := os.Stat(secretFile)
		secrets[name] = err == nil
	}

	// Check optional secrets
	for _, name := range OptionalSecrets() {
		secretFile := filepath.Join(p.secretsPath, name)
		_, err := os.Stat(secretFile)
		secrets[name] = err == nil
	}

	return secrets
}

// ClearCache clears the in-memory secret cache.
// Call this after rotating secrets to force re-reading from the filesystem.
func (p *DockerSecretsProvider) ClearCache() {
	p.cache = make(map[string]string)
	log.Info().
		Str("provider", "docker-secrets").
		Msg("Cleared secret cache")
}

// RefreshSecret clears a specific secret from the cache.
// Call this after rotating a specific secret to force re-reading.
func (p *DockerSecretsProvider) RefreshSecret(name string) {
	delete(p.cache, name)
	log.Info().
		Str("secret", name).
		Str("provider", "docker-secrets").
		Msg("Refreshed secret from cache")
}
