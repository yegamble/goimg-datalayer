// Package ipfs implements an IPFS storage provider using the Kubo HTTP API.
// This provider enables decentralized, content-addressed storage alongside
// traditional storage backends (local, S3).
package ipfs

import (
	"time"
)

// Default configuration values.
const (
	defaultAPIEndpoint     = "http://localhost:5001"
	defaultGatewayEndpoint = "http://localhost:8080"
	defaultTimeout         = 30 * time.Second
	defaultPublicGateway   = "https://ipfs.io"
)

// Config holds IPFS connection and behavior settings.
type Config struct {
	// APIEndpoint is the IPFS HTTP API address (e.g., "http://localhost:5001").
	// This is the Kubo node's API endpoint.
	APIEndpoint string

	// GatewayEndpoint is the HTTP gateway for public URL generation.
	// Example: "https://ipfs.io" or "http://localhost:8080"
	GatewayEndpoint string

	// Timeout is the maximum duration for API operations.
	// Defaults to 30 seconds if not specified.
	Timeout time.Duration

	// PinByDefault automatically pins uploaded content to the local node.
	// Pinned content is protected from garbage collection.
	PinByDefault bool
}

// DefaultConfig returns a configuration suitable for local development
// with a Kubo node running on standard ports.
func DefaultConfig() Config {
	return Config{
		APIEndpoint:     defaultAPIEndpoint,
		GatewayEndpoint: defaultGatewayEndpoint,
		Timeout:         defaultTimeout,
		PinByDefault:    true,
	}
}

// Validate checks the configuration for required values.
func (c *Config) Validate() error {
	if c.APIEndpoint == "" {
		return ErrConfigInvalid
	}
	return nil
}

// WithDefaults fills in default values for unset fields.
func (c *Config) WithDefaults() Config {
	cfg := *c
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.GatewayEndpoint == "" {
		cfg.GatewayEndpoint = defaultPublicGateway
	}
	return cfg
}
