package tasks

import "context"

// Storage is an interface for storing and retrieving image data.
// This interface is used by both image processing and scanning tasks.
type Storage interface {
	// Get retrieves data by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Put stores data with the given key.
	Put(ctx context.Context, key string, data []byte) error
}
