package reply_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/toolset/reply"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

func withLineContext(ctx context.Context, replyToken, sourceID string) context.Context {
	ctx = line.WithReplyToken(ctx, replyToken)
	ctx = line.WithSourceID(ctx, sourceID)
	return ctx
}

// =============================================================================
// NewTool Tests
// =============================================================================

func TestNewTool(t *testing.T) {
	t.Run("creates tool with dependencies", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		logger := slog.New(slog.DiscardHandler)

		tool := reply.NewTool(sender, historyRepo, logger)

		require.NotNil(t, tool)
		assert.Equal(t, "reply", tool.Name())
	})
}

// =============================================================================
// Tool Metadata Tests
// =============================================================================

func TestTool_Metadata(t *testing.T) {
	t.Run("Name returns reply", func(t *testing.T) {
		tool := reply.NewTool(&mockSender{}, &mockHistoryRepo{}, slog.New(slog.DiscardHandler))
		assert.Equal(t, "reply", tool.Name())
	})

	t.Run("Description is not empty", func(t *testing.T) {
		tool := reply.NewTool(&mockSender{}, &mockHistoryRepo{}, slog.New(slog.DiscardHandler))
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("ParametersJsonSchema is valid JSON", func(t *testing.T) {
		tool := reply.NewTool(&mockSender{}, &mockHistoryRepo{}, slog.New(slog.DiscardHandler))
		schema := tool.ParametersJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "message")
	})

	t.Run("ResponseJsonSchema is valid JSON", func(t *testing.T) {
		tool := reply.NewTool(&mockSender{}, &mockHistoryRepo{}, slog.New(slog.DiscardHandler))
		schema := tool.ResponseJsonSchema()
		assert.NotEmpty(t, schema)
		assert.Contains(t, string(schema), "status")
	})
}

// =============================================================================
// Callback Tests
// =============================================================================

func TestTool_Callback(t *testing.T) {
	t.Run("success - sends reply and saves history", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"status": "sent"}, result)
		assert.Equal(t, "reply-token", sender.lastReplyToken)
		assert.Equal(t, "Hello!", sender.lastText)
		assert.Equal(t, 1, historyRepo.putCount)
	})

	t.Run("error - invalid message (missing)", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid message")
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("error - invalid message (empty string)", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid message")
	})

	t.Run("error - reply token not in context", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		// Only set sourceID, not replyToken
		ctx := line.WithSourceID(t.Context(), "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("error - source ID not in context", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		// Only set replyToken, not sourceID
		ctx := line.WithReplyToken(t.Context(), "reply-token")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("error - history load fails", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{
			getErr: errors.New("storage error"),
		}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to load conversation")
		assert.Equal(t, 0, sender.callCount)
	})

	t.Run("error - send reply fails", func(t *testing.T) {
		sender := &mockSender{
			err: errors.New("LINE API error"),
		}
		historyRepo := &mockHistoryRepo{}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to send reply")
		assert.Equal(t, 0, historyRepo.putCount)
	})

	t.Run("error - history save fails", func(t *testing.T) {
		sender := &mockSender{}
		historyRepo := &mockHistoryRepo{
			putErr: errors.New("storage error"),
		}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		result, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to save message")
		// Reply was sent before save failed
		assert.Equal(t, 1, sender.callCount)
	})

	t.Run("appends assistant message to existing history", func(t *testing.T) {
		sender := &mockSender{}
		existingHistory := []history.Message{
			&history.UserMessage{
				UserID: "user-1",
				Parts:  []history.UserPart{&history.UserTextPart{Text: "Hi"}},
			},
		}
		historyRepo := &mockHistoryRepo{
			history: existingHistory,
		}
		tool := reply.NewTool(sender, historyRepo, slog.New(slog.DiscardHandler))

		ctx := withLineContext(t.Context(), "reply-token", "source-123")
		_, err := tool.Callback(ctx, map[string]any{
			"message": "Hello!",
		})

		require.NoError(t, err)
		// Verify history has both messages
		require.Len(t, historyRepo.lastPutMessages, 2)
		// First message is user message
		userMsg, ok := historyRepo.lastPutMessages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "user-1", userMsg.UserID)
		// Second message is assistant message
		assistantMsg, ok := historyRepo.lastPutMessages[1].(*history.AssistantMessage)
		require.True(t, ok)
		require.Len(t, assistantMsg.Parts, 1)
		textPart, ok := assistantMsg.Parts[0].(*history.AssistantTextPart)
		require.True(t, ok)
		assert.Equal(t, "Hello!", textPart.Text)
	})
}

// =============================================================================
// Mocks
// =============================================================================

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

type mockHistoryRepo struct {
	history         []history.Message
	generation      int64
	getErr          error
	putErr          error
	getCount        int
	putCount        int
	lastPutMessages []history.Message
}

func (m *mockHistoryRepo) GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error) {
	m.getCount++
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	return m.history, m.generation, nil
}

func (m *mockHistoryRepo) PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error) {
	m.putCount++
	m.lastPutMessages = messages
	if m.putErr != nil {
		return 0, m.putErr
	}
	return m.generation + 1, nil
}
