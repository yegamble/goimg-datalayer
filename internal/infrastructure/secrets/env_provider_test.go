package secrets

import (
	"context"
	"os"
	"testing"
)

func TestEnvProvider_GetSecret(t *testing.T) {
	provider := NewEnvProvider()
	ctx := context.Background()

	tests := []struct {
		name      string
		secretKey string
		setValue  string
		wantErr   bool
	}{
		{
			name:      "existing secret",
			secretKey: "TEST_SECRET",
			setValue:  "test_value",
			wantErr:   false,
		},
		{
			name:      "missing secret",
			secretKey: "MISSING_SECRET",
			setValue:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setValue != "" {
				if err := os.Setenv(tt.secretKey, tt.setValue); err != nil {
					t.Fatalf("failed to set env var: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.secretKey); err != nil {
						t.Logf("failed to unset env var: %v", err)
					}
				}()
			}

			// Execute
			got, err := provider.GetSecret(ctx, tt.secretKey)

			// Verify
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.setValue {
				t.Errorf("GetSecret() = %v, want %v", got, tt.setValue)
			}
		})
	}
}

func TestEnvProvider_GetSecretWithDefault(t *testing.T) {
	provider := NewEnvProvider()
	ctx := context.Background()

	tests := []struct {
		name         string
		secretKey    string
		setValue     string
		defaultValue string
		want         string
	}{
		{
			name:         "existing secret",
			secretKey:    "TEST_SECRET",
			setValue:     "actual_value",
			defaultValue: "default_value",
			want:         "actual_value",
		},
		{
			name:         "missing secret with default",
			secretKey:    "MISSING_SECRET",
			setValue:     "",
			defaultValue: "default_value",
			want:         "default_value",
		},
		{
			name:         "missing secret without default",
			secretKey:    "MISSING_SECRET",
			setValue:     "",
			defaultValue: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setValue != "" {
				if err := os.Setenv(tt.secretKey, tt.setValue); err != nil {
					t.Fatalf("failed to set env var: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.secretKey); err != nil {
						t.Logf("failed to unset env var: %v", err)
					}
				}()
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

func TestEnvProvider_MustGetSecret(t *testing.T) {
	provider := NewEnvProvider()
	ctx := context.Background()

	t.Run("existing secret", func(t *testing.T) {
		if err := os.Setenv("TEST_MUST_SECRET", "test_value"); err != nil {
			t.Fatalf("failed to set env var: %v", err)
		}
		defer func() {
			if err := os.Unsetenv("TEST_MUST_SECRET"); err != nil {
				t.Logf("failed to unset env var: %v", err)
			}
		}()

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

func TestEnvProvider_ValidateRequiredSecrets(t *testing.T) {
	provider := NewEnvProvider()
	ctx := context.Background()

	t.Run("all required secrets present", func(t *testing.T) {
		// Set all required secrets
		for _, name := range RequiredSecrets() {
			_ = os.Setenv(name, "test_value")
			defer func(n string) { _ = os.Unsetenv(n) }(name)
		}

		err := provider.ValidateRequiredSecrets(ctx)
		if err != nil {
			t.Errorf("ValidateRequiredSecrets() error = %v, want nil", err)
		}
	})

	t.Run("missing required secret", func(t *testing.T) {
		// Clear all secrets
		for _, name := range RequiredSecrets() {
			_ = os.Unsetenv(name)
		}

		err := provider.ValidateRequiredSecrets(ctx)
		if err == nil {
			t.Error("ValidateRequiredSecrets() error = nil, want error for missing secrets")
		}
	})
}

func TestEnvProvider_ProviderName(t *testing.T) {
	provider := NewEnvProvider()
	if got := provider.ProviderName(); got != "env" {
		t.Errorf("ProviderName() = %v, want %v", got, "env")
	}
}
