package userprofile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"yuruppu/internal/storage"
)

// UserProfile contains LINE user profile information.
type UserProfile struct {
	DisplayName     string `json:"displayName"`
	PictureURL      string `json:"pictureUrl,omitempty"`
	PictureMIMEType string `json:"pictureMimeType,omitempty"`
	StatusMessage   string `json:"statusMessage,omitempty"`
}

// Service provides user profile management with caching and persistence.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger

	cache sync.Map // userID -> *UserProfile
}

// NewService creates a new user profile service.
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

// GetUserProfile retrieves user profile from cache or storage.
func (s *Service) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	if cached, ok := s.cache.Load(userID); ok {
		if profile, ok := cached.(*UserProfile); ok {
			return profile, nil
		}
		// Cache contains wrong type, delete and continue to storage
		s.cache.Delete(userID)
	}

	data, _, err := s.storage.Read(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to read user profile: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("user profile not found: %s", userID)
	}

	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	s.cache.Store(userID, &profile)
	return &profile, nil
}

// SetUserProfile stores user profile to cache and storage.
func (s *Service) SetUserProfile(ctx context.Context, userID string, profile *UserProfile) error {
	if profile == nil {
		return errors.New("profile cannot be nil")
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal user profile: %w", err)
	}

	_, err = s.storage.Write(ctx, userID, "application/json", data, 0)
	if err != nil {
		return fmt.Errorf("failed to write user profile: %w", err)
	}

	// Update cache only after successful storage write
	s.cache.Store(userID, profile)
	return nil
}
