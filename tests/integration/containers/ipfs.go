package containers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/ipfs"
)

type IPFSContainer struct {
	Container   testcontainers.Container
	Client      *ipfs.Client
	APIEndpoint string
}

func NewIPFSContainer(ctx context.Context, t testing.TB) (*IPFSContainer, error) {
	if t != nil {
		t.Helper()
	}

	defer func() {
		if r := recover(); r != nil {
			if t != nil && isDockerUnavailablePanic(r) {
				skipDockerUnavailable(t, r)
				return
			}
			fmt.Printf("ipfs panic: %v\n", r) //nolint:forbidigo
		}
	}()

	req := testcontainers.ContainerRequest{
		Image:        "ipfs/kubo:v0.31.0",
		ExposedPorts: []string{"5001/tcp"},
		WaitingFor: wait.ForLog("API server listening on").
			WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if t != nil && isDockerUnavailable(err) {
			skipDockerUnavailable(t, err)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to start IPFS container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get IPFS container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5001/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get IPFS mapped port: %w", err)
	}

	apiEndpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	client, err := ipfs.New(ipfs.Config{
		APIEndpoint:     apiEndpoint,
		GatewayEndpoint: "",
		Timeout:         30 * time.Second,
		PinByDefault:    true,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to create IPFS client: %w", err)
	}

	if err := verifyIPFSReady(ctx, client); err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("IPFS container did not become ready: %w", err)
	}

	return &IPFSContainer{
		Container:   container,
		Client:      client,
		APIEndpoint: apiEndpoint,
	}, nil
}

func verifyIPFSReady(ctx context.Context, client *ipfs.Client) error {
	const maxRetries = 10
	const retryInterval = 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		nodeID, err := client.NodeID(ctx)
		if err == nil && nodeID != "" {
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(retryInterval)
		}
	}
	return fmt.Errorf("NodeID did not respond after %d retries", maxRetries)
}

func (c *IPFSContainer) Terminate(ctx context.Context) error {
	if c.Container != nil {
		if err := c.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("terminate IPFS container: %w", err)
		}
	}
	return nil
}
