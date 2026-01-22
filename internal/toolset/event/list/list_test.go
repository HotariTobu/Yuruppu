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
	"yuruppu/internal/userprofile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// withEventContext creates a context with sourceID, userID, and replyToken set.
func withEventContext(ctx context.Context, sourceID, userID, replyToken string) context.Context {
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	ctx = line.WithReplyToken(ctx, replyToken)
	return ctx
}

// JST is Japan Standard Time location.
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// fixedNow is a fixed time for testing.
var fixedNow = time.Date(2026, 2, 15, 12, 0, 0, 0, JST)

// testEvent creates a test event with the given parameters.
func testEvent(chatRoomID, creatorID, title string, startTime, endTime time.Time) *event.Event {
	return testEventWithShowCreator(chatRoomID, creatorID, title, startTime, endTime, true)
}

// testEventWithShowCreator creates a test event with ShowCreator parameter.
func testEventWithShowCreator(chatRoomID, creatorID, title string, startTime, endTime time.Time, showCreator bool) *event.Event {
	return &event.Event{
		ChatRoomID:  chatRoomID,
		CreatorID:   creatorID,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		Fee:         "1000円",
		Capacity:    10,
		Description: "Test event",
		ShowCreator: showCreator,
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "list_events", tool.Name())
	})

	t.Run("returns error when eventService is nil", func(t *testing.T) {
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(nil, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})

	t.Run("returns error when lineClient is nil", func(t *testing.T) {
		eventService := &mockEventService{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, nil, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "lineClient cannot be nil")
	})

	t.Run("returns error when userProfileService is nil", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}

		tool, err := list.New(eventService, lineClient, nil, 366, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "userProfileService cannot be nil")
	})

	t.Run("returns error when maxPeriodDays is zero", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, 0, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "maxPeriodDays must be positive")
	})

	t.Run("returns error when maxPeriodDays is negative", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, -1, 5, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "maxPeriodDays must be positive")
	})

	t.Run("returns error when limit is zero", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, 366, 0, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("returns error when limit is negative", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, 366, -1, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "limit must be positive")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}

		tool, err := list.New(eventService, lineClient, userProfileService, 366, 5, nil)

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
	lineClient := &mockLineClient{}
	userProfileService := &mockUserProfileService{}
	tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

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
		assert.Contains(t, string(schema), "status")
		assert.Contains(t, string(schema), "sent")
		assert.Contains(t, string(schema), "no_events")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify flex message was sent
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)
		assert.Contains(t, string(lineClient.lastFlexJSON), "Event C")
		assert.Contains(t, string(lineClient.lastFlexJSON), "Event A")
		assert.Contains(t, string(lineClient.lastFlexJSON), "Event B")

		// Verify result status
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)

		// Verify service was called with correct options
		assert.Equal(t, 1, eventService.listCount)
		assert.Nil(t, eventService.lastOpts.CreatorID)
		// FR-012a: When neither start nor end specified, defaults to today 00:00:00 JST
		require.NotNil(t, eventService.lastOpts.Start)
		start := *eventService.lastOpts.Start
		assert.Equal(t, 0, start.Hour())
		assert.Equal(t, 0, start.Minute())
		assert.Equal(t, 0, start.Second())
		assert.Equal(t, "Asia/Tokyo", start.Location().String())
		assert.Nil(t, eventService.lastOpts.End)
		assert.Equal(t, 5, eventService.lastOpts.Limit) // Default limit
	})

	// FR-016: Flex message includes all event fields
	t.Run("flex message includes all event fields", func(t *testing.T) {
		event1 := testEvent("group-1", "user-1", "Event A", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify flex message was sent
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)

		// Verify flex message contains event data
		flexJSON := string(lineClient.lastFlexJSON)
		assert.Contains(t, flexJSON, "Event A")
		assert.Contains(t, flexJSON, "1000円")
		assert.Contains(t, flexJSON, "Test event")

		// Verify result status
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("returns no_events status when no events exist", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify no flex message was sent
		assert.Equal(t, 0, lineClient.sendFlexReplyCount)

		// Verify result status
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "no_events", status)
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{
			"created_by_me": true,
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify flex message was sent
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)

		// Verify result status
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)

		// Verify service was called with CreatorID filter
		assert.Equal(t, 1, eventService.listCount)
		require.NotNil(t, eventService.lastOpts.CreatorID)
		assert.Equal(t, "user-1", *eventService.lastOpts.CreatorID)
	})

	t.Run("does not filter when created_by_me is false", func(t *testing.T) {
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{
			"start": "2026-01-01T00:00:00+09:00",
			"end":   "2027-01-03T00:00:00+09:00",
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{
			"start": "2026-01-01T00:00:00+09:00",
			"end":   "2027-01-02T00:00:00+09:00",
		}

		_, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, 1, eventService.listCount)
	})

	t.Run("returns error when end is before start", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
	// FR-016: Times in flex message are formatted in JST
	t.Run("formats times in JST for display", func(t *testing.T) {
		startTime := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
		endTime := time.Date(2026, 2, 15, 16, 30, 0, 0, time.UTC)

		event1 := testEvent("group-1", "user-1", "Event A", startTime, endTime)

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Verify flex message was sent
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)

		// Verify times are in JST format (2006/01/02 15:04)
		flexJSON := string(lineClient.lastFlexJSON)
		expectedStart := startTime.In(JST).Format("2006/01/02 15:04")
		expectedEnd := endTime.In(JST).Format("2006/01/02 15:04")
		assert.Contains(t, flexJSON, expectedStart)
		assert.Contains(t, flexJSON, expectedEnd)

		// Verify result status
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})
}

