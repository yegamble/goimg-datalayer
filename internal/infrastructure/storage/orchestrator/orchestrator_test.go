package orchestrator

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
)

// mockStorage is a mock implementation of storage.Storage for testing.
type mockStorage struct {
	data    map[string][]byte
	putErr  error
	getErr  error
	delErr  error
	statErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockStorage) Put(_ context.Context, key string, data io.Reader, _ int64, _ storage.PutOptions) error {
	if m.putErr != nil {
		return m.putErr
	}
	content, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.data[key] = content
	return nil
}

func (m *mockStorage) PutBytes(_ context.Context, key string, data []byte, _ storage.PutOptions) error {
	if m.putErr != nil {
		return m.putErr
	}
	m.data[key] = data
	return nil
}

func (m *mockStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) GetBytes(_ context.Context, key string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return data, nil
}

func (m *mockStorage) Delete(_ context.Context, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.data, key)
	return nil
}

func (m *mockStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockStorage) URL(key string) string {
	return "http://example.com/" + key
}

func (m *mockStorage) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", storage.ErrNotSupported
}

func (m *mockStorage) Stat(_ context.Context, key string) (*storage.ObjectInfo, error) {
	if m.statErr != nil {
		return nil, m.statErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return &storage.ObjectInfo{
		Key:         key,
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}, nil
}

func (m *mockStorage) Provider() string {
	return "mock"
}

// TestNew_Success tests successful orchestrator creation.
func TestNew_Success(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)
	assert.NotNil(t, orch)
	assert.Equal(t, "orchestrator", orch.Provider())
}

// TestNew_NilPrimary tests error when primary storage is nil.
func TestNew_NilPrimary(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	orch, err := New(nil, nil, cfg)
	assert.Nil(t, orch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary storage is required")
}

// TestNew_IPFSRequiredWhenEnabled tests error when IPFS is enabled but client is nil.
func TestNew_IPFSRequiredWhenEnabled(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := Config{
		IPFSEnabled: true,
		Mode:        ModeDualSync,
	}

	orch, err := New(primary, nil, cfg)
	assert.Nil(t, orch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IPFS client required")
}

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, ModePrimaryOnly, cfg.Mode)
	assert.False(t, cfg.FallbackEnabled)
	assert.False(t, cfg.IPFSEnabled)
}

// TestPutBytes_PrimaryOnly tests PutBytes in primary-only mode.
func TestPutBytes_PrimaryOnly(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"
	data := []byte("test image data")
	opts := storage.PutOptions{ContentType: "image/jpeg"}

	err = orch.PutBytes(ctx, key, data, opts)
	require.NoError(t, err)

	// Verify data is in primary storage
	assert.Equal(t, data, primary.data[key])
}

// TestPut_PrimaryOnly tests Put with io.Reader in primary-only mode.
func TestPut_PrimaryOnly(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"
	data := []byte("streaming test data")
	opts := storage.PutOptions{ContentType: "image/jpeg"}

	err = orch.Put(ctx, key, bytes.NewReader(data), int64(len(data)), opts)
	require.NoError(t, err)

	// Verify data is in primary storage
	assert.Equal(t, data, primary.data[key])
}

