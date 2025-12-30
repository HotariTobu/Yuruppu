package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks
var (
	_ agent.Agent  = (*mockAgent)(nil)
	_ bot.Sender   = (*mockSender)(nil)
	_ line.Handler = (*bot.Handler)(nil)
)

// =============================================================================
// NewHandler Tests
// =============================================================================

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		mockAg := &mockAgent{}
		sender := &mockSender{}
		mediaStor := &mockStorage{}
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(historyRepo, mediaStor, mockAg, sender, logger)

		require.NoError(t, err)
		require.NotNil(t, h)
	})

	t.Run("returns error when historyRepo is nil", func(t *testing.T) {
		h, err := bot.NewHandler(nil, &mockStorage{}, &mockAgent{}, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "historyRepo is required")
	})

	t.Run("returns error when mediaStorage is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, nil, &mockAgent{}, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "mediaStorage is required")
	})

	t.Run("returns error when agent is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockStorage{}, nil, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("returns error when sender is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockStorage{}, &mockAgent{}, nil, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "sender is required")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockStorage{}, &mockAgent{}, &mockSender{}, nil)

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "logger is required")
	})
}

// =============================================================================
// Handle* Method Tests
// =============================================================================

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - generates response and sends reply", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.NoError(t, err)
		assert.Equal(t, "reply-token", sender.lastReplyToken)
		assert.Equal(t, "Hello!", sender.lastText)
		assert.Equal(t, 1, sender.callCount)
	})

	t.Run("agent error - returns error", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Equal(t, "LLM failed", err.Error())
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("sender error - returns error", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Equal(t, "LINE API failed", err.Error())
	})
}

func TestHandler_HandleImage(t *testing.T) {
	t.Run("converts image to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I see an image!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleImage(t.Context(), msgCtx, "msg-456")

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent an image]")
	})
}

func TestHandler_HandleSticker(t *testing.T) {
	t.Run("converts sticker to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice sticker!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleSticker(t.Context(), msgCtx, "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent a sticker]")
	})
}

func TestHandler_HandleVideo(t *testing.T) {
	t.Run("converts video to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I see a video!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleVideo(t.Context(), msgCtx, "msg-789")

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent a video]")
	})
}

func TestHandler_HandleAudio(t *testing.T) {
	t.Run("converts audio to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I hear audio!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleAudio(t.Context(), msgCtx, "msg-101")

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent an audio]")
	})
}

func TestHandler_HandleLocation(t *testing.T) {
	t.Run("converts location to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice place!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleLocation(t.Context(), msgCtx, 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent a location]")
	})
}

func TestHandler_HandleUnknown(t *testing.T) {
	t.Run("converts unknown message to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I got your message!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleUnknown(t.Context(), msgCtx)

		require.NoError(t, err)
		assert.Contains(t, mockAg.lastUserMessageText, "[User sent a message]")
	})
}

// =============================================================================
// History Integration Tests
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message and bot response to history", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.NoError(t, err)
		// Verify storage was called twice (user message, then assistant message)
		require.Equal(t, 2, mockStore.writeCallCount)
	})

	t.Run("does not respond when history read fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.readErr = errors.New("GCS read failed")
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read history")
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("does not respond when user message storage fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.writeErr = errors.New("GCS failed")
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write history")
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("saves only user message when agent fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		// User message is saved before agent is called
		assert.Equal(t, 1, mockStore.writeCallCount)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockAgent struct {
	response            string
	err                 error
	lastUserMessageText string
}

func (m *mockAgent) Generate(ctx context.Context, hist []agent.Message, userMessage *agent.UserMessage) (*agent.AssistantMessage, error) {
	// Extract text from user message for testing
	if len(userMessage.Parts) > 0 {
		if textPart, ok := userMessage.Parts[0].(*agent.UserTextPart); ok {
			m.lastUserMessageText = textPart.Text
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return &agent.AssistantMessage{
		ModelName: "test-model",
		Parts:     []agent.AssistantPart{&agent.AssistantTextPart{Text: m.response}},
		LocalTime: time.Now().Format(time.RFC3339),
	}, nil
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
	data          map[string][]byte
	generation    map[string]int64
	readErr       error
	readCallCount int

	// Write behavior
	writeErr       error
	writeCallCount int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: make(map[string]int64),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	data, exists := m.data[key]
	if !exists {
		return nil, 0, nil
	}
	return data, m.generation[key], nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) error {
	m.writeCallCount++
	if m.writeErr != nil {
		return m.writeErr
	}
	m.data[key] = data
	m.generation[key] = expectedGeneration + 1
	return nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + key, nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
