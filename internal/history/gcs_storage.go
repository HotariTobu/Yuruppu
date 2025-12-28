package history

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
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
// It stores conversation history in JSONL format, one file per SourceID.
type GCSStorage struct {
	bucket BucketHandle
}

// NewGCSStorage creates a new GCS storage backend.
// The client is managed externally and should not be closed by this struct.
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

// GetHistory retrieves conversation history for a source.
// Returns empty slice if no history exists.
func (s *GCSStorage) GetHistory(ctx context.Context, sourceID string) ([]Message, error) {
	// Check for timeout/cancellation at the start
	select {
	case <-ctx.Done():
		return nil, &StorageTimeoutError{Message: "context cancelled before reading history"}
	default:
	}

	objectName := sourceID + ".jsonl"
	obj := s.bucket.Object(objectName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		// Object doesn't exist - return empty slice
		if errors.Is(err, storage.ErrObjectNotExist) {
			return []Message{}, nil
		}
		return nil, &StorageReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}
	defer reader.Close()

	var messages []Message
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, &StorageReadError{Message: fmt.Sprintf("failed to parse JSONL for %s: %v", sourceID, err)}
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, &StorageReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}

	return messages, nil
}

// AppendMessages saves user message and bot response atomically.
func (s *GCSStorage) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg Message) error {
	// Check for timeout/cancellation at the start
	select {
	case <-ctx.Done():
		return &StorageTimeoutError{Message: "context cancelled before appending messages"}
	default:
	}

	objectName := sourceID + ".jsonl"
	obj := s.bucket.Object(objectName)

	// Read-Modify-Write pattern:
	// 1. Read existing history (if exists) and get generation
	existingMessages, generation, err := s.readExistingHistory(ctx, obj, sourceID)
	if err != nil {
		return err
	}

	// 2. Append new messages
	allMessages := make([]Message, 0, len(existingMessages)+2)
	allMessages = append(allMessages, existingMessages...)
	allMessages = append(allMessages, userMsg, botMsg)

	// 3. Write back with generation precondition
	return s.writeHistory(ctx, obj, allMessages, generation, sourceID)
}

// readExistingHistory reads existing messages and returns generation.
// Returns empty slice and generation 0 if object doesn't exist.
func (s *GCSStorage) readExistingHistory(ctx context.Context, obj ObjectHandle, sourceID string) ([]Message, int64, error) {
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, 0, nil
		}
		return nil, 0, &StorageWriteError{Message: fmt.Sprintf("failed to read existing history for %s: %v", sourceID, err)}
	}

	generation := attrs.Generation

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, 0, &StorageWriteError{Message: fmt.Sprintf("failed to read existing history for %s: %v", sourceID, err)}
	}
	defer reader.Close()

	var messages []Message
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, 0, &StorageWriteError{Message: fmt.Sprintf("failed to parse existing history for %s: %v", sourceID, err)}
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, &StorageWriteError{Message: fmt.Sprintf("failed to read existing history for %s: %v", sourceID, err)}
	}

	return messages, generation, nil
}

// writeHistory writes messages to GCS with generation precondition.
func (s *GCSStorage) writeHistory(ctx context.Context, obj ObjectHandle, messages []Message, generation int64, sourceID string) error {
	var writer io.WriteCloser
	if generation == 0 {
		writer = obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	} else {
		writer = obj.If(storage.Conditions{GenerationMatch: generation}).NewWriter(ctx)
	}

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			_ = writer.Close()
			return &StorageWriteError{Message: fmt.Sprintf("failed to marshal message for %s: %v", sourceID, err)}
		}
		if _, err := writer.Write(data); err != nil {
			_ = writer.Close()
			return &StorageWriteError{Message: fmt.Sprintf("failed to write message for %s: %v", sourceID, err)}
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			_ = writer.Close()
			return &StorageWriteError{Message: fmt.Sprintf("failed to write newline for %s: %v", sourceID, err)}
		}
	}

	if err := writer.Close(); err != nil {
		return &StorageWriteError{Message: fmt.Sprintf("failed to write history for %s: %v", sourceID, err)}
	}

	return nil
}

// Close releases storage resources.
// Since the GCS client is managed externally, this is a no-op.
func (s *GCSStorage) Close(ctx context.Context) error {
	return nil
}
