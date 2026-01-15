package list_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/toolset/event/list"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// withEventContext creates a context with sourceID and userID set.
func withEventContext(ctx context.Context, sourceID, userID string) context.Context {
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	return ctx
}

// JST is Japan Standard Time location.
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// fixedNow is a fixed time for testing.
var fixedNow = time.Date(2026, 2, 15, 12, 0, 0, 0, JST)

// testEvent creates a test event with the given parameters.
func testEvent(chatRoomID, creatorID, title string, startTime, endTime time.Time) *event.Event {
	return &event.Event{
		ChatRoomID:  chatRoomID,
		CreatorID:   creatorID,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		Fee:         "1000円",
		Capacity:    10,
		Description: "Test event",
		ShowCreator: true,
	}
}

// parseTime parses RFC3339 time string or panics.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// New() Tests
// =============================================================================

func TestNew(t *testing.T) {
	// AC-XXX: Tool constructor validates dependencies and parameters
	t.Run("creates tool with valid parameters", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "list_events", tool.Name())
	})

	t.Run("returns error when eventService is nil", func(t *testing.T) {
		tool, err := list.New(nil, 365, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})

	t.Run("returns error when maxPeriodDays is zero", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, 0, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "maxPeriodDays must be positive")
	})

	t.Run("returns error when maxPeriodDays is negative", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, -1, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "maxPeriodDays must be positive")
	})

	t.Run("returns error when limit is zero", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, 365, 0, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("returns error when limit is negative", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, 365, -1, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := list.New(eventService, 365, 5, nil)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})
}

// =============================================================================
// Tool Interface Tests
// =============================================================================

func TestTool_Metadata(t *testing.T) {
	eventService := &mockEventService{}
	tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

	t.Run("Name returns list_events", func(t *testing.T) {
		assert.Equal(t, "list_events", tool.Name())
	})

	t.Run("Description is meaningful", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "event")
	})

	t.Run("ParametersJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ParametersJsonSchema()
		assert.NotEmpty(t, schema)
		// Verify it contains expected fields
		assert.Contains(t, string(schema), "created_by_me")
		assert.Contains(t, string(schema), "start")
		assert.Contains(t, string(schema), "end")
	})

	t.Run("ResponseJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ResponseJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "events")
		assert.Contains(t, string(schema), "chat_room_id")
		assert.Contains(t, string(schema), "title")
		assert.Contains(t, string(schema), "start_time")
		assert.Contains(t, string(schema), "end_time")
		assert.Contains(t, string(schema), "fee")
	})
}

// =============================================================================
// Callback Tests - Basic Listing
// =============================================================================