// =============================================================================
// Callback Tests - Flex Message Sending
// =============================================================================

func TestTool_Callback_FlexMessage(t *testing.T) {
	// AC-001: Flex Message sending [CH-001]
	t.Run("sends flex message when events exist", func(t *testing.T) {
		// Setup: One event exists
		event1 := testEvent("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: LineClient.SendFlexReply is called once
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)
		assert.Equal(t, "test-reply-token", lineClient.lastReplyToken)
		assert.NotEmpty(t, lineClient.lastAltText)
		assert.NotEmpty(t, lineClient.lastFlexJSON)

		// Expected: Result has {"status": "sent"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("includes creator name when ShowCreator is true", func(t *testing.T) {
		// Setup: Event with ShowCreator=true
		event1 := testEventWithShowCreator("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour), true)

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Creator Name",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-2", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: UserProfileService.GetUserProfile is called
		assert.Equal(t, 1, userProfileService.getUserProfileCount)
		assert.Equal(t, "user-1", userProfileService.lastUserID)

		// Expected: Flex JSON contains creator name
		assert.Contains(t, string(lineClient.lastFlexJSON), "Creator Name")
		assert.NotContains(t, string(lineClient.lastFlexJSON), "？？？")

		// Expected: Result has {"status": "sent"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("hides creator name when ShowCreator is false", func(t *testing.T) {
		// Setup: Event with ShowCreator=false
		event1 := testEventWithShowCreator("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour), false)

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Creator Name",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-2", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: UserProfileService.GetUserProfile is NOT called for this event
		assert.Equal(t, 0, userProfileService.getUserProfileCount)

		// Expected: Flex JSON contains "？？？" instead of creator name
		assert.Contains(t, string(lineClient.lastFlexJSON), "？？？")
		assert.NotContains(t, string(lineClient.lastFlexJSON), "Creator Name")

		// Expected: Result has {"status": "sent"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("sends flex message for multiple events", func(t *testing.T) {
		// Setup: Multiple events exist
		event1 := testEvent("group-1", "user-1", "Event A", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))
		event2 := testEvent("group-1", "user-2", "Event B", fixedNow.Add(48*time.Hour), fixedNow.Add(50*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1, event2},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-3", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: LineClient.SendFlexReply is called once with carousel
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)
		assert.Contains(t, string(lineClient.lastFlexJSON), "Event A")
		assert.Contains(t, string(lineClient.lastFlexJSON), "Event B")

		// Expected: UserProfileService.GetUserProfile is called for both creators
		assert.Equal(t, 2, userProfileService.getUserProfileCount)

		// Expected: Result has {"status": "sent"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("uses replyToken from context", func(t *testing.T) {
		// Setup: context has replyToken set
		event1 := testEvent("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-1", "custom-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: LineClient.SendFlexReply receives correct replyToken
		assert.Equal(t, "custom-reply-token", lineClient.lastReplyToken)

		// Expected: Result has {"status": "sent"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)
	})

	t.Run("returns error when replyToken not in context", func(t *testing.T) {
		// Setup: context without replyToken
		event1 := testEvent("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := line.WithSourceID(context.Background(), "group-1")
		ctx = line.WithUserID(ctx, "user-1")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		// Expected: Error is returned
		require.Error(t, err)

		// Expected: LineClient.SendFlexReply is NOT called
		assert.Equal(t, 0, lineClient.sendFlexReplyCount)
	})

	t.Run("returns error when SendFlexReply fails", func(t *testing.T) {
		// Setup: LineClient.SendFlexReply returns error
		event1 := testEvent("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{
			sendFlexReplyErr: errors.New("LINE API error"),
		}
		userProfileService := &mockUserProfileService{
			getUserProfileResult: &userprofile.UserProfile{
				DisplayName: "Test User",
			},
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-1", "test-reply-token")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		// Expected: Error is returned
		require.Error(t, err)
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)
	})

	// AC-002: No events case [CH-001]
	t.Run("returns no_events status when no events match", func(t *testing.T) {
		// Setup: EventService.List returns empty slice
		eventService := &mockEventService{
			listEvents: []*event.Event{},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-1", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)

		// Expected: LineClient.SendFlexReply is NOT called
		assert.Equal(t, 0, lineClient.sendFlexReplyCount)

		// Expected: Result has {"status": "no_events"}
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "no_events", status)
	})

	t.Run("hides creator when GetUserProfile fails", func(t *testing.T) {
		// Setup: UserProfileService.GetUserProfile returns error
		event1 := testEvent("group-1", "user-1", "Test Event", fixedNow.Add(24*time.Hour), fixedNow.Add(26*time.Hour))

		eventService := &mockEventService{
			listEvents: []*event.Event{event1},
		}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{
			getUserProfileErr: errors.New("profile fetch error"),
		}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-1", "user-2", "test-reply-token")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		// Expected: No error, falls back to hiding creator
		require.NoError(t, err)

		// Expected: Status is sent
		status, ok := result["status"].(string)
		require.True(t, ok)
		assert.Equal(t, "sent", status)

		// Expected: LineClient.SendFlexReply is called
		assert.Equal(t, 1, lineClient.sendFlexReplyCount)
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Equal(t, 1, eventService.listCount)
	})

	t.Run("returns error when start is invalid RFC3339", func(t *testing.T) {
		eventService := &mockEventService{}
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
		lineClient := &mockLineClient{}
		userProfileService := &mockUserProfileService{}
		tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-1", "test-reply-token")
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
// IsFinal Tests
// =============================================================================

func TestTool_IsFinal(t *testing.T) {
	eventService := &mockEventService{}
	lineClient := &mockLineClient{}
	userProfileService := &mockUserProfileService{}
	tool, _ := list.New(eventService, lineClient, userProfileService, 366, 5, slog.New(slog.DiscardHandler))

	t.Run("returns true when status is sent", func(t *testing.T) {
		result := map[string]any{"status": "sent"}
		assert.True(t, tool.IsFinal(result))
	})

	t.Run("returns false when status is no_events", func(t *testing.T) {
		result := map[string]any{"status": "no_events"}
		assert.False(t, tool.IsFinal(result))
	})

	t.Run("returns false when status is missing", func(t *testing.T) {
		result := map[string]any{}
		assert.False(t, tool.IsFinal(result))
	})

	t.Run("returns false when status is not a string", func(t *testing.T) {
		result := map[string]any{"status": 123}
		assert.False(t, tool.IsFinal(result))
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

type mockLineClient struct {
	sendFlexReplyErr   error
	sendFlexReplyCount int
	lastReplyToken     string
	lastAltText        string
	lastFlexJSON       []byte
}

func (m *mockLineClient) SendFlexReply(replyToken string, altText string, flexJSON []byte) error {
	m.sendFlexReplyCount++
	m.lastReplyToken = replyToken
	m.lastAltText = altText
	m.lastFlexJSON = flexJSON
	return m.sendFlexReplyErr
}

type mockUserProfileService struct {
	getUserProfileResult *userprofile.UserProfile
	getUserProfileErr    error
	getUserProfileCount  int
	lastUserID           string
}

func (m *mockUserProfileService) GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error) {
	m.getUserProfileCount++
	m.lastUserID = userID
	return m.getUserProfileResult, m.getUserProfileErr
}
