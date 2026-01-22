package list

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"text/template"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/userprofile"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

//go:embed flex.json
var flexTemplate string

//go:embed alt.txt
var altTemplate string

// JST is Japan Standard Time location (UTC+9).
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// flexEventData represents template data for a single event in flex message.
type flexEventData struct {
	Title       string
	StartTime   string
	EndTime     string
	Fee         string
	Capacity    int
	Description string
	ShowCreator bool
	CreatorName string
}

// EventService provides access to event list operations.
type EventService interface {
	List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
}

// LineClient provides LINE messaging operations.
type LineClient interface {
	SendFlexReply(replyToken string, altText string, flexJSON []byte) error
}

// UserProfileService provides user profile operations.
type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
}

// Tool implements the list_events tool for retrieving filtered event lists.
type Tool struct {
	eventService       EventService
	lineClient         LineClient
	userProfileService UserProfileService
	maxPeriodDays      int
	limit              int
	logger             *slog.Logger
}

// New creates a new list_events tool with the specified service and configuration.
func New(eventService EventService, lineClient LineClient, userProfileService UserProfileService, maxPeriodDays, limit int, logger *slog.Logger) (*Tool, error) {
	if eventService == nil {
		return nil, errors.New("eventService cannot be nil")
	}
	if lineClient == nil {
		return nil, errors.New("lineClient cannot be nil")
	}
	if userProfileService == nil {
		return nil, errors.New("userProfileService cannot be nil")
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
		eventService:       eventService,
		lineClient:         lineClient,
		userProfileService: userProfileService,
		maxPeriodDays:      maxPeriodDays,
		limit:              limit,
		logger:             logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "list_events"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Sends a Flex Message with full event details directly to the chat."
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
	// Get context values first
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "user ID not found in context")
		return nil, errors.New("internal error")
	}
	replyToken, ok := line.ReplyTokenFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "reply token not found in context")
		return nil, errors.New("internal error")
	}

	// Build ListOptions
	opts := event.ListOptions{}

	// Handle created_by_me filter
	if createdByMeArg, ok := args["created_by_me"]; ok {
		createdByMe, ok := createdByMeArg.(bool)
		if !ok {
			return nil, errors.New("invalid created_by_me")
		}
		if createdByMe {
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

	// If no events, return no_events status without sending message
	if len(events) == 0 {
		return map[string]any{
			"status": "no_events",
		}, nil
	}

	// Build template data for each event
	eventDataList := make([]flexEventData, len(events))
	for i, ev := range events {
		eventData := flexEventData{
			Title:       ev.Title,
			StartTime:   formatDisplayTime(ev.StartTime),
			EndTime:     formatDisplayTime(ev.EndTime),
			Fee:         ev.Fee,
			Capacity:    ev.Capacity,
			Description: ev.Description,
			ShowCreator: ev.ShowCreator,
		}

		// Fetch creator name if ShowCreator is true
		if ev.ShowCreator {
			profile, err := t.userProfileService.GetUserProfile(ctx, ev.CreatorID)
			if err != nil {
				t.logger.WarnContext(ctx, "failed to get user profile, hiding creator", slog.String("user_id", ev.CreatorID), slog.Any("error", err))
				eventData.ShowCreator = false
			} else {
				eventData.CreatorName = profile.DisplayName
			}
		}

		eventDataList[i] = eventData
	}

	// Render alt text template
	altTmpl, err := template.New("alt").Parse(altTemplate)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to parse alt template", slog.Any("error", err))
		return nil, errors.New("internal error")
	}

	var altBuf bytes.Buffer
	if err := altTmpl.Execute(&altBuf, map[string]int{"Count": len(events)}); err != nil {
		t.logger.ErrorContext(ctx, "failed to execute alt template", slog.Any("error", err))
		return nil, errors.New("internal error")
	}
	altText := altBuf.String()

	// Render flex template
	flexTmpl, err := template.New("flex").Parse(flexTemplate)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to parse flex template", slog.Any("error", err))
		return nil, errors.New("internal error")
	}

	var flexBuf bytes.Buffer
	if err := flexTmpl.Execute(&flexBuf, eventDataList); err != nil {
		t.logger.ErrorContext(ctx, "failed to execute flex template", slog.Any("error", err))
		return nil, errors.New("internal error")
	}
	flexJSON := flexBuf.Bytes()

	// Send flex message
	if err := t.lineClient.SendFlexReply(replyToken, altText, flexJSON); err != nil {
		t.logger.ErrorContext(ctx, "failed to send flex message", slog.Any("error", err))
		return nil, errors.New("failed to send flex message")
	}

	return map[string]any{
		"status": "sent",
	}, nil
}

// IsFinal returns true if the flex message was sent successfully.
// When status is "sent", the LLM turn should end.
// When status is "no_events", the LLM should continue with a follow-up response.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	status, ok := validatedResult["status"].(string)
	return ok && status == "sent"
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

// formatDisplayTime formats a time for display in flex message.
// Format: "2006/01/02 15:04" in JST.
func formatDisplayTime(t time.Time) string {
	return t.In(JST).Format("2006/01/02 15:04")
}
