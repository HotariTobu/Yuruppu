package yuruppu_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"yuruppu/internal/yuruppu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// mockLLMProvider is a test mock for LLMProvider.
type mockLLMProvider struct {
	response            string
	err                 error
	capturedSystemPrompt string
	capturedUserMessage  string
}

func (m *mockLLMProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	m.capturedSystemPrompt = systemPrompt
	m.capturedUserMessage = userMessage
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// mockReplier is a test mock for Replier.
type mockReplier struct {
	replyToken string
	text       string
	err        error
	called     bool
}

func (m *mockReplier) SendReply(replyToken string, text string) error {
	m.called = true
	m.replyToken = replyToken
	m.text = text
	return m.err
}

// =============================================================================
// Handler Creation Tests
// =============================================================================

// TestNewHandler_CreatesHandler tests Handler creation.
func TestNewHandler_CreatesHandler(t *testing.T) {
	llm := &mockLLMProvider{}
	client := &mockReplier{}
	logger := discardLogger()

	handler := yuruppu.NewHandler(llm, client, logger)

	assert.NotNil(t, handler, "handler should not be nil")
}

// =============================================================================
// Message Handling Tests
// =============================================================================

// TestHandler_HandleMessage_Success tests successful message handling.
// AC-003: Handler receives message, calls LLM, sends reply.
func TestHandler_HandleMessage_Success(t *testing.T) {
	llm := &mockLLMProvider{response: "Hello from Yuruppu!"}
	client := &mockReplier{}
	handler := yuruppu.NewHandler(llm, client, discardLogger())

	msg := yuruppu.Message{
		ReplyToken: "test-reply-token",
		Type:       "text",
		Text:       "Hello",
		UserID:     "user123",
	}

	err := handler.HandleMessage(context.Background(), msg)

	require.NoError(t, err)
	assert.True(t, client.called, "SendReply should be called")
	assert.Equal(t, "test-reply-token", client.replyToken, "reply token should match")
	assert.Equal(t, "Hello from Yuruppu!", client.text, "reply text should be LLM response")
	assert.Equal(t, "Hello", llm.capturedUserMessage, "user message should be passed to LLM")
}

// TestHandler_HandleMessage_NonTextMessage tests handling of non-text messages.
// FR-008: For non-text messages, use format "[User sent a {type}]"
func TestHandler_HandleMessage_NonTextMessage(t *testing.T) {
	tests := []struct {
		name           string
		messageType    string
		expectedPrompt string
	}{
		{
			name:           "image message",
			messageType:    "image",
			expectedPrompt: "[User sent an image]",
		},
		{
			name:           "sticker message",
			messageType:    "sticker",
			expectedPrompt: "[User sent a sticker]",
		},
		{
			name:           "video message",
			messageType:    "video",
			expectedPrompt: "[User sent a video]",
		},
		{
			name:           "audio message",
			messageType:    "audio",
			expectedPrompt: "[User sent an audio]",
		},
		{
			name:           "location message",
			messageType:    "location",
			expectedPrompt: "[User sent a location]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLMProvider{response: "Response"}
			client := &mockReplier{}
			handler := yuruppu.NewHandler(llm, client, discardLogger())

			msg := yuruppu.Message{
				ReplyToken: "test-token",
				Type:       tt.messageType,
				Text:       "", // Non-text messages have empty text
				UserID:     "user123",
			}

			err := handler.HandleMessage(context.Background(), msg)

			require.NoError(t, err)
			assert.True(t, client.called, "SendReply should be called")
			assert.Equal(t, tt.expectedPrompt, llm.capturedUserMessage,
				"user message should be formatted for non-text type")
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

// TestHandler_HandleMessage_LLMError tests error handling when LLM fails.
// AC-008: On LLM error, error is returned, no reply is sent.
func TestHandler_HandleMessage_LLMError(t *testing.T) {
	llmErr := errors.New("LLM service unavailable")
	llm := &mockLLMProvider{err: llmErr}
	client := &mockReplier{}
	handler := yuruppu.NewHandler(llm, client, discardLogger())

	msg := yuruppu.Message{
		ReplyToken: "test-reply-token",
		Type:       "text",
		Text:       "Hello",
		UserID:     "user123",
	}

	err := handler.HandleMessage(context.Background(), msg)

	require.Error(t, err)
	assert.Equal(t, llmErr, err, "should return LLM error")
	assert.False(t, client.called, "SendReply should not be called on LLM error")
}

// TestHandler_HandleMessage_SendReplyError tests error handling when reply fails.
func TestHandler_HandleMessage_SendReplyError(t *testing.T) {
	replyErr := errors.New("reply token expired")
	llm := &mockLLMProvider{response: "Hello!"}
	client := &mockReplier{err: replyErr}
	handler := yuruppu.NewHandler(llm, client, discardLogger())

	msg := yuruppu.Message{
		ReplyToken: "expired-token",
		Type:       "text",
		Text:       "Hello",
		UserID:     "user123",
	}

	err := handler.HandleMessage(context.Background(), msg)

	require.Error(t, err)
	assert.Equal(t, replyErr, err, "should return reply error")
	assert.True(t, client.called, "SendReply should be called")
}

// TestHandler_HandleMessage_ContextCancellation tests context cancellation.
func TestHandler_HandleMessage_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	llm := &mockLLMProvider{err: context.Canceled}
	client := &mockReplier{}
	handler := yuruppu.NewHandler(llm, client, discardLogger())

	msg := yuruppu.Message{
		ReplyToken: "test-token",
		Type:       "text",
		Text:       "Hello",
		UserID:     "user123",
	}

	err := handler.HandleMessage(ctx, msg)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// =============================================================================
// SystemPrompt Tests
// =============================================================================

// TestSystemPrompt_Exists verifies SystemPrompt constant exists and is non-empty.
func TestSystemPrompt_Exists(t *testing.T) {
	assert.NotEmpty(t, yuruppu.SystemPrompt, "SystemPrompt should not be empty")
	assert.Contains(t, yuruppu.SystemPrompt, "Yuruppu", "SystemPrompt should mention Yuruppu")
}