func TestTool_Callback_BasicListing(t *testing.T) {
	// AC-006: Event List Retrieval [FR-010, FR-014]
	t.Run("retrieves all events when no filters", func(t *testing.T) {
		// Events are returned by service already sorted (ascending by start time)
		// Event C (+12h) < Event A (+24h) < Event B (+48h)
		eventC := testEvent("group-3", "user-1", "Event C", fixedNow.Add(12*time.Hour), fixedNow.Add(14*time.Hour))
		eventA := testEvent("group-1", "user-1", "Event A", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))
		eventB := testEvent("group-2", "user-2", "Event B", fixedNow.Add(48*time.Hour), fixedNow.Add(50*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{eventC, eventA, eventB}, // Service returns sorted
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		events, ok := result["events"].([]map[string]any)
		require.True(t, ok)
		assert.Len(t, events, 3)

		// Verify order preserved (service handles sorting)
		assert.Equal(t, "Event C", events[0]["title"])
		assert.Equal(t, "Event A", events[1]["title"])
		assert.Equal(t, "Event B", events[2]["title"])

		// Verify service was called with correct options
		assert.Equal(t, 1, eventService.listCount)
		assert.Nil(t, eventService.lastOpts.CreatorID)
		assert.Nil(t, eventService.lastOpts.Start)
		assert.Nil(t, eventService.lastOpts.End)
		assert.Equal(t, 5, eventService.lastOpts.Limit) // Default limit
	})

	// FR-016: Response includes only specified fields
	t.Run("response includes only required fields", func(t *testing.T) {
		event1 := testEvent("group-1", "user-1", "Event A", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		events, ok := result["events"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, events, 1)

		ev := events[0]
		assert.Contains(t, ev, "chat_room_id")
		assert.Contains(t, ev, "title")
		assert.Contains(t, ev, "start_time")
		assert.Contains(t, ev, "end_time")
		assert.Contains(t, ev, "fee")

		// Verify excluded fields
		assert.NotContains(t, ev, "creator_id")
		assert.NotContains(t, ev, "capacity")
		assert.NotContains(t, ev, "description")
		assert.NotContains(t, ev, "show_creator")
	})

	t.Run("returns empty list when no events exist", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		events, ok := result["events"].([]map[string]any)
		require.True(t, ok)
		assert.Len(t, events, 0)
	})
}

// =============================================================================
// Callback Tests - Creator Filter
// =============================================================================

func TestTool_Callback_CreatorFilter(t *testing.T) {
	// AC-007: Creator Filter [FR-011]
	t.Run("filters to show only user's events when created_by_me is true", func(t *testing.T) {
		event1 := testEvent("group-1", "user-1", "My Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))
		// event2 would be filtered out by service (user-2 is not the current user)
		event3 := testEvent("group-3", "user-1", "My Other Event", fixedNow.Add(12*time.Hour), fixedNow.Add(14*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1, event3}, // Service already filtered
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": true,
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		events, ok := result["events"].([]map[string]any)
		require.True(t, ok)
		assert.Len(t, events, 2)

		// Verify service was called with CreatorID filter
		assert.Equal(t, 1, eventService.listCount)
		require.NotNil(t, eventService.lastOpts.CreatorID)
		assert.Equal(t, "user-1", *eventService.lastOpts.CreatorID)
	})

	t.Run("does not filter when created_by_me is false", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": false,
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called without CreatorID filter
		assert.Nil(t, eventService.lastOpts.CreatorID)
	})

	t.Run("returns error when created_by_me is not boolean", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": "yes",
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})

	t.Run("returns error when userID not in context and created_by_me is true", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := line.WithSourceID(context.Background(), "group-123")
		args := map[string]any{
			"created_by_me": true,
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})
}

// =============================================================================
// Callback Tests - Period Filter (before/after)
// =============================================================================

func TestTool_Callback_PeriodFilter_Before(t *testing.T) {
	// AC-008: Period Filter (before) [FR-012]
	t.Run("returns future events when start is 'today'", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called with Start set to today 00:00:00 JST
		require.NotNil(t, eventService.lastOpts.Start)
		assert.Nil(t, eventService.lastOpts.End)

		// Verify "today" was resolved to 00:00:00 JST
		start := *eventService.lastOpts.Start
		assert.Equal(t, 0, start.Hour())
		assert.Equal(t, 0, start.Minute())
		assert.Equal(t, 0, start.Second())
		assert.Equal(t, "Asia/Tokyo", start.Location().String())
	})

	t.Run("returns future events when start is RFC3339", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		startTime := "2026-03-01T00:00:00+09:00"
		args := map[string]any{
			"start": startTime,
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called with parsed start time
		require.NotNil(t, eventService.lastOpts.Start)
		assert.Equal(t, parseTime(startTime), *eventService.lastOpts.Start)
	})
}

func TestTool_Callback_PeriodFilter_After(t *testing.T) {
	// AC-009: Period Filter (after) [FR-012]
	t.Run("returns past events when end is 'today'", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called with End set to today 00:00:00 JST
		assert.Nil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)

		// Verify "today" was resolved to 00:00:00 JST
		end := *eventService.lastOpts.End
		assert.Equal(t, 0, end.Hour())
		assert.Equal(t, 0, end.Minute())
		assert.Equal(t, 0, end.Second())
		assert.Equal(t, "Asia/Tokyo", end.Location().String())
	})

	t.Run("returns past events when end is RFC3339", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		endTime := "2026-02-01T00:00:00+09:00"
		args := map[string]any{
			"end": endTime,
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called with parsed end time
		require.NotNil(t, eventService.lastOpts.End)
		assert.Equal(t, parseTime(endTime), *eventService.lastOpts.End)
	})
}

