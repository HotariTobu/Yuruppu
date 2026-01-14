package create_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/toolset/event/create"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// withEventContext creates a context with sourceID and userID set.
// For group chats: sourceID != userID
// For 1:1 chats: sourceID == userID
func withEventContext(ctx context.Context, sourceID, userID string) context.Context {
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	return ctx
}

// validEventArgs returns valid arguments for creating an event.
func validEventArgs() map[string]any {
	now := time.Now()
	return map[string]any{
		"title":        "Team Meeting",
		"start_time":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"end_time":     now.Add(26 * time.Hour).Format(time.RFC3339),
		"capacity":     10,
		"fee":          "Free",
		"description":  "Monthly team sync",
		"show_creator": true,
	}
}

// =============================================================================
// New() Tests
// =============================================================================

func TestNew(t *testing.T) {
	// AC-XXX: Tool constructor validates dependencies
	t.Run("creates tool with valid service", func(t *testing.T) {
		service := &mockEventService{}

		tool, err := create.New(service)

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "create_event", tool.Name())
	})

	t.Run("returns error when service is nil", func(t *testing.T) {
		tool, err := create.New(nil)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})
}

// =============================================================================
// Tool Interface Tests
// =============================================================================

func TestTool_Metadata(t *testing.T) {
	service := &mockEventService{}
	tool, _ := create.New(service)

	t.Run("Name returns create_event", func(t *testing.T) {
		assert.Equal(t, "create_event", tool.Name())
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
		assert.Contains(t, string(schema), "title")
		assert.Contains(t, string(schema), "start_time")
		assert.Contains(t, string(schema), "end_time")
		assert.Contains(t, string(schema), "capacity")
		assert.Contains(t, string(schema), "fee")
		assert.Contains(t, string(schema), "description")
		assert.Contains(t, string(schema), "show_creator")
	})

	t.Run("ResponseJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ResponseJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "success")
		assert.Contains(t, string(schema), "chat_room_id")
		assert.Contains(t, string(schema), "error")
	})
}

// =============================================================================
// Callback Tests - Success Cases
// =============================================================================

func TestTool_Callback_Success(t *testing.T) {
	// AC-XXX: Users can create events from group chats
	// FR-005: Chat room is automatically linked from the group chat
	// FR-006: Creator is automatically taken from user info
	t.Run("creates event with valid args from group chat", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := create.New(service)

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "group-123", result["chat_room_id"])
		assert.NotContains(t, result, "error")

		// Verify service was called with correct event
		require.Equal(t, 1, service.createCount)
		createdEvent := service.lastCreatedEvent
		assert.Equal(t, "group-123", createdEvent.ChatRoomID)
		assert.Equal(t, "user-456", createdEvent.CreatorID)
		assert.Equal(t, "Team Meeting", createdEvent.Title)
		assert.Equal(t, 10, createdEvent.Capacity)
		assert.Equal(t, "Free", createdEvent.Fee)
		assert.Equal(t, "Monthly team sync", createdEvent.Description)
		assert.Equal(t, true, createdEvent.ShowCreator)
	})

	// FR-008: Test all event attributes are correctly set
	t.Run("sets all event attributes correctly", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := create.New(service)

		ctx := withEventContext(context.Background(), "group-999", "user-888")
		now := time.Now()
		startTime := now.Add(48 * time.Hour)
		endTime := now.Add(50 * time.Hour)

		args := map[string]any{
			"title":        "Conference",
			"start_time":   startTime.Format(time.RFC3339),
			"end_time":     endTime.Format(time.RFC3339),
			"capacity":     100,
			"fee":          "5000 yen",
			"description":  "Annual tech conference",
			"show_creator": false,
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, true, result["success"])

		ev := service.lastCreatedEvent
		assert.Equal(t, "group-999", ev.ChatRoomID)
		assert.Equal(t, "user-888", ev.CreatorID)
		assert.Equal(t, "Conference", ev.Title)
		assert.WithinDuration(t, startTime, ev.StartTime, time.Second)
		assert.WithinDuration(t, endTime, ev.EndTime, time.Second)
		assert.Equal(t, 100, ev.Capacity)
		assert.Equal(t, "5000 yen", ev.Fee)
		assert.Equal(t, "Annual tech conference", ev.Description)
		assert.Equal(t, false, ev.ShowCreator)
	})
}

// =============================================================================
// Callback Tests - Context Errors
// =============================================================================

func TestTool_Callback_ContextErrors(t *testing.T) {
	// FR-003: Users can only create events from group chats
	t.Run("returns error when called from 1:1 chat (sourceID == userID)", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := create.New(service)

		// 1:1 chat: sourceID == userID
		ctx := withEventContext(context.Background(), "user-123", "user-123")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "group chat")
		assert.NotContains(t, result, "chat_room_id")

		// Service should not be called
		assert.Equal(t, 0, service.createCount)
	})

	t.Run("returns error when sourceID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := create.New(service)

		// Only set userID, not sourceID
		ctx := line.WithUserID(context.Background(), "user-123")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "internal error")
		assert.Equal(t, 0, service.createCount)
	})

	t.Run("returns error when userID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := create.New(service)

		// Only set sourceID, not userID
		ctx := line.WithSourceID(context.Background(), "group-123")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "internal error")
		assert.Equal(t, 0, service.createCount)
	})
}

// =============================================================================
// Callback Tests - Validation Errors
// =============================================================================

