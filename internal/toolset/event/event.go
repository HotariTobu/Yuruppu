package event

import (
	"context"
	"errors"
	"yuruppu/internal/agent"
	"yuruppu/internal/event"
	"yuruppu/internal/toolset/event/create"
	"yuruppu/internal/toolset/event/get"
	"yuruppu/internal/toolset/event/list"
)

// EventService provides access to event operations.
type EventService interface {
	Create(ctx context.Context, ev *event.Event) error
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
	List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
}

// ProfileService provides access to user profile operations.
type ProfileService interface {
	GetDisplayName(ctx context.Context, userID string) (string, error)
}

// NewTools creates all event management tools (create, get, list).
// Returns error if any service is nil or configuration values are invalid.
func NewTools(eventService EventService, profileService ProfileService, listMaxPeriodDays, listLimit int) ([]agent.Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if profileService == nil {
		return nil, errors.New("profileService cannot be nil")
	}
	if listMaxPeriodDays <= 0 {
		return nil, errors.New("listMaxPeriodDays must be positive")
	}
	if listLimit <= 0 {
		return nil, errors.New("listLimit must be positive")
	}

	// Create create_event tool
	createTool, err := create.New(eventService)
	if err != nil {
		return nil, err
	}

	// Create get_event tool
	getTool, err := get.New(eventService, profileService)
	if err != nil {
		return nil, err
	}

	// Create list_events tool
	listTool, err := list.New(eventService, listMaxPeriodDays, listLimit)
	if err != nil {
		return nil, err
	}

	return []agent.Tool{createTool, getTool, listTool}, nil
}