// =============================================================================
// Callback Tests - Period Filter (range)
// =============================================================================

func TestTool_Callback_PeriodFilter_Range(t *testing.T) {
	// AC-010: Period Filter (range) [FR-012]
	t.Run("filters events within specified date range", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		startTime := "2026-03-01T00:00:00+09:00"
		endTime := "2026-03-31T23:59:59+09:00"
		args := map[string]any{
			"start": startTime,
			"end":   endTime,
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify service was called with both Start and End
		require.NotNil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)
		assert.Equal(t, parseTime(startTime), *eventService.lastOpts.Start)
		assert.Equal(t, parseTime(endTime), *eventService.lastOpts.End)
		assert.Equal(t, 0, eventService.lastOpts.Limit) // No limit when both specified
	})

	// FR-012: Period validation (max 1 year when both specified)
	t.Run("returns error when range exceeds maxPeriodDays", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "2026-01-01T00:00:00+09:00",
			"end":   "2027-01-02T00:00:00+09:00", // 367 days > 365
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})

	t.Run("allows range equal to maxPeriodDays", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "2026-01-01T00:00:00+09:00",
			"end":   "2027-01-01T00:00:00+09:00", // Exactly 365 days
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 1, eventService.listCount)
	})

	t.Run("returns error when end is before start", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "2026-03-01T00:00:00+09:00",
			"end":   "2026-02-01T00:00:00+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})
}

// =============================================================================
// Callback Tests - Combined Filters
// =============================================================================

func TestTool_Callback_CombinedFilters(t *testing.T) {
	// AC-011: Filter Combination [FR-013]
	t.Run("applies both creator and period filters", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": true,
			"start":         "2026-03-01T00:00:00+09:00",
			"end":           "2026-03-31T23:59:59+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify both filters applied
		require.NotNil(t, eventService.lastOpts.CreatorID)
		assert.Equal(t, "user-1", *eventService.lastOpts.CreatorID)
		require.NotNil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)
	})

	t.Run("applies creator filter with start only", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": true,
			"start":         "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify creator filter and start filter
		require.NotNil(t, eventService.lastOpts.CreatorID)
		require.NotNil(t, eventService.lastOpts.Start)
		assert.Nil(t, eventService.lastOpts.End)
		assert.Equal(t, 5, eventService.lastOpts.Limit) // Limit applied
	})

	t.Run("applies creator filter with end only", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"created_by_me": true,
			"end":           "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify creator filter and end filter
		require.NotNil(t, eventService.lastOpts.CreatorID)
		assert.Nil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)
		assert.Equal(t, 5, eventService.lastOpts.Limit) // Limit applied
	})
}

// =============================================================================
// Callback Tests - Sort Order
// =============================================================================

func TestTool_Callback_SortOrder(t *testing.T) {
	// AC-006 & FR-014: Sort order behavior
	t.Run("sorts ascending when start only specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Service handles sorting, just verify options passed correctly
		require.NotNil(t, eventService.lastOpts.Start)
		assert.Nil(t, eventService.lastOpts.End)
	})

	t.Run("sorts ascending when both start and end specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "2026-03-01T00:00:00+09:00",
			"end":   "2026-03-31T23:59:59+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Service handles sorting, just verify options passed correctly
		require.NotNil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)
	})

	t.Run("sorts descending when end only specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Service handles sorting, just verify options passed correctly
		assert.Nil(t, eventService.lastOpts.Start)
		require.NotNil(t, eventService.lastOpts.End)
	})
}

// =============================================================================
// Callback Tests - Limit Enforcement
// =============================================================================

