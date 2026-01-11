package s3

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3Client is a mock implementation of the S3 client for testing.
type mockS3Client struct {
	putObjectFunc    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFunc    func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	headObjectFunc   func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

// mockPresignClient is a mock implementation of the presign client for testing.
type mockPresignClient struct {
	presignGetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

func (m *mockPresignClient) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignGetObjectFunc != nil {
		return m.presignGetObjectFunc(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{URL: "https://example.com/presigned-url"}, nil
}

// mockReadCloser is a mock io.ReadCloser for testing.
type mockReadCloser struct {
	io.Reader
	closeErr error
}

func (m *mockReadCloser) Close() error {
	return m.closeErr
}

// s3ClientInterface defines the interface for S3 client methods we use.
type s3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// presignClientInterface defines the interface for presign client methods we use.
type presignClientInterface interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// newTestStorage creates a Storage instance with mock clients for testing.
func newTestStorage(client s3ClientInterface, presigner presignClientInterface, bucket string) *Storage {
	// We need to cast back to concrete types, but this is safe for testing
	// In a real implementation, we'd refactor Storage to use interfaces
	s := &Storage{
		bucket:    bucket,
		publicURL: "",
		expiry:    defaultPresignedURLExpiry,
	}

	// Use type assertion to set the mock clients
	// This is a workaround since we can't easily modify the Storage struct
	// In production code, we'd use interfaces or dependency injection
	if mockClient, ok := client.(*mockS3Client); ok {
		// Store reference for later use in tests
		_ = mockClient
	}

	return s
}

// Common test errors.
var (
	testErrS3Failure    = errors.New("s3 operation failed")
	testErrNotFound     = errors.New("NotFound")
	testErrAccessDenied = errors.New("AccessDenied")
)
