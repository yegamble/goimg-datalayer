//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

func TestIPFSContainer_NodeID(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	nodeID, err := c.Client.NodeID(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeID, "node ID should be a non-empty peer ID")
	t.Logf("IPFS node ID: %s", nodeID)
}

func TestIPFSClient_AddBytes(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	data := []byte("hello ipfs integration test")

	result, err := c.Client.AddBytes(ctx, data)
	require.NoError(t, err)
	require.NotEmpty(t, result.Hash, "CID should not be empty")
	t.Logf("Added content with CID: %s", result.Hash)

	retrieved, err := c.Client.GetBytes(ctx, result.Hash)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved, "retrieved content must match original")
}

func TestIPFSClient_PinLifecycle(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	data := []byte("pin lifecycle test data")

	result, err := c.Client.AddBytes(ctx, data)
	require.NoError(t, err)
	cid := result.Hash

	pinned, err := c.Client.IsPinned(ctx, cid)
	require.NoError(t, err)
	assert.True(t, pinned, "content should be pinned after add with PinByDefault=true")

	err = c.Client.Unpin(ctx, cid)
	require.NoError(t, err)

	pinned, err = c.Client.IsPinned(ctx, cid)
	require.NoError(t, err)
	assert.False(t, pinned, "content should not be pinned after unpin")

	err = c.Client.Pin(ctx, cid)
	require.NoError(t, err)

	pinned, err = c.Client.IsPinned(ctx, cid)
	require.NoError(t, err)
	assert.True(t, pinned, "content should be pinned after explicit pin")
}

func TestIPFSClient_Stat(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	data := []byte("stat test content with known length")

	result, err := c.Client.AddBytes(ctx, data)
	require.NoError(t, err)

	info, err := c.Client.Stat(ctx, result.Hash)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, result.Hash, info.Key)
	assert.Greater(t, info.Size, int64(0), "size should be positive")
	t.Logf("Stat: key=%s size=%d", info.Key, info.Size)
}

func TestIPFSClient_GetNonExistent(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	fakeCID := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"

	_, err = c.Client.GetBytes(ctx, fakeCID)
	assert.Error(t, err, "getting a non-existent CID should return an error")
}

func TestIPFSClient_ContentAddressingDeterminism(t *testing.T) {
	ctx := context.Background()

	c, err := containers.NewIPFSContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Cleanup(func() {
		_ = c.Terminate(ctx)
	})

	data := []byte("deterministic content addressing test")

	result1, err := c.Client.AddBytes(ctx, data)
	require.NoError(t, err)

	result2, err := c.Client.AddBytes(ctx, data)
	require.NoError(t, err)

	assert.Equal(t, result1.Hash, result2.Hash,
		"same content should produce the same CID (content-addressed)")
}
