package remove

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// EventService provides access to event operations.
type EventService interface {
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
	Delete(ctx context.Context, chatRoomID string) error
}

// Tool implements the delete_event tool for deleting events.
type Tool struct {
	eventService EventService
	logger       *slog.Logger
}

// New creates a new delete_event tool.
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
	return "delete_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to delete (cancel) the event in the current group chat. Only the event creator can delete the event."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback deletes an event.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
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

	// Get existing event to check authorization
	ev, err := t.eventService.Get(ctx, sourceID)
	if err != nil {
		t.logger.ErrorContext(ctx, "event not found", slog.String("chatRoomID", sourceID), slog.Any("error", err))
		return nil, errors.New("event not found")
	}

	// Check authorization
	if ev.CreatorID != userID {
		return nil, errors.New("only the event creator can delete the event")
	}

	// Delete event
	if err := t.eventService.Delete(ctx, sourceID); err != nil {
		t.logger.ErrorContext(ctx, "failed to delete event", slog.Any("error", err))
		return nil, errors.New("failed to delete event")
	}

	return map[string]any{
		"chat_room_id": sourceID,
	}, nil
}
