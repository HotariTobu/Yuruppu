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

// errorResponse creates an error response map.
func errorResponse(msg string) (map[string]any, error) {
	return map[string]any{
		"success": false,
		"error":   msg,
	}, nil
}

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
		return errorResponse("internal error: source ID not found")
	}

	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errorResponse("internal error: user ID not found")
	}

	// FR-003: Users can only create events from group chats
	// In 1:1 chats, sourceID == userID
	if sourceID == userID {
		return errorResponse("events can only be created in group chats")
	}

	// Validate and extract arguments
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return errorResponse("title is required and must be a non-empty string")
	}

	startTimeStr, ok := args["start_time"].(string)
	if !ok || startTimeStr == "" {
		return errorResponse("start_time is required")
	}

	endTimeStr, ok := args["end_time"].(string)
	if !ok || endTimeStr == "" {
		return errorResponse("end_time is required")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return errorResponse(fmt.Sprintf("invalid start_time format: %v", err))
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return errorResponse(fmt.Sprintf("invalid end_time format: %v", err))
	}

	// FR-008: startTime must be in the future
	now := time.Now()
	if !startTime.After(now) {
		return errorResponse("start_time must be in the future, not in the past")
	}

	// FR-008: endTime must be after startTime
	if !endTime.After(startTime) {
		return errorResponse("end_time must be after start_time")
	}

	// Validate capacity
	capacity, ok := args["capacity"].(int)
	if !ok {
		// Try to convert from float64 (JSON number default)
		if capacityFloat, ok := args["capacity"].(float64); ok {
			capacity = int(capacityFloat)
		} else {
			return errorResponse("capacity is required and must be an integer")
		}
	}

	if capacity <= 0 {
		return errorResponse("capacity must be greater than 0")
	}

	// Extract fee
	fee, ok := args["fee"].(string)
	if !ok || fee == "" {
		return errorResponse("fee is required and must be a non-empty string")
	}

	// Extract description
	description, ok := args["description"].(string)
	if !ok || description == "" {
		return errorResponse("description is required and must be a non-empty string")
	}

	// Extract show_creator
	showCreator, ok := args["show_creator"].(bool)
	if !ok {
		return errorResponse("show_creator is required and must be a boolean")
	}

	// Create event struct
	ev := &event.Event{
		ChatRoomID:  sourceID,
		CreatorID:   userID,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		Capacity:    capacity,
		Fee:         fee,
		Description: description,
		ShowCreator: showCreator,
	}

	// Call service to create event
	if err := t.eventService.Create(ctx, ev); err != nil {
		return errorResponse(err.Error())
	}

	return map[string]any{
		"success":      true,
		"chat_room_id": sourceID,
	}, nil
}

// IsFinal returns true if the event was created successfully.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	success, ok := validatedResult["success"].(bool)
	return ok && success
}
