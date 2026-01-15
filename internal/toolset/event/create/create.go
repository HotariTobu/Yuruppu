package create

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// EventService provides access to event operations.
type EventService interface {
	Create(ctx context.Context, ev *event.Event) error
}

// Tool implements the create_event tool for creating events.
type Tool struct {
	eventService EventService
	logger       *slog.Logger
}

// New creates a new create_event tool with the specified event service.
func New(eventService EventService, logger *slog.Logger) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		eventService: eventService,
		logger:       logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "create_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to create a new event in a group chat."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback creates a new event.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Get sourceID and userID from context
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "source ID not found in context")
		return nil, errors.New("internal error")
	}

	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "user ID not found in context")
		return nil, errors.New("internal error")
	}

	// FR-003: Users can only create events from group chats
	// In 1:1 chats, sourceID == userID
	if sourceID == userID {
		return nil, errors.New("events can only be created in group chats")
	}

	// Extract arguments (validated by schema)
	title, ok := args["title"].(string)
	if !ok {
		return nil, errors.New("invalid title")
	}

	startTimeStr, ok := args["start_time"].(string)
	if !ok {
		return nil, errors.New("invalid start_time")
	}

	endTimeStr, ok := args["end_time"].(string)
	if !ok {
		return nil, errors.New("invalid end_time")
	}

	fee, ok := args["fee"].(string)
	if !ok {
		return nil, errors.New("invalid fee")
	}

	capacityFloat, ok := args["capacity"].(float64)
	if !ok {
		return nil, errors.New("invalid capacity")
	}
	capacity := int(capacityFloat)

	description, ok := args["description"].(string)
	if !ok {
		return nil, errors.New("invalid description")
	}

	showCreator, ok := args["show_creator"].(bool)
	if !ok {
		return nil, errors.New("invalid show_creator")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		t.logger.ErrorContext(ctx, "invalid start_time format", slog.Any("error", err))
		return nil, errors.New("invalid start_time format")
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		t.logger.ErrorContext(ctx, "invalid end_time format", slog.Any("error", err))
		return nil, errors.New("invalid end_time format")
	}

	// FR-008: startTime must be in the future
	now := time.Now()
	if !startTime.After(now) {
		return nil, errors.New("start_time must be in the future")
	}

	// FR-008: endTime must be after startTime
	if !endTime.After(startTime) {
		return nil, errors.New("end_time must be after start_time")
	}

	// Create event struct
	ev := &event.Event{
		ChatRoomID:  sourceID,
		CreatorID:   userID,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		Fee:         fee,
		Capacity:    capacity,
		Description: description,
		ShowCreator: showCreator,
	}

	// Call service to create event
	if err := t.eventService.Create(ctx, ev); err != nil {
		t.logger.ErrorContext(ctx, "failed to create event", slog.Any("error", err))
		return nil, errors.New("failed to create event")
	}

	return map[string]any{
		"chat_room_id": sourceID,
	}, nil
}
