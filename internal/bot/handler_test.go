package bot

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks
var (
	_ agent.Agent  = (*mockAgent)(nil)
	_ Sender       = (*mockSender)(nil)
	_ line.Handler = (*Handler)(nil)
)

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		mockAg := &mockAgent{}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h := NewHandler(historyRepo, mockAg, sender, logger)

		require.NotNil(t, h)
		assert.Equal(t, historyRepo, h.history)
		assert.Equal(t, mockAg, h.agent)
		assert.Equal(t, sender, h.sender)
		assert.NotNil(t, h.logger) // logger should never be nil
	})

	t.Run("uses discard logger when nil logger passed", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h := NewHandler(historyRepo, &mockAgent{}, &mockSender{}, nil)

		require.NotNil(t, h)
		assert.NotNil(t, h.logger) // should be discard handler, not nil
	})
}

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - responds and sends reply", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// User message is passed as last element in history
		require.Len(t, mockAg.lastHistory, 1)
		assert.Equal(t, "Hi", mockAg.lastHistory[0].Content)
		assert.Equal(t, "reply-token", sender.lastReplyToken)
		assert.Equal(t, "Hello!", sender.lastText)
	})

	t.Run("agent error - returns error", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, "LLM failed", err.Error())
		assert.Equal(t, 0, sender.callCount) // sender should not be called
	})

	t.Run("sender error - returns error", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, "LINE API failed", err.Error())
	})
}

func TestHandler_HandleImage(t *testing.T) {
	t.Run("converts image to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "I see an image!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleImage(context.Background(), "reply-token", "user-123", "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image]", mockAg.lastMessage)
	})
}

func TestHandler_HandleSticker(t *testing.T) {
	t.Run("converts sticker to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Nice sticker!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleSticker(context.Background(), "reply-token", "user-123", "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a sticker]", mockAg.lastMessage)
	})
}

func TestHandler_HandleVideo(t *testing.T) {
	t.Run("converts video to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "I see a video!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleVideo(context.Background(), "reply-token", "user-123", "msg-789")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a video]", mockAg.lastMessage)
	})
}

func TestHandler_HandleAudio(t *testing.T) {
	t.Run("converts audio to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "I hear audio!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleAudio(context.Background(), "reply-token", "user-123", "msg-101")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an audio]", mockAg.lastMessage)
	})
}

func TestHandler_HandleLocation(t *testing.T) {
	t.Run("converts location to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Nice place!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleLocation(context.Background(), "reply-token", "user-123", 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a location]", mockAg.lastMessage)
	})
}

func TestHandler_HandleUnknown(t *testing.T) {
	t.Run("converts unknown message to text placeholder", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "I got your message!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleUnknown(context.Background(), "reply-token", "user-123")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a message]", mockAg.lastMessage)
	})
}

// =============================================================================
// History Context Tests (FR-002)
// =============================================================================

func TestHandler_HistoryContext(t *testing.T) {
	t.Run("loads history and passes to agent with user message", func(t *testing.T) {
		existingHistory := []history.Message{
			{Role: "user", Content: "My name is Taro", Timestamp: time.Now()},
			{Role: "assistant", Content: "Nice to meet you, Taro!", Timestamp: time.Now()},
		}
		mockStore := &mockStorage{data: serializeHistory(existingHistory), generation: 1}
		mockAg := &mockAgent{response: "Hello Taro!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Do you remember my name?")

		require.NoError(t, err)
		// Verify history was loaded (GetHistory + re-read for assistant message)
		assert.Equal(t, 2, mockStore.readCallCount)
		assert.Equal(t, "user-123.jsonl", mockStore.lastReadKey)
		// Verify history with user message was passed to agent
		require.Len(t, mockAg.lastHistory, 3)
		assert.Equal(t, "My name is Taro", mockAg.lastHistory[0].Content)
		assert.Equal(t, "Nice to meet you, Taro!", mockAg.lastHistory[1].Content)
		assert.Equal(t, "Do you remember my name?", mockAg.lastHistory[2].Content)
	})

	t.Run("passes user message only when no history exists", func(t *testing.T) {
		mockStore := &mockStorage{} // nil data = no history
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// GetHistory + re-read for assistant message
		assert.Equal(t, 2, mockStore.readCallCount)
		// Agent receives user message as first and only element
		require.Len(t, mockAg.lastHistory, 1)
		assert.Equal(t, "Hi", mockAg.lastHistory[0].Content)
	})

	t.Run("does not respond when history read fails", func(t *testing.T) {
		mockStore := &mockStorage{readErr: errors.New("GCS read failed")}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		var readErr *history.ReadError
		assert.True(t, errors.As(err, &readErr), "error should be history.ReadError")
		// Agent should not be called when history read fails
		assert.Empty(t, mockAg.lastMessage)
		// Sender should not be called when history read fails
		assert.Equal(t, 0, sender.callCount)
	})
}

