package history

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"yuruppu/internal/storage"
)

var invalidSourceIDPattern = regexp.MustCompile(`/|\.\.`)

// Service provides access to conversation history storage.
type Service struct {
	storage storage.Storage
}

// NewService creates a new Service with the given storage backend.
// Returns error if storage is nil.
func NewService(s storage.Storage) (*Service, error) {
	if s == nil {
		return nil, errors.New("storage cannot be nil")
	}
	return &Service{storage: s}, nil
}

// GetHistory retrieves conversation history for a source.
// Returns messages and generation for optimistic locking.
// Returns empty slice and generation 0 if no history exists.
// Returns error if sourceID is empty or contains invalid characters.
func (s *Service) GetHistory(ctx context.Context, sourceID string) ([]Message, int64, error) {
	if err := validateSourceID(sourceID); err != nil {
		return nil, 0, err
	}

	data, generation, err := s.storage.Read(ctx, sourceID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read history for %s: %w", sourceID, err)
	}

	if data == nil {
		return []Message{}, generation, nil
	}

	messages, err := parseJSONL(data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse history for %s: %w", sourceID, err)
	}

	return messages, generation, nil
}

// PutHistory saves the given messages as the complete history for a source.
// Uses expectedGeneration for optimistic locking (from GetHistory).
// Returns the new generation number of the saved history.
// Returns error if sourceID is empty/invalid or if generation doesn't match (concurrent modification).
func (s *Service) PutHistory(ctx context.Context, sourceID string, messages []Message, expectedGeneration int64) (int64, error) {
	if err := validateSourceID(sourceID); err != nil {
		return 0, err
	}

	// Serialize to JSONL
	data, err := serializeJSONL(messages)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize history for %s: %w", sourceID, err)
	}

	// Write with generation precondition
	newGen, err := s.storage.Write(ctx, sourceID, "application/jsonl", data, expectedGeneration)
	if err != nil {
		return 0, fmt.Errorf("failed to write history for %s: %w", sourceID, err)
	}

	return newGen, nil
}

// validateSourceID checks if sourceID is valid.
// Rejects empty strings and path traversal attempts.
func validateSourceID(sourceID string) error {
	if strings.TrimSpace(sourceID) == "" {
		return errors.New("sourceID cannot be empty")
	}
	if invalidSourceIDPattern.MatchString(sourceID) {
		return errors.New("sourceID contains invalid characters")
	}
	return nil
}
