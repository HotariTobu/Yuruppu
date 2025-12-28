package bot

import (
	"context"
	"encoding/json"
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
	_ Responder      = (*mockResponder)(nil)
	_ Sender         = (*mockSender)(nil)
	_ server.Handler = (*Handler)(nil)
)

func TestNew(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		responder := &mockResponder{}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h := New(responder, sender, logger, historyRepo)

		require.NotNil(t, h)
		assert.Equal(t, responder, h.responder)
		assert.Equal(t, sender, h.sender)
		assert.NotNil(t, h.logger) // logger should never be nil
		assert.Equal(t, historyRepo, h.history)
	})

	t.Run("uses discard logger when nil logger passed", func(t *testing.T) {
		h := New(&mockResponder{}, &mockSender{}, nil, nil)

		require.NotNil(t, h)
		assert.NotNil(t, h.logger) // should be discard handler, not nil
		assert.Nil(t, h.history)
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
			{Role: "user", Content: "My name is Taro", Timestamp: time.Now()},
			{Role: "assistant", Content: "Nice to meet you, Taro!", Timestamp: time.Now()},
		}
		mockStore := &mockStorage{data: serializeHistory(existingHistory), generation: 1}
		responder := &mockResponder{response: "Hello Taro!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Do you remember my name?")

		require.NoError(t, err)
		// Verify history was loaded (read once for GetHistory, once for AppendMessages)
		assert.Equal(t, 2, mockStore.readCallCount)
		assert.Equal(t, "user-123.jsonl", mockStore.lastReadKey)
		// Verify history was passed to responder
		require.Len(t, responder.lastHistory, 2)
		assert.Equal(t, "My name is Taro", responder.lastHistory[0].Content)
		assert.Equal(t, "Nice to meet you, Taro!", responder.lastHistory[1].Content)
	})

	t.Run("passes empty history when no history exists", func(t *testing.T) {
		mockStore := &mockStorage{} // nil data = no history
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Read once for GetHistory, once for AppendMessages
		assert.Equal(t, 2, mockStore.readCallCount)
		assert.Empty(t, responder.lastHistory)
	})

	t.Run("does not respond when history read fails", func(t *testing.T) {
		mockStore := &mockStorage{readErr: errors.New("GCS read failed")}
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		var readErr *history.ReadError
		assert.True(t, errors.As(err, &readErr), "error should be history.ReadError")
		// Responder should not be called when history read fails
		assert.Empty(t, responder.lastMessage)
		// Sender should not be called when history read fails
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("passes nil history when history is nil", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil) // no history

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		assert.Nil(t, responder.lastHistory)
	})
}

// =============================================================================
// History Storage Integration Tests (FR-001)
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message and bot response to history after successful reply", func(t *testing.T) {
		mockStore := &mockStorage{}
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Verify storage was called
		require.Equal(t, 1, mockStore.writeCallCount)
		assert.Equal(t, "user-123.jsonl", mockStore.lastWriteKey)
		// Should use generation 0 for new file (DoesNotExist precondition)
		assert.Equal(t, int64(0), mockStore.lastWriteGeneration)

		// Parse the written data to verify messages
		messages := parseHistory(mockStore.lastWriteData)
		require.Len(t, messages, 2)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hi", messages[0].Content)
		assert.Equal(t, "assistant", messages[1].Role)
		assert.Equal(t, "Hello!", messages[1].Content)
		// Verify timestamps are set (within 1 second of now)
		assert.WithinDuration(t, time.Now(), messages[0].Timestamp, time.Second)
		assert.WithinDuration(t, time.Now(), messages[1].Timestamp, time.Second)
	})

	t.Run("uses generation precondition when appending to existing history", func(t *testing.T) {
		existingHistory := []history.Message{
			{Role: "user", Content: "Previous message", Timestamp: time.Now()},
		}
		mockStore := &mockStorage{
			data:       serializeHistory(existingHistory),
			generation: 12345,
		}
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Should use existing generation for update
		assert.Equal(t, int64(12345), mockStore.lastWriteGeneration)
	})

	t.Run("does not save history when responder fails", func(t *testing.T) {
		mockStore := &mockStorage{}
		responder := &mockResponder{err: errors.New("LLM failed")}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, 0, mockStore.writeCallCount, "storage should not be called when responder fails")
	})

	t.Run("saves history even when sender fails", func(t *testing.T) {
		// History is saved BEFORE sending
		// If storage succeeds and sender fails, history is already saved
		mockStore := &mockStorage{}
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		// Storage was called before sender, so history is saved
		assert.Equal(t, 1, mockStore.writeCallCount)
	})

	t.Run("does not respond when storage fails", func(t *testing.T) {
		mockStore := &mockStorage{writeErr: errors.New("GCS failed")}
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, historyRepo)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		// Should return error (logged by server)
		require.Error(t, err)
		var writeErr *history.WriteError
		assert.True(t, errors.As(err, &writeErr), "error should be history.WriteError")
		// Sender should NOT be called when storage fails
		assert.Equal(t, 0, sender.callCount, "sender should not be called when storage fails")
	})

	t.Run("works without history (nil history)", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger, nil) // no history

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

func (m *mockResponder) Respond(ctx context.Context, userMessage string, hist []history.Message) (string, error) {
	m.lastMessage = userMessage
	m.lastHistory = hist
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

// mockStorage implements storage.Storage interface
type mockStorage struct {
	// Read behavior
	data          []byte
	generation    int64
	readErr       error
	readCallCount int
	lastReadKey   string

	// Write behavior
	writeErr            error
	writeCallCount      int
	lastWriteKey        string
	lastWriteData       []byte
	lastWriteGeneration int64
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	m.lastReadKey = key
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	return m.data, m.generation, nil
}

func (m *mockStorage) Write(ctx context.Context, key string, data []byte, expectedGeneration int64) error {
	m.writeCallCount++
	m.lastWriteKey = key
	m.lastWriteData = data
	m.lastWriteGeneration = expectedGeneration
	return m.writeErr
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

// serializeHistory converts messages to JSONL bytes
func serializeHistory(messages []history.Message) []byte {
	result := make([]byte, 0, len(messages)*100) // pre-allocate
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			panic(err) // test helper only
		}
		result = append(result, data...)
		result = append(result, '\n')
	}
	return result
}

// parseHistory converts JSONL bytes to messages
func parseHistory(data []byte) []history.Message {
	var messages []history.Message
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var msg history.Message
		if err := json.Unmarshal(line, &msg); err == nil {
			messages = append(messages, msg)
		}
	}
	return messages
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
