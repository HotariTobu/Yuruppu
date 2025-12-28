package history

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"yuruppu/internal/storage"
)

// Message represents a single message in conversation history.
type Message struct {
	Role      string    `json:"Role"` // "user" or "assistant"
	Content   string    `json:"Content"`
	Timestamp time.Time `json:"Timestamp"`
}

// ConversationHistory holds messages for a specific source.
type ConversationHistory struct {
	SourceID string
	Messages []Message
}

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
// Returns empty slice if no history exists.
// Returns error if sourceID is empty.
func (r *Repository) GetHistory(ctx context.Context, sourceID string) ([]Message, error) {
	if err := validateSourceID(sourceID); err != nil {
		return nil, err
	}

	key := sourceID + ".jsonl"

	data, _, err := r.storage.Read(ctx, key)
	if err != nil {
		return nil, &ReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}

	if data == nil {
		return []Message{}, nil
	}

	return r.parseJSONL(data, sourceID)
}

// AppendMessages saves user message and bot response atomically.
// Uses generation precondition to detect concurrent modifications.
// Returns error if sourceID is empty.
func (r *Repository) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg Message) error {
	if err := validateSourceID(sourceID); err != nil {
		return err
	}

	key := sourceID + ".jsonl"

	// Read existing history with generation
	data, generation, err := r.storage.Read(ctx, key)
	if err != nil {
		return &ReadError{Message: fmt.Sprintf("failed to read existing history for %s: %v", sourceID, err)}
	}

	var messages []Message
	if data != nil {
		messages, err = r.parseJSONL(data, sourceID)
		if err != nil {
			return err // parseJSONL already returns ReadError
		}
	}

	// Append new messages
	messages = append(messages, userMsg, botMsg)

	// Serialize to JSONL
	newData, err := r.serializeJSONL(messages, sourceID)
	if err != nil {
		return err // serializeJSONL already returns WriteError
	}

	// Write back with generation precondition (storage handles retry)
	if err := r.storage.Write(ctx, key, newData, generation); err != nil {
		return &WriteError{Message: fmt.Sprintf("failed to write history for %s: %v", sourceID, err)}
	}

	return nil
}

// Close releases repository resources.
func (r *Repository) Close(ctx context.Context) error {
	return r.storage.Close(ctx)
}

// validateSourceID checks if sourceID is valid.
func validateSourceID(sourceID string) error {
	if strings.TrimSpace(sourceID) == "" {
		return &ValidationError{Message: "sourceID cannot be empty"}
	}
	return nil
}

// parseJSONL parses JSONL data into messages.
func (r *Repository) parseJSONL(data []byte, sourceID string) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, &ReadError{Message: fmt.Sprintf("failed to parse JSONL for %s: %v", sourceID, err)}
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, &ReadError{Message: fmt.Sprintf("failed to read history for %s: %v", sourceID, err)}
	}

	return messages, nil
}

// serializeJSONL serializes messages to JSONL format.
func (r *Repository) serializeJSONL(messages []Message, sourceID string) ([]byte, error) {
	var buf bytes.Buffer

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return nil, &WriteError{Message: fmt.Sprintf("failed to marshal message for %s: %v", sourceID, err)}
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