func TestTool_Callback_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		modifyArgs  func(map[string]any)
		expectError string
	}{
		{
			// FR-002: Missing required fields should return error
			name: "missing title",
			modifyArgs: func(args map[string]any) {
				delete(args, "title")
			},
			expectError: "title",
		},
		{
			// FR-002: Missing required fields should return error
			name: "missing start_time",
			modifyArgs: func(args map[string]any) {
				delete(args, "start_time")
			},
			expectError: "start_time",
		},
		{
			// FR-002: Missing required fields should return error
			name: "missing end_time",
			modifyArgs: func(args map[string]any) {
				delete(args, "end_time")
			},
			expectError: "end_time",
		},
		{
			name: "empty title",
			modifyArgs: func(args map[string]any) {
				args["title"] = ""
			},
			expectError: "title",
		},
		{
			// FR-008: startTime must be in the future
			name: "start_time in past",
			modifyArgs: func(args map[string]any) {
				past := time.Now().Add(-24 * time.Hour)
				args["start_time"] = past.Format(time.RFC3339)
			},
			expectError: "past",
		},
		{
			// FR-008: endTime must be after startTime
			name: "end_time before start_time",
			modifyArgs: func(args map[string]any) {
				now := time.Now()
				args["start_time"] = now.Add(26 * time.Hour).Format(time.RFC3339)
				args["end_time"] = now.Add(24 * time.Hour).Format(time.RFC3339)
			},
			expectError: "end_time",
		},
		{
			// FR-008: endTime must be after startTime
			name: "end_time equals start_time",
			modifyArgs: func(args map[string]any) {
				now := time.Now().Add(24 * time.Hour)
				args["start_time"] = now.Format(time.RFC3339)
				args["end_time"] = now.Format(time.RFC3339)
			},
			expectError: "end_time",
		},
		{
			name: "invalid start_time format",
			modifyArgs: func(args map[string]any) {
				args["start_time"] = "not-a-date"
			},
			expectError: "start_time",
		},
		{
			name: "invalid end_time format",
			modifyArgs: func(args map[string]any) {
				args["end_time"] = "not-a-date"
			},
			expectError: "end_time",
		},
		{
			name: "negative capacity",
			modifyArgs: func(args map[string]any) {
				args["capacity"] = -1
			},
			expectError: "capacity",
		},
		{
			name: "zero capacity",
			modifyArgs: func(args map[string]any) {
				args["capacity"] = 0
			},
			expectError: "capacity",
		},
		{
			// FR-002, FR-008: Missing required fields should return error
			name: "missing fee",
			modifyArgs: func(args map[string]any) {
				delete(args, "fee")
			},
			expectError: "fee",
		},
		{
			name: "empty fee",
			modifyArgs: func(args map[string]any) {
				args["fee"] = ""
			},
			expectError: "fee",
		},
		{
			// FR-002, FR-008: Missing required fields should return error
			name: "missing description",
			modifyArgs: func(args map[string]any) {
				delete(args, "description")
			},
			expectError: "description",
		},
		{
			name: "empty description",
			modifyArgs: func(args map[string]any) {
				args["description"] = ""
			},
			expectError: "description",
		},
		{
			// FR-002, FR-008: Missing required fields should return error
			name: "missing show_creator",
			modifyArgs: func(args map[string]any) {
				delete(args, "show_creator")
			},
			expectError: "show_creator",
		},
		{
			// FR-002, FR-008: Missing required fields should return error
			name: "missing capacity",
			modifyArgs: func(args map[string]any) {
				delete(args, "capacity")
			},
			expectError: "capacity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockEventService{}
			tool, _ := create.New(service)

			ctx := withEventContext(context.Background(), "group-123", "user-456")
			args := validEventArgs()
			tt.modifyArgs(args)

			result, err := tool.Callback(ctx, args)

			require.NoError(t, err)
			assert.Equal(t, false, result["success"])
			assert.Contains(t, result["error"], tt.expectError)
			assert.NotContains(t, result, "chat_room_id")

			// Service should not be called for validation errors
			assert.Equal(t, 0, service.createCount)
		})
	}
}

// =============================================================================
// Callback Tests - Service Errors
// =============================================================================

func TestTool_Callback_ServiceErrors(t *testing.T) {
	// FR-004: Only one event per group chat
	t.Run("returns error when event already exists", func(t *testing.T) {
		service := &mockEventService{
			createErr: errors.New("event already exists: group-123"),
		}
		tool, _ := create.New(service)

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "already exists")
		assert.NotContains(t, result, "chat_room_id")

		assert.Equal(t, 1, service.createCount)
	})

	t.Run("returns error when service Create fails", func(t *testing.T) {
		service := &mockEventService{
			createErr: errors.New("storage error"),
		}
		tool, _ := create.New(service)

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validEventArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, false, result["success"])
		assert.Contains(t, result["error"], "storage error")
		assert.NotContains(t, result, "chat_room_id")

		assert.Equal(t, 1, service.createCount)
	})
}

// =============================================================================
// IsFinal Tests
// =============================================================================

func TestTool_IsFinal(t *testing.T) {
	service := &mockEventService{}
	tool, _ := create.New(service)

	t.Run("returns true when success is true", func(t *testing.T) {
		result := map[string]any{
			"success":      true,
			"chat_room_id": "group-123",
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
			"chat_room_id": "group-123",
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
	createErr        error
	createCount      int
	lastCreatedEvent *event.Event
}

func (m *mockEventService) Create(ctx context.Context, ev *event.Event) error {
	m.createCount++
	m.lastCreatedEvent = ev
	return m.createErr
}
