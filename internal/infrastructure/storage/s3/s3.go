// Package s3 implements an S3-compatible storage provider.
// It supports AWS S3, DigitalOcean Spaces, Backblaze B2, and MinIO.
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Common storage errors (duplicated to avoid import cycle).
var (
	ErrNotFound      = errors.New("storage: object not found")
	ErrAccessDenied  = errors.New("storage: access denied")
	errInvalidKey    = errors.New("storage: invalid key")
	errPathTraversal = errors.New("storage: path traversal detected")
)

const (
	// defaultPresignedURLExpiry is the default duration for presigned URLs.
	defaultPresignedURLExpiry = 15 * time.Minute
)

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// PutOptions configures storage upload behavior.
type PutOptions struct {
	ContentType  string
	CacheControl string
	Metadata     map[string]string
}

// Storage implements the storage interface using AWS S3 or S3-compatible services.
type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	publicURL string
	expiry    time.Duration
}

// Config configures the S3 storage client.
type Config struct {
	// Endpoint for S3-compatible services (leave empty for AWS S3).
	Endpoint string

	// Region for bucket location (required for AWS, optional for others).
	Region string

	// Bucket name (must exist before application starts).
	Bucket string

	// AccessKeyID for authentication.
	AccessKeyID string

	// SecretAccessKey for authentication.
	SecretAccessKey string

	// ForcePathStyle uses path-style URLs (required for MinIO).
	ForcePathStyle bool

	// PublicURL for CDN or custom domain (optional).
	PublicURL string

	// PresignedURLExpiry sets default expiry for presigned URLs.
	PresignedURLExpiry time.Duration
}

// New creates a new S3 storage client.
func New(ctx context.Context, cfg Config) (*Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3: bucket name required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.PresignedURLExpiry == 0 {
		cfg.PresignedURLExpiry = defaultPresignedURLExpiry
	}

	// Build AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: load config: %w", err)
	}

	// Create S3 client with custom options
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &Storage{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
		expiry:    cfg.PresignedURLExpiry,
	}, nil
}

// Put uploads data to S3 using streaming.
func (s *Storage) Put(ctx context.Context, key string, data io.Reader, size int64, opts PutOptions) error {
	if err := validateKey(key); err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          data,
		ContentLength: aws.Int64(size),
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if opts.CacheControl != "" {
		input.CacheControl = aws.String(opts.CacheControl)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}

	return nil
}

// PutBytes uploads small data to S3.
func (s *Storage) PutBytes(ctx context.Context, key string, data []byte, opts PutOptions) error {
	return s.Put(ctx, key, bytes.NewReader(data), int64(len(data)), opts)
}

// Get retrieves data from S3 as a streaming reader.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrNotFound
		}
		if isAccessDeniedError(err) {
			return nil, ErrAccessDenied
		}
		return nil, fmt.Errorf("s3 get: %w", err)
	}

	return result.Body, nil
}

// GetBytes retrieves small data fully into memory.
func (s *Storage) GetBytes(ctx context.Context, key string) ([]byte, error) {
	reader, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			// Log close error but return read result
			_ = cerr
		}
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("s3 read: %w", err)
	}

	return data, nil
}

// Delete removes an object from S3.
func (s *Storage) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}

	return nil
}

// Exists checks if an object exists in S3.
func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("s3 exists: %w", err)
	}

	return true, nil
}

// URL returns the public URL for a key.
func (s *Storage) URL(key string) string {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.publicURL, "/"), key)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}

// PresignedURL generates a temporary signed URL.
func (s *Storage) PresignedURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}

	if duration == 0 {
		duration = s.expiry
	}

	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = duration
	})
	if err != nil {
		return "", fmt.Errorf("s3 presign: %w", err)
	}

	return request.URL, nil
}

// Stat returns metadata about an object.
func (s *Storage) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("s3 stat: %w", err)
	}

	info := &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		ContentType:  aws.ToString(result.ContentType),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
	}

	return info, nil
}

// Provider returns the provider type name.
func (s *Storage) Provider() string {
	return "s3"
}

// validateKey checks if a storage key is safe.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty key", errInvalidKey)
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: contains '..'", errPathTraversal)
	}
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return fmt.Errorf("%w: cannot be absolute path", errPathTraversal)
	}
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("%w: contains null byte", errInvalidKey)
	}
	return nil
}

// isNotFoundError checks if the error indicates the object was not found.
func isNotFoundError(err error) bool {
	var notFound *types.NoSuchKey
	var notFoundErr *types.NotFound
	return errors.As(err, &notFound) || errors.As(err, &notFoundErr) ||
		strings.Contains(err.Error(), "NotFound") ||
		strings.Contains(err.Error(), "NoSuchKey")
}

// isAccessDeniedError checks if the error indicates access was denied.
func isAccessDeniedError(err error) bool {
	return strings.Contains(err.Error(), "AccessDenied")
}
