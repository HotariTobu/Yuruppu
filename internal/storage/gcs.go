package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

// GCSStorage implements Storage interface using Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket *storage.BucketHandle
}

// NewGCSStorage creates a new GCS storage backend.
func NewGCSStorage(ctx context.Context, bucketName string) (*GCSStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	return &GCSStorage{
		client: client,
		bucket: client.Bucket(bucketName),
	}, nil
}

// Read retrieves data for a key. Returns nil, 0 if key doesn't exist.
func (s *GCSStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	obj := s.bucket.Object(key)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to read %s: %w", key, err)
	}

	data, readErr := io.ReadAll(reader)
	generation := reader.Attrs.Generation
	closeErr := reader.Close()

	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, 0, fmt.Errorf("failed to read %s: %w", key, err)
	}

	return data, generation, nil
}

// Write stores data for a key with generation precondition.
// Returns ErrPreconditionFailed if generation doesn't match (412).
func (s *GCSStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) error {
	obj := s.bucket.Object(key)

	var writer *storage.Writer
	switch {
	case expectedGeneration == 0:
		// Create new object, fail if exists
		writer = obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	case expectedGeneration > 0:
		// Update only if generation matches
		writer = obj.If(storage.Conditions{GenerationMatch: expectedGeneration}).NewWriter(ctx)
	default:
		return fmt.Errorf("invalid expectedGeneration: %d (must be >= 0)", expectedGeneration)
	}

	if writer == nil {
		return fmt.Errorf("failed to create writer for %s", key)
	}

	writer.ContentType = mimetype

	_, writeErr := writer.Write(data)
	closeErr := writer.Close()

	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("failed to write %s: %w", key, err)
	}

	return nil
}

// GetSignedURL generates a signed URL for accessing the object.
func (s *GCSStorage) GetSignedURL(_ context.Context, key, method string, ttl time.Duration) (string, error) {
	url, err := s.bucket.SignedURL(key, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(ttl),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for %s: %w", key, err)
	}
	return url, nil
}

// Close releases storage resources.
func (s *GCSStorage) Close(_ context.Context) error {
	return s.client.Close()
}
