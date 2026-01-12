package list

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"yuruppu/internal/event"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

const maxListResults = 100

// EventService provides event operations.
type EventService interface {
	List(ctx context.Context) ([]*event.Event, error)
}

// Tool implements the event listing tool.
type Tool struct {
	service EventService
	logger  *slog.Logger
}

// NewTool creates a new event listing tool.
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
	return "list_events"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "List events with optional filters. Can filter by creator (created_by_me) and period (upcoming, past, or custom date range)."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback lists events with optional filters.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "user ID not found in context")
		return nil, fmt.Errorf("internal error")
	}

	createdByMe, _ := args["created_by_me"].(bool)
	periodFilter, _ := args["period_filter"].(string)
	startDateStr, _ := args["start_date"].(string)
	endDateStr, _ := args["end_date"].(string)

	var startDate, endDate time.Time
	var err error
	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return map[string]any{
				"events":    []any{},
				"truncated": false,
			}, nil
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return map[string]any{
				"events":    []any{},
				"truncated": false,
			}, nil
		}
	}

	allEvents, err := t.service.List(ctx)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to list events", slog.Any("error", err))
		return nil, fmt.Errorf("failed to list events")
	}

	now := time.Now()

	filtered := make([]*event.Event, 0)
	for _, ev := range allEvents {
		// FR-011: Creator filter
		if createdByMe && ev.CreatorID != userID {
			continue
		}

		// FR-012: Period filter
		switch periodFilter {
		case "upcoming":
			if !ev.StartTime.After(now) {
				continue
			}
		case "past":
			if !ev.EndTime.Before(now) {
				continue
			}
		case "custom":
			if !startDate.IsZero() && ev.StartTime.Before(startDate) {
				continue
			}
			if !endDate.IsZero() && ev.StartTime.After(endDate) {
				continue
			}
		}

		filtered = append(filtered, ev)
	}

	// FR-014: Sort by start time ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.Before(filtered[j].StartTime)
	})

	// FR-015: Limit to 100 items
	truncated := false
	if len(filtered) > maxListResults {
		filtered = filtered[:maxListResults]
		truncated = true
	}

	// FR-016: Build response with limited fields
	events := make([]any, 0, len(filtered))
	for _, ev := range filtered {
		events = append(events, map[string]any{
			"chat_room_id": ev.ChatRoomID,
			"title":        ev.Title,
			"start_time":   ev.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			"end_time":     ev.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			"fee":          ev.Fee,
		})
	}

	return map[string]any{
		"events":    events,
		"truncated": truncated,
	}, nil
}
