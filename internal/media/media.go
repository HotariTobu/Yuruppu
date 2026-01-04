package media

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"
	"yuruppu/internal/storage"

	"github.com/google/uuid"
)

// sourceIDPattern validates LINE source IDs (user IDs, group IDs, room IDs).
// LINE IDs are alphanumeric strings, typically 33 characters (U/C/R prefix + 32 hex).
// Pattern allows alphanumeric and hyphens but prevents path traversal sequences.
var sourceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// Service provides media storage functionality.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger
}

// NewService creates a new media service.
func NewService(storage storage.Storage, logger *slog.Logger) (*Service, error) {
	if storage == nil {
		return nil, errors.New("storage cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Service{
		storage: storage,
		logger:  logger,
	}, nil
}

// Store saves media data to storage.
// sourceID is the LINE source identifier (user or group ID).
// Returns the storage key of the stored media.
func (s *Service) Store(ctx context.Context, sourceID string, data []byte, mimeType string) (string, error) {
	// Validate sourceID to prevent path traversal attacks
	if sourceID == "" || !sourceIDPattern.MatchString(sourceID) {
		return "", fmt.Errorf("invalid sourceID: %q", sourceID)
	}

	// Generate storage key: {sourceID}/{uuidv7}
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUIDv7: %w", err)
	}
	storageKey := sourceID + "/" + id.String()

	// Write to storage
	_, err = s.storage.Write(ctx, storageKey, mimeType, data, 0)
	if err != nil {
		return "", fmt.Errorf("failed to write media to storage: %w", err)
	}

	s.logger.DebugContext(ctx, "media stored successfully",
		slog.String("storageKey", storageKey),
		slog.String("mimeType", mimeType),
		slog.Int("dataSize", len(data)),
	)

	return storageKey, nil
}

// GetSignedURL returns a signed URL for accessing the media at the given storage key.
func (s *Service) GetSignedURL(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return s.storage.GetSignedURL(ctx, storageKey, "GET", ttl)
}
