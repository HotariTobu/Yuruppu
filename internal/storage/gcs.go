package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

const httpPreconditionFailed = 412

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
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read %s: %w", key, err)
	}

	return data, reader.Attrs.Generation, nil
}

// Write stores data for a key with generation precondition.
// Returns ErrPreconditionFailed if generation doesn't match (412).
func (s *GCSStorage) Write(ctx context.Context, key string, data []byte, expectedGeneration int64) error {
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
		// expectedGeneration < 0: overwrite unconditionally
		writer = obj.NewWriter(ctx)
	}

	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		if isPreconditionFailed(err) {
			return fmt.Errorf("%w: %s", ErrPreconditionFailed, key)
		}
		return fmt.Errorf("failed to write %s: %w", key, err)
	}

	if err := writer.Close(); err != nil {
		if isPreconditionFailed(err) {
			return fmt.Errorf("%w: %s", ErrPreconditionFailed, key)
		}
		return fmt.Errorf("failed to write %s: %w", key, err)
	}

	return nil
}

// isPreconditionFailed checks if error is a GCS precondition failure (HTTP 412).
func isPreconditionFailed(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == httpPreconditionFailed
	}
	return false
}

// Close releases storage resources.
func (s *GCSStorage) Close(_ context.Context) error {
	return s.client.Close()
}
