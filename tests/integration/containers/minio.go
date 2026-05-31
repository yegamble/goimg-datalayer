package containers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	MinIODefaultBucket    = "test-bucket"
	MinIODefaultAccessKey = "minioadmin"
	MinIODefaultSecretKey = "minioadmin"
)

type MinIOContainer struct {
	Container  testcontainers.Container
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
}

func NewMinIOContainer(ctx context.Context, t testing.TB) (*MinIOContainer, error) {
	if t != nil {
		t.Helper()
	}

	defer func() {
		if r := recover(); r != nil {
			if t != nil && isDockerUnavailablePanic(r) {
				skipDockerUnavailable(t, r)
				return
			}
			panic(r)
		}
	}()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:RELEASE.2024-10-13T13-34-11Z",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     MinIODefaultAccessKey,
			"MINIO_ROOT_PASSWORD": MinIODefaultSecretKey,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForLog("API:").WithStartupTimeout(60 * time.Second),
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
		return nil, fmt.Errorf("failed to start MinIO container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MinIO container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MinIO mapped port: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	if err := createMinioBucket(ctx, endpoint, MinIODefaultAccessKey, MinIODefaultSecretKey, MinIODefaultBucket); err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to create MinIO test bucket: %w", err)
	}

	return &MinIOContainer{
		Container:  container,
		Endpoint:   endpoint,
		AccessKey:  MinIODefaultAccessKey,
		SecretKey:  MinIODefaultSecretKey,
		BucketName: MinIODefaultBucket,
	}, nil
}

func createMinioBucket(ctx context.Context, endpoint, accessKey, secretKey, bucket string) error {
	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, secretKey, "",
		)),
	)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &awss3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("create bucket %q: %w", bucket, err)
	}

	return nil
}

func (c *MinIOContainer) Terminate(ctx context.Context) error {
	if c.Container != nil {
		if err := c.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("terminate MinIO container: %w", err)
		}
	}
	return nil
}
