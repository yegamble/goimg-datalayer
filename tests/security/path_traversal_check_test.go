package security_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage"
	"github.com/yegamble/goimg-datalayer/internal/infrastructure/storage/local"
)

// TestStorage_PathTraversal_ZipSlipProtection verifies that the storage implementation
// protects against Partial Directory Match (Zip Slip) vulnerabilities.
func TestStorage_PathTraversal_ZipSlipProtection(t *testing.T) {
	t.Parallel()

	// Setup a temporary directory for storage
	baseDir, err := os.MkdirTemp("", "storage_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(baseDir)

	// Create a sibling directory that should NOT be accessible
	siblingDir := baseDir + "_secret"
	err = os.Mkdir(siblingDir, 0755)
	if err == nil {
		defer os.RemoveAll(siblingDir)
	}

	// Initialize storage
	cfg := local.Config{
		BasePath: baseDir,
		BaseURL:  "http://localhost:8080",
	}
	store, err := local.New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		wantError error
	}{
		{
			name:      "simple traversal",
			key:       "../secret.txt",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "traversal to sibling",
			key:       "../" + filepath.Base(siblingDir) + "/secret.txt",
			wantError: storage.ErrPathTraversal,
		},
		{
			name:      "absolute path",
			key:       "/etc/passwd",
			wantError: storage.ErrPathTraversal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := store.Exists(ctx, tt.key)
			if tt.wantError != nil {
				// local package defines its own errPathTraversal, so check message
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "path traversal detected")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStorage_PrefixEnforcement ensures that we are strictly enforcing the path prefix.
func TestStorage_PrefixEnforcement(t *testing.T) {
	t.Parallel()

	baseDir, err := os.MkdirTemp("", "storage_prefix_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(baseDir)

	cfg := local.Config{
		BasePath: baseDir,
		BaseURL:  "http://localhost:8080",
	}
	store, err := local.New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Valid key
	validKey := "images/test.jpg"
	err = store.PutBytes(ctx, validKey, []byte("content"), local.PutOptions{})
	assert.NoError(t, err)

	exists, err := store.Exists(ctx, validKey)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify it ended up in the right place
	expectedPath := filepath.Join(baseDir, "images", "test.jpg")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
}
