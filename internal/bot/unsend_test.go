package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleUnsend Tests (FR-003)
// =============================================================================

// TestHandleUnsend_RemovesMessage tests the main functionality: removing a message by MessageID
// AC-001: Unsend event triggers message removal [FR-003]
func TestHandleUnsend_RemovesMessage(t *testing.T) {
	t.Run("removes message with matching MessageID from history", func(t *testing.T) {
		// Given: A history with a message having a specific MessageID
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		// Create initial history with a message that has MessageID
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "First message"},
				},
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Second message"},
				},
				Timestamp: time.Now().Add(-1 * time.Minute),
			},
		}

		// Save initial history
		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		// Create handler
		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called with the MessageID
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: The operation succeeds
		require.NoError(t, err)

		// And: The message with msg-001 is removed from history
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1, "history should contain 1 message after unsend")

		// Verify the remaining message is msg-002
		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok, "message should be UserMessage")
		assert.Equal(t, "msg-002", userMsg.MessageID)
		assert.Equal(t, "Second message", userMsg.Parts[0].(*history.UserTextPart).Text)
	})

	t.Run("removes correct message when multiple messages exist", func(t *testing.T) {
		// Given: A history with multiple messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "First"},
				},
				Timestamp: time.Now().Add(-3 * time.Minute),
			},
			&history.AssistantMessage{
				ModelName: "test-model",
				Parts: []history.AssistantPart{
					&history.AssistantTextPart{Text: "Reply to first"},
				},
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Second"},
				},
				Timestamp: time.Now().Add(-1 * time.Minute),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called for msg-002
		err = h.HandleUnsend(ctx, "msg-002")

		// Then: Only msg-002 is removed
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 2, "should have 2 messages remaining")

		// Verify msg-001 is still present
		userMsg1, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-001", userMsg1.MessageID)

		// Verify assistant message is still present
		_, ok = messages[1].(*history.AssistantMessage)
		require.True(t, ok)
	})

	t.Run("removes message from group chat history", func(t *testing.T) {
		// AC-003: Unsend in group chat [FR-003]
		// Given: A message exists in a group chat history
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-group-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Group message 1"},
				},
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-group-002",
				UserID:    "user-456",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Group message 2"},
				},
				Timestamp: time.Now().Add(-1 * time.Minute),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "group-789", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: A user unsends that message
		err = h.HandleUnsend(ctx, "msg-group-001")

		// Then: The message is removed from the group's history
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "group-789")
		require.NoError(t, err)
		require.Len(t, messages, 1)

		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-group-002", userMsg.MessageID)
		assert.Equal(t, "user-456", userMsg.UserID)
	})

	t.Run("persists updated history to storage", func(t *testing.T) {
		// FR-004: System persists the updated history to storage
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Message to unsend"},
				},
				Timestamp: time.Now(),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		initialWriteCount := mockStore.writeCallCount

		// When: HandleUnsend is called
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: The updated history is written to storage
		require.NoError(t, err)
		assert.Greater(t, mockStore.writeCallCount, initialWriteCount, "history should be written to storage")
	})
}

// TestHandleUnsend_MessageNotFound tests idempotent behavior when message is not found
// AC-002: Unsend for non-existent message [FR-002, Error]
func TestHandleUnsend_MessageNotFound(t *testing.T) {
	t.Run("returns nil when message ID not found in history", func(t *testing.T) {
		// Given: History does not contain a message with the specified message ID
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		// Create history without the target MessageID
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Existing message"},
				},
				Timestamp: time.Now(),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: An unsend event is received for a non-existent message ID
		err = h.HandleUnsend(ctx, "msg-nonexistent")

		// Then: The system does not fail (idempotent operation)
		require.NoError(t, err)

		// And: No changes are made to the history
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1, "history should remain unchanged")

		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-001", userMsg.MessageID)
	})

	t.Run("returns nil when history is empty", func(t *testing.T) {
		// Given: Empty history
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called on empty history
		err = h.HandleUnsend(ctx, "msg-any")

		// Then: No error is returned
		require.NoError(t, err)

		// And: History remains empty
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("returns nil when message has no MessageID (legacy message)", func(t *testing.T) {
		// Given: History contains legacy messages without MessageID
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "", // Legacy message without MessageID
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Legacy message"},
				},
				Timestamp: time.Now(),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called for any MessageID
		err = h.HandleUnsend(ctx, "msg-any")

		// Then: Operation succeeds (legacy messages cannot be matched)
		require.NoError(t, err)

		// And: Legacy message remains in history
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1)
	})
}