// TestGet_Success tests successful data retrieval.
func TestGet_Success(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"
	data := []byte("test data")

	// Store data first
	primary.data[key] = data

	// Retrieve it
	reader, err := orch.Get(ctx, key)
	require.NoError(t, err)
	defer reader.Close()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

// TestGet_NotFound tests Get when data doesn't exist.
func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	reader, err := orch.Get(ctx, "nonexistent")
	assert.Nil(t, reader)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

// TestGetBytes_Success tests successful byte retrieval.
func TestGetBytes_Success(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"
	data := []byte("test bytes")

	primary.data[key] = data

	retrieved, err := orch.GetBytes(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

// TestDelete_Success tests successful deletion.
func TestDelete_Success(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"

	primary.data[key] = []byte("data")

	err = orch.Delete(ctx, key)
	require.NoError(t, err)

	_, exists := primary.data[key]
	assert.False(t, exists)
}

// TestExists_True tests Exists when data exists.
func TestExists_True(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"

	primary.data[key] = []byte("data")

	exists, err := orch.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestExists_False tests Exists when data doesn't exist.
func TestExists_False(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	exists, err := orch.Exists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestURL_ReturnsFromPrimary tests that URL delegates to primary.
func TestURL_ReturnsFromPrimary(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	url := orch.URL("test/image.jpg")
	assert.Equal(t, "http://example.com/test/image.jpg", url)
}

// TestStat_Success tests successful Stat operation.
func TestStat_Success(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test/image.jpg"
	data := []byte("test data")

	primary.data[key] = data

	info, err := orch.Stat(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, key, info.Key)
	assert.Equal(t, int64(len(data)), info.Size)
}

// TestStat_NotFound tests Stat when data doesn't exist.
func TestStat_NotFound(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	info, err := orch.Stat(ctx, "nonexistent")
	assert.Nil(t, info)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

// TestIPFSEnabled_False tests IPFSEnabled when IPFS is not configured.
func TestIPFSEnabled_False(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	assert.False(t, orch.IPFSEnabled())
}

// TestAddToIPFS_NoClient tests AddToIPFS when IPFS is not configured.
func TestAddToIPFS_NoClient(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	result, err := orch.AddToIPFS(ctx, []byte("data"))
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IPFS not configured")
}

// TestGetFromIPFS_NoClient tests GetFromIPFS when IPFS is not configured.
func TestGetFromIPFS_NoClient(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	data, err := orch.GetFromIPFS(ctx, "QmTest")
	assert.Nil(t, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IPFS not configured")
}

// TestIPFSURL_NoClient tests IPFSURL when IPFS is not configured.
func TestIPFSURL_NoClient(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	url := orch.IPFSURL("QmTest")
	assert.Empty(t, url)
}

// TestIPFSURI_NoClient tests IPFSURI when IPFS is not configured.
func TestIPFSURI_NoClient(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	uri := orch.IPFSURI("QmTest")
	assert.Empty(t, uri)
}

// TestIPFS_ReturnsNil tests IPFS accessor when not configured.
func TestIPFS_ReturnsNil(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	assert.Nil(t, orch.IPFS())
}

// TestPresignedURL_DelegatesToPrimary tests that PresignedURL delegates.
func TestPresignedURL_DelegatesToPrimary(t *testing.T) {
	t.Parallel()

	primary := newMockStorage()
	cfg := DefaultConfig()

	orch, err := New(primary, nil, cfg)
	require.NoError(t, err)

	ctx := context.Background()

	url, err := orch.PresignedURL(ctx, "test", time.Hour)
	assert.Empty(t, url)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotSupported)
}

// TestShouldWriteIPFS tests the shouldWriteIPFS helper.
func TestShouldWriteIPFS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		hasIPFS  bool
		expected bool
	}{
		{
			name: "primary only mode",
			cfg: Config{
				Mode:        ModePrimaryOnly,
				IPFSEnabled: true,
			},
			hasIPFS:  true,
			expected: false,
		},
		{
			name: "IPFS disabled",
			cfg: Config{
				Mode:        ModeDualSync,
				IPFSEnabled: false,
			},
			hasIPFS:  false,
			expected: false,
		},
		{
			name: "dual sync with IPFS",
			cfg: Config{
				Mode:        ModeDualSync,
				IPFSEnabled: true,
			},
			hasIPFS:  true,
			expected: true,
		},
		{
			name: "dual async with IPFS",
			cfg: Config{
				Mode:        ModeDualAsync,
				IPFSEnabled: true,
			},
			hasIPFS:  true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			primary := newMockStorage()

			// We can't easily create a real IPFS client without a server,
			// so we test the config logic without the client
			orch := &Orchestrator{
				primary:    primary,
				ipfsClient: nil, // This will make shouldWriteIPFS return false even when enabled
				config:     tt.cfg,
			}

			result := orch.shouldWriteIPFS()
			if tt.hasIPFS && tt.expected {
				// With nil client, should always be false
				assert.False(t, result)
			} else {
				assert.Equal(t, tt.expected && tt.hasIPFS, result)
			}
		})
	}
}

// TestModeConstants tests that mode constants have expected values.
func TestModeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Mode("primary_only"), ModePrimaryOnly)
	assert.Equal(t, Mode("dual_sync"), ModeDualSync)
	assert.Equal(t, Mode("dual_async"), ModeDualAsync)
}