// =============================================================================
// History Storage Integration Tests (FR-001)
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message and bot response to history after successful reply", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Verify storage was called twice (user message, then assistant message)
		require.Equal(t, 2, mockStore.writeCallCount)
		assert.Equal(t, "user-123.jsonl", mockStore.lastWriteKey)

		// Parse the final written data to verify messages
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
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.NoError(t, err)
		// Storage write called twice (user message, then assistant message)
		assert.Equal(t, 2, mockStore.writeCallCount)
		// Storage read called twice (initial GetHistory, then re-read for assistant message)
		assert.Equal(t, 2, mockStore.readCallCount)
		// Both writes use generation from GetHistory for optimistic locking
		require.Len(t, mockStore.allWriteGenerations, 2)
		assert.Equal(t, int64(12345), mockStore.allWriteGenerations[0])
		assert.Equal(t, int64(12345), mockStore.allWriteGenerations[1])
	})

	t.Run("saves only user message when agent fails", func(t *testing.T) {
		mockStore := &mockStorage{}
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		// User message is saved before agent is called
		assert.Equal(t, 1, mockStore.writeCallCount)
		// Verify only user message was saved
		messages := parseHistory(mockStore.lastWriteData)
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hi", messages[0].Content)
	})

	t.Run("saves only user message when sender fails", func(t *testing.T) {
		// User message is saved before sending
		// If sender fails, assistant message is NOT saved
		mockStore := &mockStorage{}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		// Only user message was saved (before sender failed)
		assert.Equal(t, 1, mockStore.writeCallCount)
		// Verify only user message was saved
		messages := parseHistory(mockStore.lastWriteData)
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hi", messages[0].Content)
	})

	t.Run("does not respond when user message storage fails", func(t *testing.T) {
		mockStore := &mockStorage{writeErr: errors.New("GCS failed")}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		// Should return error (logged by server)
		require.Error(t, err)
		var writeErr *history.WriteError
		assert.True(t, errors.As(err, &writeErr), "error should be history.WriteError")
		// Agent should NOT be called when user message storage fails
		assert.Empty(t, mockAg.lastMessage)
		// Sender should NOT be called when storage fails
		assert.Equal(t, 0, sender.callCount, "sender should not be called when storage fails")
	})

	t.Run("returns error when assistant message save fails after successful send", func(t *testing.T) {
		// Step 5 failure: message already sent, assistant save fails
		// Expected: return error, sender was called
		mockStore := &mockStorage{
			writeErr:       errors.New("GCS failed on second write"),
			writeErrOnCall: 2, // Fail only on second write (assistant message)
		}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h := NewHandler(historyRepo, mockAg, sender, logger)

		err = h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		// Should return error
		require.Error(t, err)
		// Sender should have been called (message was sent before error)
		assert.Equal(t, 1, sender.callCount)
		assert.Equal(t, "Hello!", sender.lastText)
		// Both writes were attempted
		assert.Equal(t, 2, mockStore.writeCallCount)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockAgent struct {
	response    string
	err         error
	lastHistory []agent.Message
	lastMessage string // extracted from last message in history
}

func (m *mockAgent) GenerateText(ctx context.Context, hist []agent.Message) (string, error) {
	m.lastHistory = hist
	if len(hist) > 0 {
		m.lastMessage = hist[len(hist)-1].Content
	}
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockAgent) Close(ctx context.Context) error {
	return nil
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
	writeErrOnCall      int // If > 0, only fail on this call number (1-indexed)
	writeCallCount      int
	lastWriteKey        string
	lastWriteData       []byte
	lastWriteGeneration int64
	allWriteGenerations []int64 // Track all generations used in writes
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	m.lastReadKey = key
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	// Return written data if available, otherwise return initial data
	if m.lastWriteData != nil {
		return m.lastWriteData, m.generation, nil
	}
	return m.data, m.generation, nil
}

func (m *mockStorage) Write(ctx context.Context, key string, data []byte, expectedGeneration int64) error {
	m.writeCallCount++
	m.lastWriteKey = key
	m.lastWriteData = data
	m.lastWriteGeneration = expectedGeneration
	m.allWriteGenerations = append(m.allWriteGenerations, expectedGeneration)
	// If writeErrOnCall is set, only fail on that specific call
	if m.writeErrOnCall > 0 {
		if m.writeCallCount == m.writeErrOnCall {
			return m.writeErr
		}
		return nil
	}
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
