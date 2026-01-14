package list

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

// JST is Japan Standard Time location (UTC+9).
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// errorResponse creates an error response map.
func errorResponse(msg string) (map[string]any, error) {
	return map[string]any{
		"success": false,
		"error":   msg,
	}, nil
}

// EventService provides access to event list operations.
type EventService interface {
	List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
}

// Tool implements the list_events tool for retrieving filtered event lists.
type Tool struct {
	eventService  EventService
	maxPeriodDays int
	limit         int
}

// New creates a new list_events tool with the specified service and configuration.
func New(eventService EventService, maxPeriodDays, limit int) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if maxPeriodDays <= 0 {
		return nil, errors.New("maxPeriodDays must be positive")
	}
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	return &Tool{
		eventService:  eventService,
		maxPeriodDays: maxPeriodDays,
		limit:         limit,
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
			return errorResponse("created_by_me must be a boolean")
		}
		if createdByMe {
			// Get userID from context
			userID, ok := line.UserIDFromContext(ctx)
			if !ok {
				return errorResponse("internal error: user ID not found")
			}
			opts.CreatorID = &userID
		}
	}

	// Handle start filter
	var start *time.Time
	if startArg, ok := args["start"]; ok {
		startStr, ok := startArg.(string)
		if !ok {
			return errorResponse("start must be a string")
		}
		parsedStart, err := parseTimeParameter(startStr)
		if err != nil {
			return errorResponse(fmt.Sprintf("invalid start time: %v", err))
		}
		start = &parsedStart
		opts.Start = start
	}

	// Handle end filter
	var end *time.Time
	if endArg, ok := args["end"]; ok {
		endStr, ok := endArg.(string)
		if !ok {
			return errorResponse("end must be a string")
		}
		parsedEnd, err := parseTimeParameter(endStr)
		if err != nil {
			return errorResponse(fmt.Sprintf("invalid end time: %v", err))
		}
		end = &parsedEnd
		opts.End = end
	}

	// Validate period if both start and end are specified
	if start != nil && end != nil {
		// Check end is after start
		if end.Before(*start) {
			return errorResponse("end time must be after start time")
		}
		// Check period doesn't exceed maxPeriodDays
		duration := end.Sub(*start)
		maxDuration := time.Duration(t.maxPeriodDays) * 24 * time.Hour
		if duration > maxDuration {
			return errorResponse(fmt.Sprintf("period cannot exceed %d days", t.maxPeriodDays))
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
		return errorResponse(err.Error())
	}

	// Build response with limited fields
	eventList := make([]map[string]any, len(events))
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
		"success": true,
		"events":  eventList,
	}, nil
}

// IsFinal returns true if the events were retrieved successfully.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	success, ok := validatedResult["success"].(bool)
	return ok && success
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
