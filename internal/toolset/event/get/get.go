package get

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/userprofile"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// JST is Japan Standard Time location (UTC+9).
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// EventService provides access to event operations.
type EventService interface {
	Get(ctx context.Context, chatRoomID string) (*event.Event, error)
}

// UserProfileService provides access to user profile operations.
type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
}

// Tool implements the get_event tool for retrieving event details.
type Tool struct {
	eventService       EventService
	userProfileService UserProfileService
	logger             *slog.Logger
}

// New creates a new get_event tool with the specified services.
func New(eventService EventService, userProfileService UserProfileService, logger *slog.Logger) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if userProfileService == nil {
		return nil, errors.New("userProfileService cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		eventService:       eventService,
		userProfileService: userProfileService,
		logger:             logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "get_event"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to retrieve event details."
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
	chatRoomID, err := t.determineChatRoomID(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to determine chat room ID: %w", err)
	}

	// Retrieve event from service
	ev, err := t.eventService.Get(ctx, chatRoomID)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to get event", slog.String("chatRoomID", chatRoomID), slog.Any("error", err))
		return nil, errors.New("event not found")
	}

	// Build response
	response := map[string]any{
		"title":       ev.Title,
		"start_time":  ev.StartTime.In(JST).Format(time.RFC3339),
		"end_time":    ev.EndTime.In(JST).Format(time.RFC3339),
		"fee":         ev.Fee,
		"capacity":    ev.Capacity,
		"description": ev.Description,
	}

	// Resolve creator name if showCreator is true
	if ev.ShowCreator {
		userProfile, err := t.userProfileService.GetUserProfile(ctx, ev.CreatorID)
		if err != nil {
			t.logger.ErrorContext(ctx, "failed to get creator profile", slog.String("creatorID", ev.CreatorID), slog.Any("error", err))
			return nil, errors.New("failed to get creator profile")
		}
		response["creator_name"] = userProfile.DisplayName
	}

	return response, nil
}

// determineChatRoomID determines the chat room ID from args or context.
func (t *Tool) determineChatRoomID(ctx context.Context, args map[string]any) (string, error) {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "chat type not found in context")
		return "", errors.New("internal error")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "source ID not found in context")
		return "", errors.New("internal error")
	}

	if chatRoomIDArg, ok := args["chat_room_id"]; ok {
		chatRoomID, ok := chatRoomIDArg.(string)
		if !ok {
			return "", errors.New("invalid chat_room_id")
		}
		return chatRoomID, nil
	}
	if chatType == line.ChatTypeGroup {
		return sourceID, nil
	}
	return "", errors.New("chat_room_id is required in 1-on-1 chats")
}
