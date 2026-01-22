package event

import (
	"context"
	"errors"
	"log/slog"
	"yuruppu/internal/agent"
	"yuruppu/internal/event"
	"yuruppu/internal/toolset/event/create"
	"yuruppu/internal/toolset/event/list"
	"yuruppu/internal/toolset/event/remove"
	"yuruppu/internal/toolset/event/update"
	"yuruppu/internal/userprofile"
)

// EventService provides access to event operations.
type EventService interface {
	Create(ctx context.Context, ev *event.Event) error
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
	List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
	Update(ctx context.Context, chatRoomID string, description string) error
	Remove(ctx context.Context, chatRoomID string) error
}

// UserProfileService provides access to user profile operations.
type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
}

// LineClient provides LINE messaging operations.
type LineClient interface {
	SendFlexReply(replyToken string, altText string, flexJSON []byte) error
}

// NewTools creates all event management tools (create, list, update, remove).
// Returns error if any service is nil or configuration values are invalid.
func NewTools(eventService EventService, lineClient LineClient, userProfileService UserProfileService, listMaxPeriodDays, listLimit int, logger *slog.Logger) ([]agent.Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if lineClient == nil {
		return nil, errors.New("lineClient cannot be nil")
	}
	if userProfileService == nil {
		return nil, errors.New("userProfileService cannot be nil")
	}
	if listMaxPeriodDays <= 0 {
		return nil, errors.New("listMaxPeriodDays must be positive")
	}
	if listLimit <= 0 {
		return nil, errors.New("listLimit must be positive")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Create create_event tool
	createTool, err := create.New(eventService, logger)
	if err != nil {
		return nil, err
	}

	// Create list_events tool
	listTool, err := list.New(eventService, lineClient, userProfileService, listMaxPeriodDays, listLimit, logger)
	if err != nil {
		return nil, err
	}

	// Create update_event tool
	updateTool, err := update.New(eventService, logger)
	if err != nil {
		return nil, err
	}

	// Create remove_event tool
	removeTool, err := remove.New(eventService, logger)
	if err != nil {
		return nil, err
	}

	return []agent.Tool{createTool, listTool, updateTool, removeTool}, nil
}
