package get

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// EventService provides event operations.
type EventService interface {
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
}

// Tool implements the event retrieval tool.
type Tool struct {
	service EventService
	logger  *slog.Logger
}

// NewTool creates a new event retrieval tool.
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
	return "get_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Get details of an event. If chat_room_id is not provided, gets the event from the current chat room."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback retrieves an event.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	chatRoomID, _ := args["chat_room_id"].(string)
	if chatRoomID == "" {
		sourceID, ok := line.SourceIDFromContext(ctx)
		if !ok {
			t.logger.ErrorContext(ctx, "source ID not found in context")
			return nil, fmt.Errorf("internal error")
		}
		chatRoomID = sourceID
	}

	ev, err := t.service.Get(ctx, chatRoomID)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get event",
			slog.String("chatRoomID", chatRoomID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to get event")
	}
	if ev == nil {
		return map[string]any{
			"found": false,
		}, nil
	}

	// NFR-002: Hide creator if show_creator is false
	eventData := map[string]any{
		"chat_room_id": ev.ChatRoomID,
		"title":        ev.Title,
		"start_time":   ev.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		"end_time":     ev.EndTime.Format("2006-01-02T15:04:05Z07:00"),
		"capacity":     ev.Capacity,
		"fee":          ev.Fee,
		"description":  ev.Description,
	}

	if ev.ShowCreator {
		eventData["creator_id"] = ev.CreatorID
	}

	return map[string]any{
		"found": true,
		"event": eventData,
	}, nil
}
