package email

import (
	"fmt"
	"time"
)

// Config holds email/SMTP configuration.
// Configuration can be loaded from environment variables or config files.
type Config struct {
	// SMTP server configuration
	Host     string `env:"SMTP_HOST" envDefault:"localhost"`
	Port     int    `env:"SMTP_PORT" envDefault:"587"`
	Username string `env:"SMTP_USERNAME"`
	Password string `env:"SMTP_PASSWORD"`

	// Email sender configuration
	FromAddress string `env:"SMTP_FROM_ADDRESS"`
	FromName    string `env:"SMTP_FROM_NAME" envDefault:"goimg Gallery"`

	// TLS configuration
	UseTLS bool `env:"SMTP_USE_TLS" envDefault:"true"`

	// Operation timeouts
	Timeout time.Duration `env:"SMTP_TIMEOUT" envDefault:"30s"`

	// Rate limiting (emails per hour)
	RateLimit int `env:"SMTP_RATE_LIMIT" envDefault:"100"`

	// Feature flags
	Enabled bool `env:"SMTP_ENABLED" envDefault:"false"` // Disabled by default (opt-in)
}

// Validate ensures the configuration is valid and complete.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if SMTP is disabled
	}

	if c.Host == "" {
		return fmt.Errorf("smtp host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("smtp port must be between 1 and 65535")
	}

	if c.FromAddress == "" {
		return fmt.Errorf("smtp from address is required")
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("smtp rate limit must be positive")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("smtp timeout must be positive")
	}

	return nil
}

// IsEnabled returns true if email sending is enabled.
func (c Config) IsEnabled() bool {
	return c.Enabled
}

// DefaultConfig returns a default email configuration suitable for development.
// In production, load configuration from environment variables.
func DefaultConfig() Config {
	return Config{
		Host:        "localhost",
		Port:        587,
		FromAddress: "noreply@goimg.local",
		FromName:    "goimg Gallery",
		UseTLS:      true,
		Timeout:     30 * time.Second,
		RateLimit:   100,
		Enabled:     false, // Disabled by default
	}
}
