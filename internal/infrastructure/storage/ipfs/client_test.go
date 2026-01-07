package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_Success tests successful creation of IPFS client.
func TestNew_Success(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint:     "http://localhost:5001",
		GatewayEndpoint: "http://localhost:8080",
		Timeout:         30 * time.Second,
		PinByDefault:    true,
	}

	client, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5001", client.config.APIEndpoint)
	assert.Equal(t, "http://localhost:8080", client.config.GatewayEndpoint)
}

// TestNew_WithDefaults tests that defaults are applied.
func TestNew_WithDefaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint: "http://localhost:5001",
		// GatewayEndpoint and Timeout not set
	}

	client, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "https://ipfs.io", client.config.GatewayEndpoint)
	assert.Equal(t, 30*time.Second, client.config.Timeout)
}

// TestNew_ValidationError tests configuration validation.
func TestNew_ValidationError(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint: "", // Empty = invalid
	}

	client, err := New(cfg)
	assert.Nil(t, client)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigInvalid)
}

// TestDefaultConfig tests the default configuration values.
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	assert.Equal(t, "http://localhost:5001", cfg.APIEndpoint)
	assert.Equal(t, "http://localhost:8080", cfg.GatewayEndpoint)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.True(t, cfg.PinByDefault)
}

// validTestCID is a valid 46-character CIDv0 for testing.
const validTestCID = "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

// TestAdd_Success tests successful content upload.
func TestAdd_Success(t *testing.T) {
	t.Parallel()

	expectedCID := validTestCID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v0/add")
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		w.WriteHeader(http.StatusOK)
		resp := AddResult{
			Hash: expectedCID,
			Name: "data",
			Size: "123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	result, err := client.Add(ctx, bytes.NewReader([]byte("test data")))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedCID, result.Hash)
	assert.Equal(t, "data", result.Name)
	assert.Equal(t, "123", result.Size)
}

// TestAdd_PinByDefault tests that pin parameter is set correctly.
func TestAdd_PinByDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		pinByDefault bool
		expectPin    string
	}{
		{"pin enabled", true, "pin=true"},
		{"pin disabled", false, "pin=false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.RawQuery, tt.expectPin)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(AddResult{Hash: "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"})
			}))
			defer server.Close()

			cfg := Config{
				APIEndpoint:  server.URL,
				PinByDefault: tt.pinByDefault,
			}
			client, err := New(cfg)
			require.NoError(t, err)

			_, err = client.Add(context.Background(), bytes.NewReader([]byte("test")))
			require.NoError(t, err)
		})
	}
}

// TestAdd_ServerError tests handling of server errors during upload.
func TestAdd_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	result, err := client.Add(ctx, bytes.NewReader([]byte("test data")))
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUploadFailed)
}

// TestAddBytes_Success tests the convenience AddBytes method.
func TestAddBytes_Success(t *testing.T) {
	t.Parallel()

	expectedCID := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AddResult{Hash: expectedCID})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	result, err := client.AddBytes(ctx, []byte("test bytes"))
	require.NoError(t, err)
	assert.Equal(t, expectedCID, result.Hash)
}

// TestGet_Success tests successful content retrieval.
func TestGet_Success(t *testing.T) {
	t.Parallel()

	testContent := []byte("retrieved content")
	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v0/cat")
		assert.Contains(t, r.URL.RawQuery, cid)

		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	reader, err := client.Get(ctx, cid)
	require.NoError(t, err)
	defer reader.Close()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

// TestGet_NotFound tests handling of not found errors.
func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	reader, err := client.Get(ctx, cid)
	assert.Nil(t, reader)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestGet_InvalidCID tests CID validation in Get.
func TestGet_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	reader, err := client.Get(ctx, "invalid-cid")
	assert.Nil(t, reader)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestGetBytes_Success tests the GetBytes convenience method.
func TestGetBytes_Success(t *testing.T) {
	t.Parallel()

	testContent := []byte("byte content")
	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	data, err := client.GetBytes(ctx, cid)
	require.NoError(t, err)
	assert.Equal(t, testContent, data)
}

// TestPin_Success tests successful pinning.
func TestPin_Success(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/pin/add")
		assert.Contains(t, r.URL.RawQuery, cid)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Pins": ["` + cid + `"]}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.Pin(ctx, cid)
	require.NoError(t, err)
}

// TestPin_InvalidCID tests CID validation in Pin.
func TestPin_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	err := client.Pin(ctx, "invalid")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestPin_ServerError tests handling of server errors during pin.
func TestPin_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("pin error"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.Pin(ctx, "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPinFailed)
}

// TestUnpin_Success tests successful unpinning.
func TestUnpin_Success(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/pin/rm")
		assert.Contains(t, r.URL.RawQuery, cid)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Pins": ["` + cid + `"]}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.Unpin(ctx, cid)
	require.NoError(t, err)
}

// TestUnpin_InvalidCID tests CID validation in Unpin.
func TestUnpin_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	err := client.Unpin(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestUnpin_ServerError tests handling of server errors during unpin.
func TestUnpin_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unpin error"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.Unpin(ctx, "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnpinFailed)
}

// TestIsPinned_True tests IsPinned when content is pinned.
func TestIsPinned_True(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/pin/ls")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Keys": {"` + cid + `": {"Type": "recursive"}}}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	pinned, err := client.IsPinned(ctx, cid)
	require.NoError(t, err)
	assert.True(t, pinned)
}

