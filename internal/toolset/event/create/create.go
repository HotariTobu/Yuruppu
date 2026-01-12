package create

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// EventService provides event operations.
type EventService interface {
	Create(ctx context.Context, ev *event.Event) error
}

// Tool implements the event creation tool.
type Tool struct {
	service EventService
	logger  *slog.Logger
}

// NewTool creates a new event creation tool.
func NewTool(service EventService, logger *slog.Logger) (*Tool, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return &Tool{
		service: service,
		logger:  logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "create_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Create a new event in the current group chat. Only works in group chats. Each group can have only one event."
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
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "source ID not found in context")
		return nil, fmt.Errorf("internal error")
	}

	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "user ID not found in context")
		return nil, fmt.Errorf("internal error")
	}

	// FR-003: Check if this is a group chat (sourceID != userID means group/room)
	if sourceID == userID {
		return map[string]any{
			"success": false,
			"error":   "events can only be created in group chats",
		}, nil
	}

	title, _ := args["title"].(string)
	startTimeStr, _ := args["start_time"].(string)
	endTimeStr, _ := args["end_time"].(string)
	capacityFloat, _ := args["capacity"].(float64)
	capacity := int(capacityFloat)
	fee, _ := args["fee"].(string)
	description, _ := args["description"].(string)
	showCreator, _ := args["show_creator"].(bool)

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   "invalid start_time format, use RFC3339 (e.g., 2025-01-15T18:00:00+09:00)",
		}, nil
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   "invalid end_time format, use RFC3339 (e.g., 2025-01-15T21:00:00+09:00)",
		}, nil
	}

	// FR-008: start_time must be in the future
	if !startTime.After(time.Now()) {
		return map[string]any{
			"success": false,
			"error":   "start_time must be in the future",
		}, nil
	}

	// FR-008: end_time must be after start_time
	if !endTime.After(startTime) {
		return map[string]any{
			"success": false,
			"error":   "end_time must be after start_time",
		}, nil
	}

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

	// FR-004: Only one event per group chat
	if err := t.service.Create(ctx, ev); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return map[string]any{
				"success": false,
				"error":   "an event already exists in this group chat",
			}, nil
		}
		t.logger.ErrorContext(ctx, "failed to create event",
			slog.String("chatRoomID", sourceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to create event")
	}

	t.logger.InfoContext(ctx, "event created",
		slog.String("chatRoomID", sourceID),
		slog.String("title", title),
	)

	return map[string]any{
		"success":      true,
		"chat_room_id": sourceID,
	}, nil
}
