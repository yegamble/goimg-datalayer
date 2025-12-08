//nolint:testpackage // White-box testing required for internal implementation
package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestSecrets(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "docker-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir
}

func writeSecret(t *testing.T, dir, name, value string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(value), 0o600)
	if err != nil {
		t.Fatalf("Failed to write secret %s: %v", name, err)
	}
}

func TestDockerSecretsProvider_GetSecret(t *testing.T) {
	tmpDir := setupTestSecrets(t)
	defer os.RemoveAll(tmpDir)

	provider := NewDockerSecretsProvider(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name       string
		secretKey  string
		secretVal  string
		writeFile  bool
		wantErr    bool
		wantCached bool
	}{
		{
			name:      "existing secret",
			secretKey: "TEST_SECRET",
			secretVal: "test_value",
			writeFile: true,
			wantErr:   false,
		},
		{
			name:      "secret with trailing newline",
			secretKey: "TEST_NEWLINE",
			secretVal: "test_value\n",
			writeFile: true,
			wantErr:   false,
		},
		{
			name:      "missing secret",
			secretKey: "MISSING_SECRET",
			writeFile: false,
			wantErr:   true,
		},
		{
			name:      "cached secret",
			secretKey: "CACHED_SECRET",
			secretVal: "cached_value",
			writeFile: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.writeFile {
				writeSecret(t, tmpDir, tt.secretKey, tt.secretVal)
			}

			// Execute
			got, err := provider.GetSecret(ctx, tt.secretKey)

			// Verify
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expected := tt.secretVal
				// Docker secrets provider trims whitespace
				if tt.secretVal == "test_value\n" {
					expected = "test_value"
				}
				if got != expected {
					t.Errorf("GetSecret() = %v, want %v", got, expected)
				}
			}

			// Test caching by reading again
			if !tt.wantErr {
				got2, err2 := provider.GetSecret(ctx, tt.secretKey)
				if err2 != nil {
					t.Errorf("Second GetSecret() error = %v", err2)
				}
				if got2 != got {
					t.Errorf("Cached GetSecret() = %v, want %v", got2, got)
				}
			}
		})
	}
}

func TestDockerSecretsProvider_GetSecretWithDefault(t *testing.T) {
	tmpDir := setupTestSecrets(t)
	defer os.RemoveAll(tmpDir)

	provider := NewDockerSecretsProvider(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name         string
		secretKey    string
		secretVal    string
		writeFile    bool
		defaultValue string
		want         string
	}{
		{
			name:         "existing secret",
			secretKey:    "TEST_SECRET",
			secretVal:    "actual_value",
			writeFile:    true,
			defaultValue: "default_value",
			want:         "actual_value",
		},
		{
			name:         "missing secret with default",
			secretKey:    "MISSING_SECRET",
			writeFile:    false,
			defaultValue: "default_value",
			want:         "default_value",
		},
		{
			name:         "missing secret without default",
			secretKey:    "MISSING_SECRET_2",
			writeFile:    false,
			defaultValue: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.writeFile {
				writeSecret(t, tmpDir, tt.secretKey, tt.secretVal)
			}

			// Execute
			got := provider.GetSecretWithDefault(ctx, tt.secretKey, tt.defaultValue)

			// Verify
			if got != tt.want {
				t.Errorf("GetSecretWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDockerSecretsProvider_MustGetSecret(t *testing.T) {
	tmpDir := setupTestSecrets(t)
	defer os.RemoveAll(tmpDir)

	provider := NewDockerSecretsProvider(tmpDir)
	ctx := context.Background()

	t.Run("existing secret", func(t *testing.T) {
		writeSecret(t, tmpDir, "TEST_MUST_SECRET", "test_value")

		got := provider.MustGetSecret(ctx, "TEST_MUST_SECRET")
		if got != "test_value" {
			t.Errorf("MustGetSecret() = %v, want %v", got, "test_value")
		}
	})

	t.Run("missing secret panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetSecret() did not panic for missing secret")
			}
		}()

		provider.MustGetSecret(ctx, "MISSING_MUST_SECRET")
	})
}

func TestDockerSecretsProvider_ClearCache(t *testing.T) {
	tmpDir := setupTestSecrets(t)
	defer os.RemoveAll(tmpDir)

	provider := NewDockerSecretsProvider(tmpDir)
	ctx := context.Background()

	// Write and cache a secret
	writeSecret(t, tmpDir, "CACHE_TEST", "original_value")
	value1, _ := provider.GetSecret(ctx, "CACHE_TEST")

	// Update the file
	writeSecret(t, tmpDir, "CACHE_TEST", "updated_value")

	// Without clearing cache, should get cached value
	value2, _ := provider.GetSecret(ctx, "CACHE_TEST")
	if value2 != value1 {
		t.Errorf("Expected cached value %v, got %v", value1, value2)
	}

	// Clear cache
	provider.ClearCache()

	// After clearing cache, should get new value
	value3, _ := provider.GetSecret(ctx, "CACHE_TEST")
	if value3 != "updated_value" {
		t.Errorf("After ClearCache() got %v, want %v", value3, "updated_value")
	}
}

func TestDockerSecretsProvider_RefreshSecret(t *testing.T) {
	tmpDir := setupTestSecrets(t)
	defer os.RemoveAll(tmpDir)

	provider := NewDockerSecretsProvider(tmpDir)
	ctx := context.Background()

	// Write and cache a secret
	writeSecret(t, tmpDir, "REFRESH_TEST", "original_value")
	value1, _ := provider.GetSecret(ctx, "REFRESH_TEST")
	if value1 != "original_value" {
		t.Fatalf("Expected original_value, got %v", value1)
	}

	// Update the file
	writeSecret(t, tmpDir, "REFRESH_TEST", "updated_value")

	// Refresh this specific secret
	provider.RefreshSecret("REFRESH_TEST")

	// Should get new value
	value2, _ := provider.GetSecret(ctx, "REFRESH_TEST")
	if value2 != "updated_value" {
		t.Errorf("After RefreshSecret() got %v, want %v", value2, "updated_value")
	}
}

func TestDockerSecretsProvider_ProviderName(t *testing.T) {
	provider := NewDockerSecretsProvider("/run/secrets")
	if got := provider.ProviderName(); got != "docker-secrets" {
		t.Errorf("ProviderName() = %v, want %v", got, "docker-secrets")
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   SecretConfig
		wantType string
		wantErr  bool
	}{
		{
			name: "env provider",
			config: SecretConfig{
				Provider: "env",
			},
			wantType: "env",
			wantErr:  false,
		},
		{
			name: "docker provider",
			config: SecretConfig{
				Provider: "docker",
			},
			wantType: "docker-secrets",
			wantErr:  false,
		},
		{
			name: "unknown provider",
			config: SecretConfig{
				Provider: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && provider.ProviderName() != tt.wantType {
				t.Errorf("NewProvider() type = %v, want %v", provider.ProviderName(), tt.wantType)
			}
		})
	}
}
