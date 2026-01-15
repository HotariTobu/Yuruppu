package get_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/profile"
	"yuruppu/internal/toolset/event/get"

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

// validEvent returns a valid event for testing.
func validEvent() *event.Event {
	now := time.Now()
	return &event.Event{
		ChatRoomID:  "group-123",
		CreatorID:   "user-456",
		Title:       "Team Meeting",
		StartTime:   now.Add(24 * time.Hour),
		EndTime:     now.Add(26 * time.Hour),
		Capacity:    10,
		Fee:         "Free",
		Description: "Monthly team sync",
		ShowCreator: true,
	}
}

// =============================================================================
// New() Tests
// =============================================================================

func TestNew(t *testing.T) {
	// AC-XXX: Tool constructor validates dependencies
	t.Run("creates tool with valid services", func(t *testing.T) {
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		tool, err := get.New(eventService, profileService)

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "get_event", tool.Name())
	})

	t.Run("returns error when eventService is nil", func(t *testing.T) {
		profileService := &mockProfileService{}

		tool, err := get.New(nil, profileService)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})

	t.Run("returns error when profileService is nil", func(t *testing.T) {
		eventService := &mockEventService{}

		tool, err := get.New(eventService, nil)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "profileService cannot be nil")
	})
}

// =============================================================================
// Tool Interface Tests
// =============================================================================

func TestTool_Metadata(t *testing.T) {
	eventService := &mockEventService{}
	profileService := &mockProfileService{}
	tool, _ := get.New(eventService, profileService)

	t.Run("Name returns get_event", func(t *testing.T) {
		assert.Equal(t, "get_event", tool.Name())
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
		assert.Contains(t, string(schema), "chat_room_id")
	})

	t.Run("ResponseJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ResponseJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "success")
		assert.Contains(t, string(schema), "creator_name")
		assert.Contains(t, string(schema), "title")
		assert.Contains(t, string(schema), "start_time")
		assert.Contains(t, string(schema), "end_time")
		assert.Contains(t, string(schema), "fee")
		assert.Contains(t, string(schema), "capacity")
		assert.Contains(t, string(schema), "description")
		assert.Contains(t, string(schema), "error")
	})
}

// =============================================================================
// Callback Tests - Success Cases
// =============================================================================

func TestTool_Callback_Success(t *testing.T) {
	// AC-004: Event Detail Retrieval [FR-009]
	t.Run("retrieves event by explicit chat_room_id", func(t *testing.T) {
		ev := validEvent()
		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{
			displayName: "Alice",
		}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-999", "user-789")
		args := map[string]any{
			"chat_room_id": "group-123",
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "Alice", result["creator_name"])
		assert.Equal(t, "Team Meeting", result["title"])
		assert.NotEmpty(t, result["start_time"])
		assert.NotEmpty(t, result["end_time"])
		assert.Equal(t, "Free", result["fee"])
		assert.Equal(t, 10, result["capacity"])
		assert.Equal(t, "Monthly team sync", result["description"])
		assert.NotContains(t, result, "error")

		// Verify service was called with correct chatRoomID
		assert.Equal(t, 1, eventService.getCount)
		assert.Equal(t, "group-123", eventService.lastChatRoomID)

		// Verify profile service was called
		assert.Equal(t, 1, profileService.getCount)
		assert.Equal(t, "user-456", profileService.lastUserID)
	})

	// AC-004: Event Detail Retrieval using implicit sourceID from context
	t.Run("retrieves event using implicit sourceID when chat_room_id not provided", func(t *testing.T) {
		ev := validEvent()
		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{
			displayName: "Bob",
		}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "Bob", result["creator_name"])
		assert.Equal(t, "Team Meeting", result["title"])

		// Verify sourceID was used
		assert.Equal(t, "group-123", eventService.lastChatRoomID)
	})

	// AC-012: Creator Public [NFR-002]
	t.Run("displays creator name when showCreator is true", func(t *testing.T) {
		ev := validEvent()
		ev.ShowCreator = true
		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{
			displayName: "Charlie",
		}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "Charlie", result["creator_name"])

		// Verify profile service was called
		assert.Equal(t, 1, profileService.getCount)
		assert.Equal(t, "user-456", profileService.lastUserID)
	})

	// AC-013: Creator Private [NFR-002]
	t.Run("does not display creator name when showCreator is false", func(t *testing.T) {
		ev := validEvent()
		ev.ShowCreator = false
		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{
			displayName: "Should Not Appear",
		}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])
		assert.NotContains(t, result, "creator_name")

		// Verify profile service was NOT called
		assert.Equal(t, 0, profileService.getCount)
	})

	// AC-004: Test time formatting in JST RFC3339
	t.Run("formats times in JST RFC3339", func(t *testing.T) {
		startTime := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)
		endTime := time.Date(2026, 2, 15, 16, 30, 0, 0, time.UTC)

		ev := validEvent()
		ev.StartTime = startTime
		ev.EndTime = endTime
		ev.ShowCreator = false

		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])

		// Verify times are in JST RFC3339 format
		startTimeStr, ok := result["start_time"].(string)
		require.True(t, ok)
		endTimeStr, ok := result["end_time"].(string)
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
// Callback Tests - Error Cases
// =============================================================================

