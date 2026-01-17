package groupprofile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"yuruppu/internal/storage"
)

// GroupProfile contains LINE group profile information.
type GroupProfile struct {
	DisplayName     string `json:"displayName"`
	PictureURL      string `json:"pictureUrl,omitempty"`
	PictureMIMEType string `json:"pictureMimeType,omitempty"`
}

// Service provides group profile management with caching and persistence.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger

	cache sync.Map // groupID -> *GroupProfile
}

// NewService creates a new group profile service.
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

// GetGroupProfile retrieves group profile from cache or storage.
func (s *Service) GetGroupProfile(ctx context.Context, groupID string) (*GroupProfile, error) {
	if cached, ok := s.cache.Load(groupID); ok {
		if profile, ok := cached.(*GroupProfile); ok {
			return profile, nil
		}
		// Cache contains wrong type, delete and continue to storage
		s.cache.Delete(groupID)
	}

	data, _, err := s.storage.Read(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to read group profile: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("group profile not found: %s", groupID)
	}

	var profile GroupProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group profile: %w", err)
	}

	s.cache.Store(groupID, &profile)
	return &profile, nil
}

// SetGroupProfile stores group profile to cache and storage.
func (s *Service) SetGroupProfile(ctx context.Context, groupID string, profile *GroupProfile) error {
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal group profile: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, 0)
	if err != nil {
		return fmt.Errorf("failed to write group profile: %w", err)
	}

	// Update cache only after successful storage write
	s.cache.Store(groupID, profile)
	return nil
}
