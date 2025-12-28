package bot

import (
	"context"
	"errors"
	"log/slog"
	"testing"
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
		logger := slog.New(slog.DiscardHandler)

		h := New(responder, sender, logger)

		require.NotNil(t, h)
		assert.Equal(t, responder, h.responder)
		assert.Equal(t, sender, h.sender)
		assert.Equal(t, logger, h.logger)
	})

	t.Run("accepts nil logger", func(t *testing.T) {
		h := New(&mockResponder{}, &mockSender{}, nil)

		require.NotNil(t, h)
		assert.Nil(t, h.logger)
	})
}

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - responds and sends reply", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

		err := h.HandleText(context.Background(), "reply-token", "user-123", "Hi")

		require.Error(t, err)
		assert.Equal(t, "LLM failed", err.Error())
		assert.Equal(t, 0, sender.callCount) // sender should not be called
	})

	t.Run("sender error - returns error", func(t *testing.T) {
		responder := &mockResponder{response: "Hello!"}
		sender := &mockSender{err: errors.New("LINE API failed")}
		logger := slog.New(slog.DiscardHandler)
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

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
		h := New(responder, sender, logger)

		err := h.HandleUnknown(context.Background(), "reply-token", "user-123")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a message]", responder.lastMessage)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockResponder struct {
	response    string
	err         error
	lastMessage string
}

func (m *mockResponder) Respond(ctx context.Context, userMessage string) (string, error) {
	m.lastMessage = userMessage
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
