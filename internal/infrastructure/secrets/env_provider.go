package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// EnvProvider loads secrets from environment variables.
// This is the recommended provider for local development and testing.
//
// Environment variables are the standard configuration mechanism for
// containerized applications (12-factor apps) and work well in development.
// For production, consider using DockerSecretsProvider or a dedicated
// secrets management system like HashiCorp Vault or AWS Secrets Manager.
type EnvProvider struct{}

// NewEnvProvider creates a new environment variable secret provider.
func NewEnvProvider() *EnvProvider {
	log.Info().
		Str("provider", "env").
		Msg("Initialized environment variable secret provider")
	return &EnvProvider{}
}

// GetSecret retrieves a secret from an environment variable.
// Returns an error if the environment variable is not set.
func (p *EnvProvider) GetSecret(_ context.Context, name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("secret %s not found in environment variables", name)
	}

	log.Debug().
		Str("secret", name).
		Str("provider", "env").
		Msg("Retrieved secret from environment")

	return value, nil
}

// GetSecretWithDefault retrieves a secret from an environment variable,
// returning the default value if not set.
func (p *EnvProvider) GetSecretWithDefault(_ context.Context, name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		if defaultValue != "" {
			log.Debug().
				Str("secret", name).
				Str("provider", "env").
				Msg("Secret not found, using default value")
		}
		return defaultValue
	}

	log.Debug().
		Str("secret", name).
		Str("provider", "env").
		Msg("Retrieved secret from environment")

	return value
}

// MustGetSecret retrieves a secret from an environment variable and panics if not found.
// Use this during application initialization for required secrets.
func (p *EnvProvider) MustGetSecret(ctx context.Context, name string) string {
	value, err := p.GetSecret(ctx, name)
	if err != nil {
		log.Error().
			Str("secret", name).
			Str("provider", "env").
			Msg("Required secret not found in environment variables")
		panic(fmt.Sprintf("required secret %s not found", name))
	}
	return value
}

// ProviderName returns the name of this provider.
func (p *EnvProvider) ProviderName() string {
	return "env"
}

// ValidateRequiredSecrets checks that all required secrets are present.
// Returns an error listing all missing secrets, or nil if all are present.
// This should be called during application initialization.
func (p *EnvProvider) ValidateRequiredSecrets(_ context.Context) error {
	missing := make([]string, 0)

	for _, name := range RequiredSecrets() {
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required secrets: %v", missing)
	}

	log.Info().
		Int("count", len(RequiredSecrets())).
		Str("provider", "env").
		Msg("All required secrets validated")

	return nil
}

// ListAvailableSecrets returns information about which secrets are available.
// This is useful for debugging configuration issues.
// WARNING: Never log the actual secret values!
func (p *EnvProvider) ListAvailableSecrets(_ context.Context) map[string]bool {
	secrets := make(map[string]bool)

	// Check required secrets
	for _, name := range RequiredSecrets() {
		secrets[name] = os.Getenv(name) != ""
	}

	// Check optional secrets
	for _, name := range OptionalSecrets() {
		secrets[name] = os.Getenv(name) != ""
	}

	return secrets
}