func TestTool_Callback_Errors(t *testing.T) {
	// AC-005: Event Detail Retrieval (Not Found) [FR-009]
	t.Run("returns error when event not found", func(t *testing.T) {
		eventService := &mockEventService{
			getErr: errors.New("event not found: group-404"),
		}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-404", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "not found")
		assert.NotContains(t, result, "title")
		assert.NotContains(t, result, "creator_name")

		// Service should be called
		assert.Equal(t, 1, eventService.getCount)
		// Profile service should not be called
		assert.Equal(t, 0, profileService.getCount)
	})

	t.Run("returns error when event service fails", func(t *testing.T) {
		eventService := &mockEventService{
			getErr: errors.New("storage error"),
		}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "storage error")

		assert.Equal(t, 1, eventService.getCount)
		assert.Equal(t, 0, profileService.getCount)
	})

	t.Run("returns error when profile service fails and showCreator is true", func(t *testing.T) {
		ev := validEvent()
		ev.ShowCreator = true
		eventService := &mockEventService{
			getEvent: ev,
		}
		profileService := &mockProfileService{
			getErr: errors.New("profile not found"),
		}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-789")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "profile not found")

		assert.Equal(t, 1, eventService.getCount)
		assert.Equal(t, 1, profileService.getCount)
	})

	t.Run("returns error when sourceID not in context and chat_room_id not provided", func(t *testing.T) {
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		// Only set userID, not sourceID
		ctx := line.WithUserID(context.Background(), "user-123")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "internal error")

		// Service should not be called
		assert.Equal(t, 0, eventService.getCount)
		assert.Equal(t, 0, profileService.getCount)
	})
}

// =============================================================================
// Callback Tests - Validation Errors
// =============================================================================

func TestTool_Callback_ValidationErrors(t *testing.T) {
	t.Run("returns error when chat_room_id is empty string", func(t *testing.T) {
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{
			"chat_room_id": "",
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "chat_room_id")

		// Service should not be called
		assert.Equal(t, 0, eventService.getCount)
	})

	t.Run("returns error when chat_room_id is not a string", func(t *testing.T) {
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		tool, _ := get.New(eventService, profileService)

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{
			"chat_room_id": 123,
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "chat_room_id")

		// Service should not be called
		assert.Equal(t, 0, eventService.getCount)
	})
}

// =============================================================================
// IsFinal Tests
// =============================================================================

func TestTool_IsFinal(t *testing.T) {
	eventService := &mockEventService{}
	profileService := &mockProfileService{}
	tool, _ := get.New(eventService, profileService)

	t.Run("returns true when success is true", func(t *testing.T) {
		result := map[string]any{
			"success":      true,
			"title":        "Team Meeting",
			"creator_name": "Alice",
		}
		assert.True(t, tool.IsFinal(result))
	})

	t.Run("returns false when success is false", func(t *testing.T) {
		result := map[string]any{
			"success": false,
			"error":   "some error",
		}
		assert.False(t, tool.IsFinal(result))
	})

	t.Run("returns false when success is missing", func(t *testing.T) {
		result := map[string]any{
			"title": "Team Meeting",
		}
		assert.False(t, tool.IsFinal(result))
	})

	t.Run("returns false when success is not boolean", func(t *testing.T) {
		result := map[string]any{
			"success": "true",
		}
		assert.False(t, tool.IsFinal(result))
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockEventService struct {
	getEvent       *event.Event
	getErr         error
	getCount       int
	lastChatRoomID string
}

func (m *mockEventService) Get(ctx context.Context, chatRoomID string) (*event.Event, error) {
	m.getCount++
	m.lastChatRoomID = chatRoomID
	return m.getEvent, m.getErr
}

type mockProfileService struct {
	displayName string
	getErr      error
	getCount    int
	lastUserID  string
}

func (m *mockProfileService) GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error) {
	m.getCount++
	m.lastUserID = userID
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &profile.UserProfile{DisplayName: m.displayName}, nil
}
