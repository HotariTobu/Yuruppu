package remove_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/event"
	"yuruppu/internal/line"
	"yuruppu/internal/toolset/event/remove"

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

// =============================================================================
// New() Tests
// =============================================================================

func TestNew(t *testing.T) {
	t.Run("creates tool with valid service", func(t *testing.T) {
		service := &mockEventService{}

		tool, err := remove.New(service, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "remove_event", tool.Name())
	})

	t.Run("returns error when service is nil", func(t *testing.T) {
		tool, err := remove.New(nil, slog.New(slog.NewTextHandler(nil, nil)))

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "eventService cannot be nil")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		service := &mockEventService{}

		tool, err := remove.New(service, nil)

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
	tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

	t.Run("Name returns remove_event", func(t *testing.T) {
		assert.Equal(t, "remove_event", tool.Name())
	})

	t.Run("Description is meaningful", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "remove")
		assert.Contains(t, desc, "event")
		assert.Contains(t, desc, "creator")
	})

	t.Run("ParametersJsonSchema is valid JSON", func(t *testing.T) {
		schema := tool.ParametersJsonSchema()
		assert.NotEmpty(t, schema)
		// remove_event has no parameters
		assert.Contains(t, string(schema), "object")
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
	// AC-004: イベント削除（作成者による削除）[FR-007, FR-008, FR-009, FR-011]
	t.Run("deletes event when called by creator", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Some description",
			},
		}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{} // delete_event has no parameters

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, "group-123", result["chat_room_id"])

		require.Equal(t, 1, service.getCount)
		assert.Equal(t, "group-123", service.lastGetChatRoomID)

		require.Equal(t, 1, service.removeCount)
		assert.Equal(t, "group-123", service.lastRemoveChatRoomID)
	})

	t.Run("deletes event with different chat room ID", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-999",
				CreatorID:   "user-888",
				Title:       "Workshop",
				Description: "Workshop details",
			},
		}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-999", "user-888")
		args := map[string]any{}

		result, err := tool.Callback(ctx, args)

		require.NoError(t, err)
		assert.Equal(t, "group-999", result["chat_room_id"])
		assert.Equal(t, "group-999", service.lastRemoveChatRoomID)
	})
}

// =============================================================================
// Callback Tests - Authorization Errors
// =============================================================================

func TestTool_Callback_AuthorizationError(t *testing.T) {
	// AC-005: イベント削除（作成者以外）[FR-008]
	t.Run("returns error when user is not creator", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456", // Creator
				Title:       "Team Meeting",
				Description: "Some description",
			},
		}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-999") // Different user
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "only the event creator can remove the event")

		// Get should be called to check authorization
		assert.Equal(t, 1, service.getCount)
		// Delete should NOT be called
		assert.Equal(t, 0, service.removeCount)
	})
}

// =============================================================================
// Callback Tests - Context Errors
// =============================================================================

func TestTool_Callback_ContextErrors(t *testing.T) {
	// FR-009: Delete can only be executed from the group chat where the event exists
	t.Run("returns error when sourceID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := line.WithUserID(context.Background(), "user-123")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
		assert.Equal(t, 0, service.getCount)
		assert.Equal(t, 0, service.removeCount)
	})

	t.Run("returns error when userID not in context", func(t *testing.T) {
		service := &mockEventService{}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := line.WithSourceID(context.Background(), "group-123")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
		assert.Equal(t, 0, service.getCount)
		assert.Equal(t, 0, service.removeCount)
	})
}

// =============================================================================
// Callback Tests - Not Found Errors
// =============================================================================

func TestTool_Callback_NotFoundError(t *testing.T) {
	// AC-006: イベント削除（イベントが存在しない）[FR-010]
	t.Run("returns error when event does not exist in current chat room", func(t *testing.T) {
		service := &mockEventService{
			getErr: errors.New("event not found: group-123"),
		}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")

		// Get should be called
		assert.Equal(t, 1, service.getCount)
		// Delete should NOT be called
		assert.Equal(t, 0, service.removeCount)
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
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Equal(t, 1, service.getCount)
		assert.Equal(t, 0, service.removeCount)
	})

	t.Run("returns error when service Delete fails", func(t *testing.T) {
		service := &mockEventService{
			getEvent: &event.Event{
				ChatRoomID:  "group-123",
				CreatorID:   "user-456",
				Title:       "Team Meeting",
				Description: "Some description",
			},
			removeErr: errors.New("storage write error"),
		}
		tool, _ := remove.New(service, slog.New(slog.DiscardHandler))

		ctx := withEventContext(context.Background(), "group-123", "user-456")
		args := map[string]any{}

		_, err := tool.Callback(ctx, args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove event")
		assert.Equal(t, 1, service.getCount)
		assert.Equal(t, 1, service.removeCount)
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

	// Remove method
	removeErr            error
	removeCount          int
	lastRemoveChatRoomID string
}

func (m *mockEventService) Get(ctx context.Context, chatRoomID string) (*event.Event, error) {
	m.getCount++
	m.lastGetChatRoomID = chatRoomID
	return m.getEvent, m.getErr
}

func (m *mockEventService) Remove(ctx context.Context, chatRoomID string) error {
	m.removeCount++
	m.lastRemoveChatRoomID = chatRoomID
	return m.removeErr
}
