package storage

import (
	"context"
)

// Storage defines the interface for generic byte-level storage with atomic operations.
type Storage interface {
	// Read retrieves data for a key. Returns nil, 0 if key doesn't exist.
	Read(ctx context.Context, key string) (data []byte, generation int64, err error)

	// Write stores data for a key with optional generation precondition.
	// If expectedGeneration is 0, creates new object (fails if exists).
	// If expectedGeneration > 0, updates only if generation matches (fails if mismatch).
	// If expectedGeneration < 0, overwrites unconditionally.
	Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) error

	// Close releases storage resources.
	Close(ctx context.Context) error
}
