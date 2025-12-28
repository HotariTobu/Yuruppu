package bot

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/history"
	"yuruppu/internal/line/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks
var (
	_ Responder       = (*mockResponder)(nil)
	_ Sender          = (*mockSender)(nil)
	_ server.Handler  = (*Handler)(nil)
	_ history.Storage = (*mockStorage)(nil)
)

func TestNew(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		responder := &mockResponder{}
		sender := &mockSender{}
		storage := &mockStorage{}
		logger := slog.New(slog.DiscardHandler)

		h := New(responder, sender, logger, storage)

		require.NotNil(t, h)
		assert.Equal(t, responder, h.responder)
		assert.Equal(t, sender, h.sender)
		assert.Equal(t, logger, h.logger)
		assert.Equal(t, storage, h.storage)
	})

	t.Run("accepts nil logger and storage", func(t *testing.T) {
		h := New(&mockResponder{}, &mockSender{}, nil, nil)

		require.NotNil(t, h)
		assert.Nil(t, h.logger)
		assert.Nil(t, h.storage)
	})
}

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - responds and sends reply", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		assert.Equal(t, "Hi", responder.lastMessage)
		assert.Equal(t, "reply-token", sender.lastReplyToken)
		assert.Equal(t, "Hello!", sender.lastText)
	})

	t.Run("responder error - returns error", func(t *testing.T) {
		responder := &mockResponder{err: errors.New("LLM failed")}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, "LLM failed", err.Error())
		assert.Equal(t, 0, sender.callCount) // sender should not be called
	})

	t.Run("sender error - returns error", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, "LINE API failed", err.Error())
	})
}

func TestHandler_HandleImage(t *testing.T) {
	t.Run("converts image to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "I see an image!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleImage(context.Background(), "reply-token", "user-123", "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image]", responder.lastMessage)
	})
}

func TestHandler_HandleSticker(t *testing.T) {
	t.Run("converts sticker to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "Nice sticker!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleSticker(context.Background(), "reply-token", "user-123", "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a sticker]", responder.lastMessage)
	})
}

func TestHandler_HandleVideo(t *testing.T) {
	t.Run("converts video to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "I see a video!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleVideo(context.Background(), "reply-token", "user-123", "msg-789")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a video]", responder.lastMessage)
	})
}

func TestHandler_HandleAudio(t *testing.T) {
	t.Run("converts audio to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "I hear audio!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleAudio(context.Background(), "reply-token", "user-123", "msg-101")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an audio]", responder.lastMessage)
	})
}

func TestHandler_HandleLocation(t *testing.T) {
	t.Run("converts location to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "Nice place!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleLocation(context.Background(), "reply-token", "user-123", 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a location]", responder.lastMessage)
	})
}

func TestHandler_HandleUnknown(t *testing.T) {
	t.Run("converts unknown message to text placeholder", func(t *testing.T) {
		responder := &mockResponder{response: "I got your message!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil)

		err := h.HandleUnknown(context.Background(), "reply-token", "user-123")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a message]", responder.lastMessage)
	})
}

// =============================================================================
// History Context Tests (FR-002)
// =============================================================================

func TestHandler_HistoryContext(t *testing.T) {
	t.Run("loads history and passes to responder", func(t *testing.T) {
		existingHistory := []history.Message{
			{Role: "user", Content: "My name is Taro"},
			{Role: "assistant", Content: "Nice to meet you, Taro!"},
		}
		responder := &mockResponder{response: "Hello Taro!"}
		sender := &mockSender{}
		storage := &mockStorage{history: existingHistory}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Do you remember my name?")

		require.NoError(t, err)
		// Verify history was loaded
		assert.Equal(t, 1, storage.getCallCount)
		assert.Equal(t, "user-123", storage.lastGetSourceID)
		// Verify history was passed to responder
		require.Len(t, responder.lastHistory, 2)
		assert.Equal(t, "My name is Taro", responder.lastHistory[0].Content)
		assert.Equal(t, "Nice to meet you, Taro!", responder.lastHistory[1].Content)
	})

	t.Run("passes empty history when no history exists", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		storage := &mockStorage{history: []history.Message{}} // empty history
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		assert.Equal(t, 1, storage.getCallCount)
		assert.Empty(t, responder.lastHistory)
	})

	t.Run("does not respond when history read fails (NFR-002)", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		storage := &mockStorage{getErr: &history.StorageReadError{Message: "GCS read failed"}}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		var storageErr *history.StorageReadError
		assert.True(t, errors.As(err, &storageErr), "error should be StorageReadError")
		// Responder should not be called when history read fails
		assert.Empty(t, responder.lastMessage)
		// Sender should not be called when history read fails
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("passes nil history when storage is nil", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil) // no storage

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		assert.Nil(t, responder.lastHistory)
	})
}

