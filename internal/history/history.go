package history

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"yuruppu/internal/message"
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
func (r *Repository) GetHistory(ctx context.Context, sourceID string) ([]message.Message, int64, error) {
	if strings.TrimSpace(sourceID) == "" {
		return nil, 0, &ValidationError{Message: "sourceID cannot be empty"}
	}

	data, generation, err := r.storage.Read(ctx, sourceID)
	if err != nil {
		return nil, 0, &ReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}

	if data == nil {
		return []message.Message{}, generation, nil
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
func (r *Repository) PutHistory(ctx context.Context, sourceID string, messages []message.Message, expectedGeneration int64) error {
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

// parseJSONL parses JSONL data into messages.
func (r *Repository) parseJSONL(data []byte) ([]message.Message, error) {
	var messages []message.Message
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg message.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// serializeJSONL serializes messages to JSONL format.
func (r *Repository) serializeJSONL(messages []message.Message) ([]byte, error) {
	var buf bytes.Buffer

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
