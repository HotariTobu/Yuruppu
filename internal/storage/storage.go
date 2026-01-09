package storage

import (
	"context"
	"time"
)

// Storage defines the interface for generic byte-level storage with atomic operations.
type Storage interface {
	// Read retrieves data for a key. Returns nil, 0 if key doesn't exist.
	Read(ctx context.Context, key string) (data []byte, generation int64, err error)

	// Write stores data for a key with optional generation precondition.
	// If expectedGeneration is 0, creates new object (fails if exists).
	// If expectedGeneration > 0, updates only if generation matches (fails if mismatch).
	// Returns the new generation number of the written object.
	Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (newGeneration int64, err error)

	// GetSignedURL generates a signed URL for accessing the object.
	// method is the HTTP method (GET, PUT, etc.).
	// ttl is how long the URL should be valid.
	GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error)
}