// TestHandleUnsend_PreservesOtherMessages tests that unsending preserves other messages
// AC-003: Other messages in the group history remain unaffected [FR-003]
func TestHandleUnsend_PreservesOtherMessages(t *testing.T) {
	t.Run("preserves other user messages", func(t *testing.T) {
		// Given: A history with multiple user messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		ts := time.Now()
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Message 1"},
				},
				Timestamp: ts.Add(-3 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Message 2"},
				},
				Timestamp: ts.Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-003",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Message 3"},
				},
				Timestamp: ts.Add(-1 * time.Minute),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: One message is unsent
		err = h.HandleUnsend(ctx, "msg-002")

		// Then: Other messages remain intact
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 2)

		// Verify msg-001 is preserved
		userMsg1, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-001", userMsg1.MessageID)
		assert.Equal(t, "Message 1", userMsg1.Parts[0].(*history.UserTextPart).Text)
		assert.Equal(t, ts.Add(-3*time.Minute).Unix(), userMsg1.Timestamp.Unix())

		// Verify msg-003 is preserved
		userMsg3, ok := messages[1].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-003", userMsg3.MessageID)
		assert.Equal(t, "Message 3", userMsg3.Parts[0].(*history.UserTextPart).Text)
		assert.Equal(t, ts.Add(-1*time.Minute).Unix(), userMsg3.Timestamp.Unix())
	})

	t.Run("preserves assistant messages", func(t *testing.T) {
		// Given: A history with user and assistant messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		ts := time.Now()
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Question"},
				},
				Timestamp: ts.Add(-2 * time.Minute),
			},
			&history.AssistantMessage{
				ModelName: "test-model",
				Parts: []history.AssistantPart{
					&history.AssistantTextPart{Text: "Answer"},
				},
				Timestamp: ts.Add(-1 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Follow-up"},
				},
				Timestamp: ts,
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: The first user message is unsent
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: Assistant message and other user messages are preserved
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 2)

		// Verify assistant message is preserved
		assistantMsg, ok := messages[0].(*history.AssistantMessage)
		require.True(t, ok)
		assert.Equal(t, "test-model", assistantMsg.ModelName)
		assert.Equal(t, "Answer", assistantMsg.Parts[0].(*history.AssistantTextPart).Text)

		// Verify second user message is preserved
		userMsg2, ok := messages[1].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-002", userMsg2.MessageID)
	})

	t.Run("preserves messages with multiple parts", func(t *testing.T) {
		// Given: Messages with file data parts
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Check this image"},
					&history.UserFileDataPart{
						StorageKey:  "user-123/image1.jpg",
						MIMEType:    "image/jpeg",
						DisplayName: "photo.jpg",
					},
				},
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts: []history.UserPart{
					&history.UserTextPart{Text: "Simple text"},
				},
				Timestamp: time.Now().Add(-1 * time.Minute),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: The simple text message is unsent
		err = h.HandleUnsend(ctx, "msg-002")

		// Then: The message with multiple parts is preserved completely
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1)

		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-001", userMsg.MessageID)
		require.Len(t, userMsg.Parts, 2)

		// Verify text part
		textPart, ok := userMsg.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Check this image", textPart.Text)

		// Verify file data part
		filePart, ok := userMsg.Parts[1].(*history.UserFileDataPart)
		require.True(t, ok)
		assert.Equal(t, "user-123/image1.jpg", filePart.StorageKey)
		assert.Equal(t, "image/jpeg", filePart.MIMEType)
		assert.Equal(t, "photo.jpg", filePart.DisplayName)
	})

	t.Run("preserves message order after unsend", func(t *testing.T) {
		// Given: A history with chronological messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		ts := time.Now()
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "First"}},
				Timestamp: ts.Add(-5 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-002",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Second"}},
				Timestamp: ts.Add(-4 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-003",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Third"}},
				Timestamp: ts.Add(-3 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-004",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Fourth"}},
				Timestamp: ts.Add(-2 * time.Minute),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: A middle message is unsent
		err = h.HandleUnsend(ctx, "msg-002")

		// Then: The order of remaining messages is preserved
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 3)

		// Verify chronological order is maintained
		msg1, _ := messages[0].(*history.UserMessage)
		assert.Equal(t, "msg-001", msg1.MessageID)
		assert.Equal(t, "First", msg1.Parts[0].(*history.UserTextPart).Text)

		msg3, _ := messages[1].(*history.UserMessage)
		assert.Equal(t, "msg-003", msg3.MessageID)
		assert.Equal(t, "Third", msg3.Parts[0].(*history.UserTextPart).Text)

		msg4, _ := messages[2].(*history.UserMessage)
		assert.Equal(t, "msg-004", msg4.MessageID)
		assert.Equal(t, "Fourth", msg4.Parts[0].(*history.UserTextPart).Text)

		// Verify timestamps are still in order
		assert.True(t, msg1.Timestamp.Before(msg3.Timestamp))
		assert.True(t, msg3.Timestamp.Before(msg4.Timestamp))
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestHandleUnsend_ErrorHandling(t *testing.T) {
	t.Run("returns error when history read fails", func(t *testing.T) {
		// Given: History service that fails to read
		mockStore := newMockStorage()
		mockStore.readErr = errors.New("GCS read failed")
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: Error is returned
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load history")
	})

	t.Run("returns error when history write fails", func(t *testing.T) {
		// Given: History service that fails to write
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		// Set up initial history
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message"}},
				Timestamp: time.Now(),
			},
		}
		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		// Configure write to fail on the next call
		mockStore.writeResults = []writeResult{
			{gen: 0, err: errors.New("GCS write quota exceeded")},
		}

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: Error is returned
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save history")
	})

	t.Run("preserves error chain for history read failure", func(t *testing.T) {
		// Given: History service with specific error
		mockStore := newMockStorage()
		storageErr := errors.New("GCS bucket not found")
		mockStore.readErr = storageErr
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: Error chain is preserved
		require.Error(t, err)
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
	})
}

