package history

import (
	"context"
	"fmt"
	"strings"
	"yuruppu/internal/storage"
)

// Repository provides access to conversation history storage.
type Repository struct {
	storage storage.Storage
}

// NewRepository creates a new Repository with the given storage backend.
// Returns error if storage is nil.
func NewRepository(s storage.Storage) (*Repository, error) {
	if s == nil {
		return nil, ErrNilStorage
	}
	return &Repository{storage: s}, nil
}

// GetHistory retrieves conversation history for a source.
// Returns messages and generation for optimistic locking.
// Returns empty slice and generation 0 if no history exists.
// Returns error if sourceID is empty.
func (r *Repository) GetHistory(ctx context.Context, sourceID string) ([]Message, int64, error) {
	if strings.TrimSpace(sourceID) == "" {
		return nil, 0, &ValidationError{Message: "sourceID cannot be empty"}
	}

	data, generation, err := r.storage.Read(ctx, sourceID)
	if err != nil {
		return nil, 0, &ReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}

	if data == nil {
		return []Message{}, generation, nil
	}

	messages, err := r.parseJSONL(data)
	if err != nil {
		return nil, 0, &ReadError{Message: fmt.Sprintf("failed to parse history for %s: %v", sourceID, err)}
	}

	return messages, generation, nil
}

// PutHistory saves the given messages as the complete history for a source.
// Uses expectedGeneration for optimistic locking (from GetHistory).
// Returns error if sourceID is empty or if generation doesn't match (concurrent modification).
func (r *Repository) PutHistory(ctx context.Context, sourceID string, messages []Message, expectedGeneration int64) error {
	if strings.TrimSpace(sourceID) == "" {
		return &ValidationError{Message: "sourceID cannot be empty"}
	}

	// Serialize to JSONL
	data, err := r.serializeJSONL(messages)
	if err != nil {
		return &WriteError{Message: fmt.Sprintf("failed to serialize history for %s: %v", sourceID, err)}
	}

	// Write with generation precondition
	if err := r.storage.Write(ctx, sourceID, "application/jsonl", data, expectedGeneration); err != nil {
		return &WriteError{Message: fmt.Sprintf("failed to write history for %s: %v", sourceID, err)}
	}

	return nil
}

// Close releases repository resources.
func (r *Repository) Close(ctx context.Context) error {
	return r.storage.Close(ctx)
}
