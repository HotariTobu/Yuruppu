package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

const (
	maxRetries             = 3
	retryDelayMs           = 100
	httpPreconditionFailed = 412
)

// ObjectHandle abstracts GCS object operations for testing.
type ObjectHandle interface {
	NewReader(ctx context.Context) (io.ReadCloser, error)
	NewWriter(ctx context.Context) io.WriteCloser
	Attrs(ctx context.Context) (*storage.ObjectAttrs, error)
	Generation(gen int64) ObjectHandle
	If(conds storage.Conditions) ObjectHandle
}

// BucketHandle abstracts GCS bucket operations for testing.
type BucketHandle interface {
	Object(name string) ObjectHandle
}

// gcsObjectHandle wraps *storage.ObjectHandle to implement ObjectHandle.
type gcsObjectHandle struct {
	handle *storage.ObjectHandle
}

func (h *gcsObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return h.handle.NewReader(ctx)
}

func (h *gcsObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	return h.handle.NewWriter(ctx)
}

func (h *gcsObjectHandle) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	return h.handle.Attrs(ctx)
}

func (h *gcsObjectHandle) Generation(gen int64) ObjectHandle {
	return &gcsObjectHandle{handle: h.handle.Generation(gen)}
}

func (h *gcsObjectHandle) If(conds storage.Conditions) ObjectHandle {
	return &gcsObjectHandle{handle: h.handle.If(conds)}
}

// gcsBucketHandle wraps *storage.BucketHandle to implement BucketHandle.
type gcsBucketHandle struct {
	handle *storage.BucketHandle
}

func (h *gcsBucketHandle) Object(name string) ObjectHandle {
	return &gcsObjectHandle{handle: h.handle.Object(name)}
}

// GCSStorage implements Storage interface using Google Cloud Storage.
type GCSStorage struct {
	bucket BucketHandle
}

// NewGCSStorage creates a new GCS storage backend.
func NewGCSStorage(client *storage.Client, bucketName string) *GCSStorage {
	return &GCSStorage{
		bucket: &gcsBucketHandle{handle: client.Bucket(bucketName)},
	}
}

// NewGCSStorageWithBucket creates a GCS storage backend with a custom bucket handle.
// This is primarily used for testing with mock bucket implementations.
func NewGCSStorageWithBucket(bucket BucketHandle) *GCSStorage {
	return &GCSStorage{
		bucket: bucket,
	}
}

// Read retrieves data for a key. Returns nil, 0 if key doesn't exist.
func (s *GCSStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	obj := s.bucket.Object(key)

	// Get attributes first to retrieve generation
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to get attrs for %s: %w", key, err)
	}

	generation := attrs.Generation

	reader, err := obj.Generation(generation).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to read %s: %w", key, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read %s: %w", key, err)
	}

	return data, generation, nil
}

// Write stores data for a key with optional generation precondition.
// Retries on precondition failure with exponential backoff.
func (s *GCSStorage) Write(ctx context.Context, key string, data []byte, expectedGeneration int64) error {
	for attempt := range maxRetries {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms
			delay := time.Duration(retryDelayMs*(1<<(attempt-1))) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := s.doWrite(ctx, key, data, expectedGeneration)
		if err == nil {
			return nil
		}

		if !isPreconditionFailed(err) {
			return fmt.Errorf("failed to write %s: %w", key, err)
		}
		// Precondition failed, retry
	}

	return fmt.Errorf("%w: %s after %d attempts", ErrPreconditionFailed, key, maxRetries)
}

// doWrite performs a single write attempt.
func (s *GCSStorage) doWrite(ctx context.Context, key string, data []byte, expectedGeneration int64) error {
	obj := s.bucket.Object(key)

	var writer io.WriteCloser
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
		return err
	}

	return writer.Close()
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
// Since the GCS client is managed externally, this is a no-op.
func (s *GCSStorage) Close(ctx context.Context) error {
	return nil
}
