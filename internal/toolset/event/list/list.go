package list

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

// JST is Japan Standard Time location (UTC+9).
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// EventService provides access to event list operations.
type EventService interface {
	List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
}

// Tool implements the list_events tool for retrieving filtered event lists.
type Tool struct {
	eventService  EventService
	maxPeriodDays int
	limit         int
	logger        *slog.Logger
}

// New creates a new list_events tool with the specified service and configuration.
func New(eventService EventService, maxPeriodDays, limit int, logger *slog.Logger) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if maxPeriodDays <= 0 {
		return nil, errors.New("maxPeriodDays must be positive")
	}
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		eventService:  eventService,
		maxPeriodDays: maxPeriodDays,
		limit:         limit,
		logger:        logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "list_events"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to retrieve a list of events with optional filters."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback retrieves a filtered list of events.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Build ListOptions
	opts := event.ListOptions{}

	// Handle created_by_me filter
	if createdByMeArg, ok := args["created_by_me"]; ok {
		createdByMe, ok := createdByMeArg.(bool)
		if !ok {
			return nil, errors.New("invalid created_by_me")
		}
		if createdByMe {
			userID, ok := line.UserIDFromContext(ctx)
			if !ok {
				t.logger.ErrorContext(ctx, "user ID not found in context")
				return nil, errors.New("internal error")
			}
			opts.CreatorID = &userID
		}
	}

	// Handle start filter
	var start *time.Time
	if startArg, ok := args["start"]; ok {
		startStr, ok := startArg.(string)
		if !ok {
			return nil, errors.New("invalid start")
		}
		parsedStart, err := parseTimeParameter(startStr)
		if err != nil {
			t.logger.ErrorContext(ctx, "invalid start time", slog.Any("error", err))
			return nil, errors.New("invalid start")
		}
		start = &parsedStart
		opts.Start = start
	}

	// Handle end filter
	var end *time.Time
	if endArg, ok := args["end"]; ok {
		endStr, ok := endArg.(string)
		if !ok {
			return nil, errors.New("invalid end")
		}
		parsedEnd, err := parseTimeParameter(endStr)
		if err != nil {
			t.logger.ErrorContext(ctx, "invalid end time", slog.Any("error", err))
			return nil, errors.New("invalid end")
		}
		end = &parsedEnd
		opts.End = end
	}

	// Validate period if both start and end are specified
	if start != nil && end != nil {
		// Check end is after start
		if end.Before(*start) {
			return nil, errors.New("end time must be after start time")
		}
		// Check period doesn't exceed maxPeriodDays
		duration := end.Sub(*start)
		maxDuration := time.Duration(t.maxPeriodDays) * 24 * time.Hour
		if duration > maxDuration {
			return nil, errors.New("period is too long")
		}
		// No limit when both start and end specified
		opts.Limit = 0
	} else {
		// Apply limit when only start or end (or neither) specified
		opts.Limit = t.limit
	}

	// Retrieve events from service
	events, err := t.eventService.List(ctx, opts)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to list events", slog.Any("error", err))
		return nil, errors.New("failed to list events")
	}

	// Build response with limited fields
	eventList := make([]any, len(events))
	for i, ev := range events {
		eventList[i] = map[string]any{
			"chat_room_id": ev.ChatRoomID,
			"title":        ev.Title,
			"start_time":   ev.StartTime.In(JST).Format(time.RFC3339),
			"end_time":     ev.EndTime.In(JST).Format(time.RFC3339),
			"fee":          ev.Fee,
		}
	}

	return map[string]any{
		"events": eventList,
	}, nil
}

// parseTimeParameter parses a time parameter that can be either "today" or RFC3339 format.
// "today" resolves to current date 00:00:00 in JST.
func parseTimeParameter(s string) (time.Time, error) {
	if s == "today" {
		// Get current time in JST
		now := time.Now().In(JST)
		// Set to 00:00:00
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, JST), nil
	}
	// Parse as RFC3339
	return time.Parse(time.RFC3339, s)
}
