package gallery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/domain/shared"
)

// validTestCID is a valid 46-character CIDv0 for testing.
const validTestCID = "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

// TestNewIPFSMetadata_Success tests successful creation.
func TestNewIPFSMetadata_Success(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	meta, err := NewIPFSMetadata(validTestCID, true, &now)
	require.NoError(t, err)

	assert.Equal(t, validTestCID, meta.CID())
	assert.True(t, meta.Pinned())
	assert.NotNil(t, meta.PinnedAt())
}

// TestNewIPFSMetadata_Unpinned tests creation without pinning.
func TestNewIPFSMetadata_Unpinned(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	meta, err := NewIPFSMetadata(validTestCID, false, &now)
	require.NoError(t, err)

	assert.Equal(t, validTestCID, meta.CID())
	assert.False(t, meta.Pinned())
	assert.Nil(t, meta.PinnedAt()) // Should be nil when not pinned
}

// TestNewIPFSMetadata_EmptyCID tests validation of empty CID.
func TestNewIPFSMetadata_EmptyCID(t *testing.T) {
	t.Parallel()

	meta, err := NewIPFSMetadata("", false, nil)
	assert.ErrorIs(t, err, shared.ErrInvalidInput)
	assert.True(t, meta.IsZero())
}

// TestNewIPFSMetadata_InvalidCID tests validation of invalid CIDs.
func TestNewIPFSMetadata_InvalidCID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cid  string
	}{
		{"too short CIDv0", "Qm123"},
		{"too short CIDv1", "bafy123"},
		{"wrong prefix", "Zm1234567890abcdefghijklmnopqrstuvw"},
		{"wrong length CIDv0", "QmTest12345"},
		{"unrecognized format", "xyz12345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta, err := NewIPFSMetadata(tt.cid, false, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, shared.ErrInvalidInput)
			assert.True(t, meta.IsZero())
		})
	}
}

// TestNewIPFSMetadata_ValidCIDFormats tests various valid CID formats.
func TestNewIPFSMetadata_ValidCIDFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cid  string
	}{
		{"CIDv0", "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"},
		{"CIDv1 bafy", "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"},
		{"CIDv1 bafk", "bafkreigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta, err := NewIPFSMetadata(tt.cid, true, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.cid, meta.CID())
		})
	}
}

// TestIPFSMetadata_IsZero tests the IsZero method.
func TestIPFSMetadata_IsZero(t *testing.T) {
	t.Parallel()

	var zero IPFSMetadata
	assert.True(t, zero.IsZero())

	nonZero, err := NewIPFSMetadata(validTestCID, false, nil)
	require.NoError(t, err)
	assert.False(t, nonZero.IsZero())
}

// TestIPFSMetadata_URI tests the URI method.
func TestIPFSMetadata_URI(t *testing.T) {
	t.Parallel()

	meta, err := NewIPFSMetadata(validTestCID, false, nil)
	require.NoError(t, err)

	expected := "ipfs://" + validTestCID
	assert.Equal(t, expected, meta.URI())
}

// TestIPFSMetadata_URI_Empty tests URI returns empty for zero value.
func TestIPFSMetadata_URI_Empty(t *testing.T) {
	t.Parallel()

	var zero IPFSMetadata
	assert.Empty(t, zero.URI())
}

// TestIPFSMetadata_GatewayURL tests the GatewayURL method.
func TestIPFSMetadata_GatewayURL(t *testing.T) {
	t.Parallel()

	meta, err := NewIPFSMetadata(validTestCID, false, nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		gateway  string
		expected string
	}{
		{"standard gateway", "https://ipfs.io", "https://ipfs.io/ipfs/" + validTestCID},
		{"gateway with trailing slash", "https://ipfs.io/", "https://ipfs.io/ipfs/" + validTestCID},
		{"custom gateway", "https://gateway.pinata.cloud", "https://gateway.pinata.cloud/ipfs/" + validTestCID},
		{"empty gateway", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, meta.GatewayURL(tt.gateway))
		})
	}
}

// TestIPFSMetadata_GatewayURL_Empty tests GatewayURL for zero value.
func TestIPFSMetadata_GatewayURL_Empty(t *testing.T) {
	t.Parallel()

	var zero IPFSMetadata
	assert.Empty(t, zero.GatewayURL("https://ipfs.io"))
}

// TestIPFSMetadata_WithPinned tests the WithPinned method.
func TestIPFSMetadata_WithPinned(t *testing.T) {
	t.Parallel()

	meta, err := NewIPFSMetadata(validTestCID, false, nil)
	require.NoError(t, err)
	assert.False(t, meta.Pinned())
	assert.Nil(t, meta.PinnedAt())

	// Pin the content
	pinned := meta.WithPinned(true)
	assert.True(t, pinned.Pinned())
	assert.NotNil(t, pinned.PinnedAt())

	// Original should be unchanged
	assert.False(t, meta.Pinned())

	// Unpin the content
	unpinned := pinned.WithPinned(false)
	assert.False(t, unpinned.Pinned())
	assert.Nil(t, unpinned.PinnedAt())
}

// TestIPFSMetadata_Equals tests the Equals method.
func TestIPFSMetadata_Equals(t *testing.T) {
	t.Parallel()

	meta1, err := NewIPFSMetadata(validTestCID, false, nil)
	require.NoError(t, err)

	meta2, err := NewIPFSMetadata(validTestCID, true, nil)
	require.NoError(t, err)

	// Same CID, different pin status should be equal
	assert.True(t, meta1.Equals(meta2))

	// Different CID should not be equal
	meta3, err := NewIPFSMetadata("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", false, nil)
	require.NoError(t, err)
	assert.False(t, meta1.Equals(meta3))
}

// TestIPFSMetadata_TrimsWhitespace tests that CID is trimmed.
func TestIPFSMetadata_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	meta, err := NewIPFSMetadata("  "+validTestCID+"  ", false, nil)
	require.NoError(t, err)
	assert.Equal(t, validTestCID, meta.CID())
}
