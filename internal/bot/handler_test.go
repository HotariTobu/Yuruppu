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

		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mediaStor, mockAg, sender, logger)

		require.NoError(t, err)
		require.NotNil(t, h)
	})

	t.Run("returns error when historyRepo is nil", func(t *testing.T) {
		h, err := bot.NewHandler(nil, &mockMediaDownloader{}, &mockStorage{}, &mockAgent{}, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "historyRepo is required")
	})

	t.Run("returns error when mediaDownloader is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, nil, &mockStorage{}, &mockAgent{}, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "mediaDownloader is required")
	})

	t.Run("returns error when mediaStorage is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, nil, &mockAgent{}, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "mediaStorage is required")
	})

	t.Run("returns error when agent is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, &mockStorage{}, nil, &mockSender{}, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("returns error when sender is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, &mockStorage{}, &mockAgent{}, nil, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "sender is required")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		historyRepo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, &mockStorage{}, &mockAgent{}, &mockSender{}, nil)

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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "LLM failed")
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("sender error - returns error", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "LINE API failed")
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleSticker(t.Context(), msgCtx, "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a sticker]", mockAg.lastUserMessageText)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleVideo(t.Context(), msgCtx, "msg-789")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a video]", mockAg.lastUserMessageText)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleAudio(t.Context(), msgCtx, "msg-101")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an audio]", mockAg.lastUserMessageText)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleLocation(t.Context(), msgCtx, 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a location]", mockAg.lastUserMessageText)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleUnknown(t.Context(), msgCtx)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a message]", mockAg.lastUserMessageText)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
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
		mockStore.writeResults = []writeResult{{gen: 0, err: errors.New("GCS failed")}}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
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
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
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

	t.Run("returns error when assistant message save fails after reply sent", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.writeResults = []writeResult{
			{gen: 1, err: nil},                      // user message save succeeds
			{gen: 0, err: errors.New("GCS failed")}, // assistant message save fails
		}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write history")
		// Reply was already sent before assistant message save
		assert.Equal(t, 1, sender.callCount)
		assert.Equal(t, "Hello!", sender.lastText)
		// Both writes were attempted
		assert.Equal(t, 2, mockStore.writeCallCount)
		// History only contains user message (assistant not saved)
		hist, _, _ := historyRepo.GetHistory(t.Context(), "user-123")
		assert.Len(t, hist, 1)
	})
}

// =============================================================================
// Error Chain Tests (errors.Is verification)
// =============================================================================

func TestHandler_ErrorChain(t *testing.T) {
	t.Run("agent error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		agentErr := errors.New("LLM generation failed")
		mockAg := &mockAgent{err: agentErr}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, agentErr), "error chain should contain original agent error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to generate response")
	})

	t.Run("sender error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		senderErr := errors.New("LINE API connection refused")
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{err: senderErr}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, senderErr), "error chain should contain original sender error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to send reply")
	})

	t.Run("storage read error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS bucket not found")
		mockStore.readErr = storageErr
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to load history")
	})

	t.Run("storage write error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS write quota exceeded")
		mockStore.writeResults = []writeResult{{gen: 0, err: storageErr}}
		mockAg := &mockAgent{response: "Hello!"}
		sender := &mockSender{}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(historyRepo, &mockMediaDownloader{}, mockStore, mockAg, sender, logger)
		require.NoError(t, err)

		msgCtx := line.MessageContext{
			ReplyToken: "reply-token",
			SourceID:   "user-123",
			UserID:     "user-123",
		}
		err = h.HandleText(t.Context(), msgCtx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to save user message to history")
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

func (m *mockAgent) Generate(ctx context.Context, hist []agent.Message) (*agent.AssistantMessage, error) {
	// Extract text from last user message in history for testing
	if len(hist) > 0 {
		if userMsg, ok := hist[len(hist)-1].(*agent.UserMessage); ok && len(userMsg.Parts) > 0 {
			if textPart, ok := userMsg.Parts[0].(*agent.UserTextPart); ok {
				m.lastUserMessageText = textPart.Text
			}
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

type mockMediaDownloader struct {
	data          []byte
	mimeType      string
	err           error
	lastMessageID string
}

func (m *mockMediaDownloader) GetMessageContent(messageID string) ([]byte, string, error) {
	m.lastMessageID = messageID
	if m.err != nil {
		return nil, "", m.err
	}
	return m.data, m.mimeType, nil
}

// writeResult represents a single Write call result
type writeResult struct {
	gen int64
	err error
}

// writeRecord represents a recorded Write call
type writeRecord struct {
	key      string
	mimeType string
	data     []byte
}

// mockStorage implements storage.Storage interface
type mockStorage struct {
	// Read behavior
	data          map[string][]byte
	generation    map[string]int64
	readErr       error
	readCallCount int

	// Write behavior
	writeResults         []writeResult
	writeCallCount       int
	writes               []writeRecord
	lastWriteKey         string
	lastWriteMIMEType    string
	lastWriteData        []byte
	lastWriteExpectedGen int64
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

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	m.writeCallCount++
	m.writes = append(m.writes, writeRecord{key: key, mimeType: mimetype, data: data})
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimetype
	m.lastWriteData = data
	m.lastWriteExpectedGen = expectedGeneration

	if len(m.writeResults) > 0 {
		r := m.writeResults[0]
		m.writeResults = m.writeResults[1:]
		if r.err != nil {
			return 0, r.err
		}
		m.data[key] = data
		m.generation[key] = r.gen
		return r.gen, nil
	}
	m.data[key] = data
	newGen := expectedGeneration + 1
	m.generation[key] = newGen
	return newGen, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + key, nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