// TestIsPinned_False tests IsPinned when content is not pinned.
func TestIsPinned_False(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Not pinned returns 500
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	pinned, err := client.IsPinned(ctx, cid)
	require.NoError(t, err)
	assert.False(t, pinned)
}

// TestIsPinned_InvalidCID tests CID validation in IsPinned.
func TestIsPinned_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	pinned, err := client.IsPinned(ctx, "bad")
	assert.False(t, pinned)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestDelete_Success tests that Delete calls Unpin.
func TestDelete_Success(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	unpinCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/v0/pin/rm") {
			unpinCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.Delete(ctx, cid)
	require.NoError(t, err)
	assert.True(t, unpinCalled, "Delete should call Unpin")
}

// TestExists_True tests Exists when content is available locally.
func TestExists_True(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/block/stat")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Key": "` + cid + `", "Size": 123}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	exists, err := client.Exists(ctx, cid)
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestExists_False tests Exists when content is not available.
func TestExists_False(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	exists, err := client.Exists(ctx, cid)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestExists_InvalidCID tests CID validation in Exists.
func TestExists_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	exists, err := client.Exists(ctx, "")
	assert.False(t, exists)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestURL_WithGateway tests URL generation with gateway configured.
func TestURL_WithGateway(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint:     "http://localhost:5001",
		GatewayEndpoint: "https://ipfs.io",
	}
	client, err := New(cfg)
	require.NoError(t, err)

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	url := client.URL(cid)

	assert.Equal(t, "https://ipfs.io/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", url)
}

// TestURL_WithoutGateway tests URL generation without gateway configured.
func TestURL_WithoutGateway(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint:     "http://localhost:5001",
		GatewayEndpoint: "",
	}
	cfg = cfg.WithDefaults()
	client, err := New(cfg)
	require.NoError(t, err)

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	url := client.URL(cid)

	// Should use default gateway
	assert.Equal(t, "https://ipfs.io/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", url)
}

// TestIPFSURI tests IPFS URI generation.
func TestIPFSURI(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	uri := client.IPFSURI(cid)
	assert.Equal(t, "ipfs://QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", uri)
}

// TestPresignedURL_NotSupported tests that presigned URLs are not supported.
func TestPresignedURL_NotSupported(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	url, err := client.PresignedURL(ctx, "test", time.Hour)
	assert.Empty(t, url)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

// TestStat_Success tests successful Stat operation.
func TestStat_Success(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/block/stat")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Key": "` + cid + `", "Size": 12345}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	info, err := client.Stat(ctx, cid)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, cid, info.Key)
	assert.Equal(t, int64(12345), info.Size)
	assert.Equal(t, "application/octet-stream", info.ContentType)
	assert.Equal(t, `"`+cid+`"`, info.ETag)
}

// TestStat_NotFound tests Stat with non-existent CID.
func TestStat_NotFound(t *testing.T) {
	t.Parallel()

	cid := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	info, err := client.Stat(ctx, cid)
	assert.Nil(t, info)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestStat_InvalidCID tests CID validation in Stat.
func TestStat_InvalidCID(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	ctx := context.Background()

	info, err := client.Stat(ctx, "short")
	assert.Nil(t, info)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCID)
}