func TestTool_Callback_LimitEnforcement(t *testing.T) {
	// FR-015: Limit enforcement rules
	t.Run("applies limit when start only specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 5, eventService.lastOpts.Limit)
	})

	t.Run("applies limit when end only specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 5, eventService.lastOpts.Limit)
	})

	t.Run("does not apply limit when both start and end specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "2026-03-01T00:00:00+09:00",
			"end":   "2026-03-31T23:59:59+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 0, eventService.lastOpts.Limit) // No limit
	})

	t.Run("applies limit when no filters specified", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 5, eventService.lastOpts.Limit)
	})
}

// =============================================================================
// Callback Tests - Time Format
// =============================================================================

func TestTool_Callback_TimeFormat(t *testing.T) {
	// FR-016: Times in JST RFC3339 format
	t.Run("formats times in JST RFC3339", func(t *testing.T) {
		startTime := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
		endTime := time.Date(2026, 2, 15, 16, 30, 0, 0, time.UTC)

		event1 := testEvent("group-1", "user-1", "Event A", startTime, endTime)

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		events, ok := result["events"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, events, 1)

		// Verify times are in JST RFC3339 format
		startTimeStr, ok := events[0]["start_time"].(string)
		require.True(t, ok)
		endTimeStr, ok := events[0]["end_time"].(string)
		require.True(t, ok)

		// Verify times are parseable as RFC3339
		_, err = time.Parse(time.RFC3339, startTimeStr)
		require.NoError(t, err)
		_, err = time.Parse(time.RFC3339, endTimeStr)
		require.NoError(t, err)

		// Verify times are in JST (UTC+9)
		expectedStart := startTime.In(JST)
		expectedEnd := endTime.In(JST)
		assert.Equal(t, expectedStart.Format(time.RFC3339), startTimeStr)
		assert.Equal(t, expectedEnd.Format(time.RFC3339), endTimeStr)
	})
}

// =============================================================================
// Callback Tests - Today Resolution
// =============================================================================

func TestTool_Callback_TodayResolution(t *testing.T) {
	// FR-012: "today" resolves to current date 00:00:00 JST
	t.Run("resolves 'today' to current date 00:00:00 JST", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify "today" was resolved to 00:00:00 JST
		require.NotNil(t, eventService.lastOpts.Start)
		start := *eventService.lastOpts.Start
		assert.Equal(t, 0, start.Hour())
		assert.Equal(t, 0, start.Minute())
		assert.Equal(t, 0, start.Second())
		assert.Equal(t, 0, start.Nanosecond())
		assert.Equal(t, "Asia/Tokyo", start.Location().String())
	})

	t.Run("resolves 'today' for end parameter", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": "today",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify "today" was resolved to 00:00:00 JST
		require.NotNil(t, eventService.lastOpts.End)
		end := *eventService.lastOpts.End
		assert.Equal(t, 0, end.Hour())
		assert.Equal(t, 0, end.Minute())
		assert.Equal(t, 0, end.Second())
		assert.Equal(t, 0, end.Nanosecond())
		assert.Equal(t, "Asia/Tokyo", end.Location().String())
	})
}

// =============================================================================
// Callback Tests - Error Cases
// =============================================================================

func TestTool_Callback_Errors(t *testing.T) {
	t.Run("returns error when event service fails", func(t *testing.T) {
		eventService := &mockEventService{
			listErr: errors.New("storage error"),
		}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Equal(t, 1, eventService.listCount)
	})

	t.Run("returns error when start is invalid RFC3339", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": "not-a-date",
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})

	t.Run("returns error when end is invalid RFC3339", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": "2026-13-40T25:61:61+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})

	t.Run("returns error when start is not a string", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"start": 123,
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})

	t.Run("returns error when end is not a string", func(t *testing.T) {
		eventService := &mockEventService{}
		tool, _ := list.New(eventService, 365, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1")
		args := map[string]any{
			"end": true,
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)

		// Service should not be called
		assert.Equal(t, 0, eventService.listCount)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockEventService struct {
	listEvents []*event.Event
	listErr    error
	listCount  int
	lastOpts   event.ListOptions
}

func (m *mockEventService) List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error) {
	m.listCount++
	m.lastOpts = opts
	return m.listEvents, m.listErr
}
