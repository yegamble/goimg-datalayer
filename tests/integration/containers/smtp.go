package containers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type SMTPContainer struct {
	Container   testcontainers.Container
	SMTPHost    string
	SMTPPort    string
	APIEndpoint string
}

func NewSMTPContainer(ctx context.Context, t testing.TB) (*SMTPContainer, error) {
	if t != nil {
		t.Helper()
	}

	defer func() {
		if r := recover(); r != nil {
			if t != nil && isDockerUnavailablePanic(r) {
				skipDockerUnavailable(t, r)
				return
			}
			fmt.Printf("smtp panic: %v\n", r) //nolint:forbidigo
		}
	}()

	req := testcontainers.ContainerRequest{
		Image:        "axllent/mailpit:latest",
		ExposedPorts: []string{"1025/tcp", "8025/tcp"},
		WaitingFor: wait.ForLog("accessible via").
			WithStartupTimeout(30 * time.Second),
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
		return nil, fmt.Errorf("failed to start Mailpit container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Mailpit container host: %w", err)
	}

	smtpPort, err := container.MappedPort(ctx, "1025/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Mailpit SMTP port: %w", err)
	}

	apiPort, err := container.MappedPort(ctx, "8025/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Mailpit API port: %w", err)
	}

	return &SMTPContainer{
		Container:   container,
		SMTPHost:    host,
		SMTPPort:    smtpPort.Port(),
		APIEndpoint: fmt.Sprintf("http://%s:%s", host, apiPort.Port()),
	}, nil
}

func (c *SMTPContainer) Terminate(ctx context.Context) error {
	if c.Container != nil {
		if err := c.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("terminate Mailpit container: %w", err)
		}
	}
	return nil
}
