package create

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
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
}

// New creates a new create_event tool with the specified event service.
func New(eventService EventService) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	return &Tool{
		eventService: eventService,
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
		return nil, errors.New("source ID not found")
	}

	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("user ID not found")
	}

	// FR-003: Users can only create events from group chats
	// In 1:1 chats, sourceID == userID
	if sourceID == userID {
		return nil, errors.New("events can only be created in group chats")
	}

	// Validate and extract arguments
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, errors.New("title is required")
	}

	startTimeStr, ok := args["start_time"].(string)
	if !ok || startTimeStr == "" {
		return nil, errors.New("start_time is required")
	}

	endTimeStr, ok := args["end_time"].(string)
	if !ok || endTimeStr == "" {
		return nil, errors.New("end_time is required")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
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

	// Extract fee
	fee, ok := args["fee"].(string)
	if !ok || fee == "" {
		return nil, errors.New("fee is required")
	}

	// Validate capacity
	capacity, ok := args["capacity"].(int)
	if !ok {
		// Try to convert from float64 (JSON number default)
		if capacityFloat, ok := args["capacity"].(float64); ok {
			capacity = int(capacityFloat)
		} else {
			return nil, errors.New("capacity is required")
		}
	}

	if capacity <= 0 {
		return nil, errors.New("capacity must be greater than 0")
	}

	// Extract description
	description, ok := args["description"].(string)
	if !ok || description == "" {
		return nil, errors.New("description is required")
	}

	// Extract show_creator
	showCreator, ok := args["show_creator"].(bool)
	if !ok {
		return nil, errors.New("show_creator is required")
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
		return nil, errors.New("failed to create event")
	}

	return map[string]any{
		"chat_room_id": sourceID,
	}, nil
}
