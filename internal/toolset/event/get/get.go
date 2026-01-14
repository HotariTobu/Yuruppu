package get

import (
	"context"
	_ "embed"
	"errors"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// JST is Japan Standard Time location (UTC+9).
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// errorResponse creates an error response map.
func errorResponse(msg string) (map[string]any, error) {
	return map[string]any{
		"success": false,
		"error":   msg,
	}, nil
}

// EventService provides access to event operations.
type EventService interface {
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
}

// ProfileService provides access to user profile operations.
type ProfileService interface {
	GetDisplayName(ctx context.Context, userID string) (string, error)
}

// Tool implements the get_event tool for retrieving event details.
type Tool struct {
	eventService   EventService
	profileService ProfileService
}

// New creates a new get_event tool with the specified services.
func New(eventService EventService, profileService ProfileService) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if profileService == nil {
		return nil, errors.New("profileService cannot be nil")
	}
	return &Tool{
		eventService:   eventService,
		profileService: profileService,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "get_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to retrieve event details from a group chat."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback retrieves event details.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Determine chat_room_id (explicit or from context)
	var chatRoomID string
	if chatRoomIDArg, ok := args["chat_room_id"]; ok {
		// Explicit chat_room_id provided
		chatRoomIDStr, ok := chatRoomIDArg.(string)
		if !ok {
			return errorResponse("chat_room_id must be a string")
		}
		if chatRoomIDStr == "" {
			return errorResponse("chat_room_id cannot be empty")
		}
		chatRoomID = chatRoomIDStr
	} else {
		// Use sourceID from context
		sourceID, ok := line.SourceIDFromContext(ctx)
		if !ok {
			return errorResponse("internal error: source ID not found")
		}
		chatRoomID = sourceID
	}

	// Retrieve event from service
	ev, err := t.eventService.Get(ctx, chatRoomID)
	if err != nil {
		return errorResponse(err.Error())
	}

	// Build response
	response := map[string]any{
		"success":     true,
		"title":       ev.Title,
		"start_time":  ev.StartTime.In(JST).Format(time.RFC3339),
		"end_time":    ev.EndTime.In(JST).Format(time.RFC3339),
		"fee":         ev.Fee,
		"capacity":    ev.Capacity,
		"description": ev.Description,
	}

	// Resolve creator name if showCreator is true
	if ev.ShowCreator {
		displayName, err := t.profileService.GetDisplayName(ctx, ev.CreatorID)
		if err != nil {
			return errorResponse(err.Error())
		}
		response["creator_name"] = displayName
	}

	return response, nil
}

// IsFinal returns true if the event was retrieved successfully.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	success, ok := validatedResult["success"].(bool)
	return ok && success
}