// =============================================================================
// Concurrency Tests (NFR-001)
// =============================================================================

func TestHandleUnsend_Concurrency(t *testing.T) {
	t.Run("uses optimistic locking for concurrent modifications", func(t *testing.T) {
		// NFR-001: History updates must prevent data corruption from concurrent modifications
		// Given: A history with a message
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message"}},
				Timestamp: time.Now(),
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: The write uses the expected generation from the read
		require.NoError(t, err)
		// Verify that expectedGeneration passed to Write matches the generation from Read
		// The mockStorage should have recorded the expectedGen parameter
		assert.Equal(t, int64(1), mockStore.lastWriteExpectedGen,
			"write should use generation from read (optimistic locking)")
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestHandleUnsend_EdgeCases(t *testing.T) {
	t.Run("handles empty MessageID parameter", func(t *testing.T) {
		// Given: A history with messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-001",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message"}},
				Timestamp: time.Now(),
			},
		}
		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called with empty MessageID
		err = h.HandleUnsend(ctx, "")

		// Then: Operation succeeds (no match, idempotent)
		require.NoError(t, err)

		// And: History remains unchanged
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1)
	})

	t.Run("handles duplicate MessageIDs (removes all matches)", func(t *testing.T) {
		// Edge case: If somehow history has duplicate MessageIDs, remove all of them
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		ts := time.Now()
		initialMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-duplicate",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "First instance"}},
				Timestamp: ts.Add(-2 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-duplicate",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Second instance"}},
				Timestamp: ts.Add(-1 * time.Minute),
			},
			&history.UserMessage{
				MessageID: "msg-other",
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Other message"}},
				Timestamp: ts,
			},
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called for duplicate MessageID
		err = h.HandleUnsend(ctx, "msg-duplicate")

		// Then: All messages with that MessageID are removed
		require.NoError(t, err)

		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 1, "both duplicate messages should be removed")

		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-other", userMsg.MessageID)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Given: A cancelled context
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		ctx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleUnsend is called with cancelled context
		err = h.HandleUnsend(ctx, "msg-001")

		// Then: Context error is returned
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled), "should return context.Canceled error")
	})

	t.Run("handles very large history", func(t *testing.T) {
		// Given: A history with many messages
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		// Create 100 messages
		initialMessages := make([]history.Message, 0, 100)
		ts := time.Now()
		for i := range 100 {
			initialMessages = append(initialMessages, &history.UserMessage{
				MessageID: "msg-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
				UserID:    "user-123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message " + string(rune('0'+i))}},
				Timestamp: ts.Add(-time.Duration(100-i) * time.Minute),
			})
		}

		_, err = historyRepo.PutHistory(ctx, "user-123", initialMessages, 0)
		require.NoError(t, err)

		h, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			&mockGroupProfileService{},
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: A message in the middle is unsent (i=30 → 'A'+4='E', '0'+1='1' → "msg-E1")
		err = h.HandleUnsend(ctx, "msg-E1")

		// Then: Operation succeeds
		require.NoError(t, err)

		// And: Correct message is removed
		messages, _, err := historyRepo.GetHistory(ctx, "user-123")
		require.NoError(t, err)
		require.Len(t, messages, 99)

		// Verify the removed message is not present
		for _, msg := range messages {
			userMsg, ok := msg.(*history.UserMessage)
			require.True(t, ok)
			assert.NotEqual(t, "msg-E1", userMsg.MessageID)
		}
	})
}
