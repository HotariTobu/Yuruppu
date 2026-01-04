package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	lineclient "yuruppu/internal/line/client"
	"yuruppu/internal/profile"
)

// LineClient provides access to LINE API.
type LineClient interface {
	GetMessageContent(messageID string) (data []byte, mimeType string, err error)
	GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
}

// ProfileService provides access to user profiles.
type ProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error)
	SetUserProfile(ctx context.Context, userID string, profile *profile.UserProfile) error
}

// HistoryService provides access to conversation history.
type HistoryService interface {
	GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error)
	PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error)
}

// MediaService provides media storage functionality.
type MediaService interface {
	Store(ctx context.Context, sourceID string, data []byte, mimeType string) (string, error)
	GetSignedURL(ctx context.Context, storageKey string, ttl time.Duration) (string, error)
}

// Handler implements the server.Handler interface for handling LINE messages.
type Handler struct {
	lineClient     LineClient
	profileService ProfileService
	history        HistoryService
	media          MediaService
	agent          agent.Agent
	logger         *slog.Logger
}

// NewHandler creates a new Handler with the given dependencies.
// Returns error if any dependency is nil.
func NewHandler(lineClient LineClient, profileService ProfileService, historySvc HistoryService, mediaSvc MediaService, agent agent.Agent, logger *slog.Logger) (*Handler, error) {
	if lineClient == nil {
		return nil, fmt.Errorf("lineClient is required")
	}
	if profileService == nil {
		return nil, fmt.Errorf("profileService is required")
	}
	if historySvc == nil {
		return nil, fmt.Errorf("historySvc is required")
	}
	if mediaSvc == nil {
		return nil, fmt.Errorf("mediaSvc is required")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Handler{
		lineClient:     lineClient,
		profileService: profileService,
		history:        historySvc,
		media:          mediaSvc,
		agent:          agent,
		logger:         logger,
	}, nil
}