// TestProvider tests the Provider method.
func TestProvider(t *testing.T) {
	t.Parallel()

	client := setupTestClient(t, "http://localhost:5001")
	assert.Equal(t, "ipfs", client.Provider())
}

// TestNodeID_Success tests successful NodeID retrieval.
func TestNodeID_Success(t *testing.T) {
	t.Parallel()

	expectedID := "12D3KooWTestPeerID1234567890"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v0/id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ID": "` + expectedID + `"}`))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	id, err := client.NodeID(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedID, id)
}

// TestNodeID_ServerError tests NodeID when node is unavailable.
func TestNodeID_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	id, err := client.NodeID(ctx)
	assert.Empty(t, id)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNodeUnavailable)
}

// TestPut_Success tests the Storage interface Put method.
func TestPut_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AddResult{Hash: "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	// Key is ignored for IPFS
	err := client.Put(ctx, "ignored-key", bytes.NewReader([]byte("data")), 4, PutOptions{})
	require.NoError(t, err)
}

// TestPutBytes_Success tests the Storage interface PutBytes method.
func TestPutBytes_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AddResult{Hash: "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	ctx := context.Background()

	err := client.PutBytes(ctx, "ignored-key", []byte("data"), PutOptions{})
	require.NoError(t, err)
}

// TestValidateCID tests CID validation for various formats.
func TestValidateCID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cid     string
		wantErr bool
	}{
		// Valid CIDv0 (Qm prefix, 46 characters)
		{"valid CIDv0", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", false},
		{"valid CIDv0 real", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG", false},

		// Valid CIDv1 (bafy prefix)
		{"valid CIDv1 bafy", "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", false},
		{"valid CIDv1 bafk", "bafkreigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", false},

		// Valid CIDv1 other bases (starts with b, longer than 10 chars)
		{"valid CIDv1 other base", "baegbeibz22e4qmqj7ivx4bk7e3xvqsq4ipq", false},

		// Invalid CIDs
		{"empty CID", "", true},
		{"too short CIDv0", "Qm123", true},
		{"too short CIDv1", "bafy123", true},
		{"wrong prefix", "Zm1234567890abcdefghijklmnopqrstuvw", true},
		{"wrong length CIDv0", "QmTest12345", true},
		{"unrecognized format", "xyz12345678901234567890", true},
		{"just b prefix too short", "b123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCID(tt.cid)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidCID)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestContextCancellation tests that operations respect context cancellation.
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Add(ctx, bytes.NewReader([]byte("test")))
	require.Error(t, err)
}

// TestIsRetryable tests the IsRetryable helper function.
func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"node unavailable", ErrNodeUnavailable, true},
		{"timeout", ErrTimeout, true},
		{"not found", ErrNotFound, false},
		{"invalid CID", ErrInvalidCID, false},
		{"upload failed", ErrUploadFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

// TestIsNotFound tests the IsNotFound helper function.
func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"not found", ErrNotFound, true},
		{"node unavailable", ErrNodeUnavailable, false},
		{"invalid CID", ErrInvalidCID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsNotFound(tt.err))
		})
	}
}

// TestConfigValidate tests configuration validation.
func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{APIEndpoint: "http://localhost:5001"},
			wantErr: false,
		},
		{
			name:    "empty endpoint",
			cfg:     Config{APIEndpoint: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrConfigInvalid)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfigWithDefaults tests default value application.
func TestConfigWithDefaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		APIEndpoint: "http://localhost:5001",
		// Other fields empty
	}

	result := cfg.WithDefaults()

	assert.Equal(t, "http://localhost:5001", result.APIEndpoint)
	assert.Equal(t, "https://ipfs.io", result.GatewayEndpoint)
	assert.Equal(t, 30*time.Second, result.Timeout)
}

// setupTestClient creates an IPFS client for testing with the given API endpoint.
func setupTestClient(t *testing.T, apiEndpoint string) *Client {
	t.Helper()

	cfg := Config{
		APIEndpoint:     apiEndpoint,
		GatewayEndpoint: "https://ipfs.io",
		Timeout:         5 * time.Second,
		PinByDefault:    true,
	}

	client, err := New(cfg)
	require.NoError(t, err)
	return client
}
