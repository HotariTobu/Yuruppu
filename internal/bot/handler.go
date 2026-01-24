package bot

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/history"
	lineclient "yuruppu/internal/line/client"
	"yuruppu/internal/userprofile"
)

// Agent defines the interface for LLM agents used by bot handler.
type Agent interface {
	Generate(ctx context.Context, history []agent.Message) (*agent.AssistantMessage, error)
}

// LineClient provides access to LINE API.
type LineClient interface {
	GetMessageContent(messageID string) (data []byte, mimeType string, err error)
	GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
	GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
	GetGroupMemberCount(ctx context.Context, groupID string) (int, error)
	ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error
}

// HandlerConfig holds handler configuration.
type HandlerConfig struct {
	TypingIndicatorDelay   time.Duration // time to wait before showing indicator (default 3s)
	TypingIndicatorTimeout time.Duration // indicator display duration (5-60s)
}

// UserProfileService provides access to user profiles.
type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
	SetUserProfile(ctx context.Context, userID string, profile *userprofile.UserProfile) error
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

// GroupProfileService provides access to group profiles.
type GroupProfileService interface {
	GetGroupProfile(ctx context.Context, groupID string) (*groupprofile.GroupProfile, error)
	SetGroupProfile(ctx context.Context, groupID string, profile *groupprofile.GroupProfile) error
}

// Handler implements the server.Handler interface for handling LINE messages.
type Handler struct {
	lineClient          LineClient
	userProfileService  UserProfileService
	groupProfileService GroupProfileService
	history             HistoryService
	media               MediaService
	agent               Agent
	config              HandlerConfig
	logger              *slog.Logger
}

// NewHandler creates a new Handler with the given dependencies.
// Returns error if any dependency is nil.
func NewHandler(lineClient LineClient, userProfileSvc UserProfileService, groupProfileSvc GroupProfileService, historySvc HistoryService, mediaSvc MediaService, agent Agent, config HandlerConfig, logger *slog.Logger) (*Handler, error) {
	if lineClient == nil {
		return nil, errors.New("lineClient is required")
	}
	if userProfileSvc == nil {
		return nil, errors.New("userProfileSvc is required")
	}
	if groupProfileSvc == nil {
		return nil, errors.New("groupProfileSvc is required")
	}
	if historySvc == nil {
		return nil, errors.New("historySvc is required")
	}
	if mediaSvc == nil {
		return nil, errors.New("mediaSvc is required")
	}
	if agent == nil {
		return nil, errors.New("agent is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}
	return &Handler{
		lineClient:          lineClient,
		userProfileService:  userProfileSvc,
		groupProfileService: groupProfileSvc,
		history:             historySvc,
		media:               mediaSvc,
		agent:               agent,
		config:              config,
		logger:              logger,
	}, nil
}
