package local

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurity_ResolvePath_DefenseInDepth demonstrates that resolvePath prevents
// path traversal even if the input key bypasses validateKey (e.g. via symbolic links
// or other filesystem tricks, though here we simulate it by calling resolvePath directly).
func TestSecurity_ResolvePath_DefenseInDepth(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	storageBase := filepath.Join(tempDir, "storage")

	cfg := Config{
		BasePath: storageBase,
		BaseURL:  "http://localhost:8080/uploads",
	}

	storage, err := New(cfg)
	require.NoError(t, err)

	// Attack scenario: Try to access a file in a sibling directory
	// e.g. base is /tmp/.../storage, we want /tmp/.../storage-hack

	// Construct a key that traverses up
	hackKey := "../storage-hack/pwned.txt"

	// resolvePath should return an error
	resolvedPath, err := storage.resolvePath(hackKey)

	require.Error(t, err)
	assert.ErrorIs(t, err, errPathTraversal)
	assert.Empty(t, resolvedPath)

	// Also try a case that would look like a valid prefix but isn't
	// e.g. /tmp/storage-hack is prefixed with /tmp/storage if no separator check
	// But our resolvePath logic handles this.

	// Note: We can't easily construct a key that results in /tmp/storage-hack
	// without using .. or absolute paths, both of which are cleaned/joined.
	// But if we could, resolvePath protects against it.

	hackKey2 := "../storage-hack"
	resolvedPath2, err := storage.resolvePath(hackKey2)
	require.Error(t, err)
	assert.ErrorIs(t, err, errPathTraversal)
	assert.Empty(t, resolvedPath2)
}
