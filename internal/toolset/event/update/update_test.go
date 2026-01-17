package update_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/toolset/event/update"

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

// validUpdateArgs returns valid arguments for updating an event.
func validUpdateArgs() map[string]any {
	return map[string]any{
		"description": "Updated event description",
	}
}

// =============================================================================
// New() Tests
// =============================================================================

func TestNew(t *testing.T) {
	t.Run("creates tool with valid service", func(t *testing.T) {
		service := &mockEventService{}

		tool, err := update.New(service, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "update_event", tool.Name())
	})

	t.Run("returns error when service is nil", func(t *testing.T) {
		tool, err := update.New(nil, slog.New(slog.NewTextHandler(nil, nil)))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		service := &mockEventService{}

		tool, err := update.New(service, nil)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})
}

// =============================================================================
// Tool Interface Tests
// =============================================================================

func TestTool_Metadata(t *testing.T) {
	service := &mockEventService{}
	tool, _ := update.New(service, slog.New(slog.DiscardHandler))

	t.Run("Name returns update_event", func(t *testing.T) {
		assert.Equal(t, "update_event", tool.Name())
	})

	t.Run("Description is meaningful", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "update")
		assert.Contains(t, desc, "event")
		assert.Contains(t, desc, "creator")
	})

	t.Run("ParametersJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ParametersJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "description")
	})

	t.Run("ResponseJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ResponseJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "chat_room_id")
	})
}

// =============================================================================
// Callback Tests - Success Cases
// =============================================================================

func TestTool_Callback_Success(t *testing.T) {
	// AC-001: イベント説明の更新
	t.Run("updates event description when called by creator", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Old description",
			},
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validUpdateArgs()

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, "group-123", result["chat_room_id"])

		require.Equal(t, 1, service.getCount)
		assert.Equal(t, "group-123", service.lastGetChatRoomID)

		require.Equal(t, 1, service.updateCount)
		assert.Equal(t, "group-123", service.lastUpdateChatRoomID)
		assert.Equal(t, "Updated event description", service.lastUpdateDescription)
	})

	t.Run("handles different description content", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-999",
				CreatorID:   "user-888",
				Title:       "Workshop",
				Description: "Original content",
			},
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-888")
		args := map[string]any{
			"description": "New workshop details with special characters: @#$%",
		}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, "group-999", result["chat_room_id"])
		assert.Equal(t, "New workshop details with special characters: @#$%", service.lastUpdateDescription)
	})
}

// =============================================================================
// Callback Tests - Authorization Errors
// =============================================================================

func TestTool_Callback_AuthorizationError(t *testing.T) {
	// AC-002: イベント更新（作成者以外） [FR-004]
	t.Run("returns error when user is not creator", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456", // Creator
				Title:       "Team Meeting",
				Description: "Some description",
			},
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-999") // Different user
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "only the event creator can update the event")

		// Get should be called to check authorization
		assert.Equal(t, 1, service.getCount)
		// Update should NOT be called
		assert.Equal(t, 0, service.updateCount)
	})
}

// =============================================================================
// Callback Tests - Context Errors
// =============================================================================

func TestTool_Callback_ContextErrors(t *testing.T) {
	// FR-005: Update can only be executed from the group chat where the event exists
	t.Run("returns error when sourceID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := line.WithUserID(context.Background(), "user-123")
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
		assert.Equal(t, 0, service.getCount)
		assert.Equal(t, 0, service.updateCount)
	})

	t.Run("returns error when userID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := line.WithSourceID(context.Background(), "group-123")
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
		assert.Equal(t, 0, service.getCount)
		assert.Equal(t, 0, service.updateCount)
	})
}

// =============================================================================
// Callback Tests - Not Found Errors
// =============================================================================

func TestTool_Callback_NotFoundError(t *testing.T) {
	// AC-003: イベント更新（イベントが存在しない） [FR-006]
	t.Run("returns error when event does not exist in current chat room", func(t *testing.T) {
		service := &mockEventService{
			getErr: errors.New("event not found: group-123"),
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")

		// Get should be called
		assert.Equal(t, 1, service.getCount)
		// Update should NOT be called
		assert.Equal(t, 0, service.updateCount)
	})
}

// =============================================================================
// Callback Tests - Validation Errors
// =============================================================================

func TestTool_Callback_ValidationErrors(t *testing.T) {
	t.Run("returns error when description is not a string", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Old description",
			},
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{
			"description": 12345, // Invalid type
		}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid description")
		assert.Equal(t, 0, service.updateCount)
	})

	t.Run("returns error when description is missing", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Old description",
			},
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{} // Missing description

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid description")
		assert.Equal(t, 0, service.updateCount)
	})
}

// =============================================================================
// Callback Tests - Service Errors
// =============================================================================

func TestTool_Callback_ServiceErrors(t *testing.T) {
	t.Run("returns error when service Get fails", func(t *testing.T) {
		service := &mockEventService{
			getErr: errors.New("storage error"),
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Equal(t, 1, service.getCount)
		assert.Equal(t, 0, service.updateCount)
	})

	t.Run("returns error when service Update fails", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Old description",
			},
			updateErr: errors.New("storage write error"),
		}
		tool, _ := update.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := validUpdateArgs()

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update event")
		assert.Equal(t, 1, service.getCount)
		assert.Equal(t, 1, service.updateCount)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockEventService struct {
	// Get method
	getEvent          *event.Event
	getErr            error
	getCount          int
	lastGetChatRoomID string

	// Update method
	updateErr             error
	updateCount           int
	lastUpdateChatRoomID  string
	lastUpdateDescription string
}

func (m *mockEventService) Get(ctx context.Context, chatRoomID string) (*event.Event, error) {
	m.getCount++
	m.lastGetChatRoomID = chatRoomID
	return m.getEvent, m.getErr
}

func (m *mockEventService) Update(ctx context.Context, chatRoomID string, description string) error {
	m.updateCount++
	m.lastUpdateChatRoomID = chatRoomID
	m.lastUpdateDescription = description
	return m.updateErr
}