// =============================================================================
// History Storage Integration Tests (FR-001, NFR-002)
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message and bot response to history after successful reply", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		storage := &mockStorage{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Verify storage was called with correct messages
		require.Equal(t, 1, storage.appendCallCount)
		assert.Equal(t, "user-123", storage.lastSourceID)
		assert.Equal(t, "user", storage.lastUserMsg.Role)
		assert.Equal(t, "Hi", storage.lastUserMsg.Content)
		assert.Equal(t, "assistant", storage.lastBotMsg.Role)
		assert.Equal(t, "Hello!", storage.lastBotMsg.Content)
		// Verify timestamps are set (within 1 second of now)
		assert.WithinDuration(t, time.Now(), storage.lastUserMsg.Timestamp, time.Second)
		assert.WithinDuration(t, time.Now(), storage.lastBotMsg.Timestamp, time.Second)
	})

	t.Run("does not save history when responder fails", func(t *testing.T) {
		responder := &mockResponder{err: errors.New("LLM failed")}
		sender := &mockSender{}
		storage := &mockStorage{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, 0, storage.appendCallCount, "storage should not be called when responder fails")
	})

	t.Run("saves history even when sender fails", func(t *testing.T) {
		// History is saved BEFORE sending (per NFR-002: storage errors must prevent response)
		// If storage succeeds and sender fails, history is already saved
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		storage := &mockStorage{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		// Storage was called before sender, so history is saved
		assert.Equal(t, 1, storage.appendCallCount)
	})

	t.Run("does not respond when storage fails (NFR-002)", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		storage := &mockStorage{appendErr: &history.StorageWriteError{Message: "GCS failed"}}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, storage)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		// Should return error (logged by server)
		require.Error(t, err)
		var storageErr *history.StorageWriteError
		assert.True(t, errors.As(err, &storageErr), "error should be StorageWriteError")
		// Sender should still be called because we need to send the response
		// But since storage failed, we don't send
		// Actually per NFR-002: "do not respond when storage fails" - sender should NOT be called
		// Re-reading NFR-002: "ストレージ障害時は応答を生成しない" = "do not generate response when storage fails"
		// This means we should not send a reply when storage write fails
		assert.Equal(t, 0, sender.callCount, "sender should not be called when storage fails")
	})

	t.Run("works without storage (nil storage)", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil) // no storage

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		assert.Equal(t, "Hello!", sender.lastText)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockResponder struct {
	response    string
	err         error
	lastMessage string
	lastHistory []history.Message
}

func (m *mockResponder) Respond(ctx context.Context, userMessage string, history []history.Message) (string, error) {
	m.lastMessage = userMessage
	m.lastHistory = history
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type mockSender struct {
	err            error
	lastReplyToken string
	lastText       string
	callCount      int
}

func (m *mockSender) SendReply(replyToken string, text string) error {
	m.callCount++
	m.lastReplyToken = replyToken
	m.lastText = text
	return m.err
}

type mockStorage struct {
	// GetHistory behavior
	history         []history.Message
	getErr          error
	getCallCount    int
	lastGetSourceID string

	// AppendMessages behavior
	appendErr       error
	appendCallCount int
	lastSourceID    string
	lastUserMsg     history.Message
	lastBotMsg      history.Message
}

func (m *mockStorage) GetHistory(ctx context.Context, sourceID string) ([]history.Message, error) {
	m.getCallCount++
	m.lastGetSourceID = sourceID
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.history, nil
}

func (m *mockStorage) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg history.Message) error {
	m.appendCallCount++
	m.lastSourceID = sourceID
	m.lastUserMsg = userMsg
	m.lastBotMsg = botMsg
	return m.appendErr
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
