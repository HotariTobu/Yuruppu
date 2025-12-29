package yuruppu

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/agent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	historyPkg "yuruppu/internal/history"
)

// TestSystemPrompt_NotEmpty verifies systemPrompt is embedded and non-empty.
func TestSystemPrompt_NotEmpty(t *testing.T) {
	if systemPrompt == "" {
		t.Fatal("systemPrompt should not be empty - check that prompt/system.txt exists and contains content")
	}
}

// =============================================================================
// Yuruppu Wrapper Tests
// =============================================================================

func TestYuruppu_New(t *testing.T) {
	t.Run("creates Yuruppu with agent", func(t *testing.T) {
		mockAgent := &mockAgent{}
		logger := slog.New(slog.DiscardHandler)

		yuruppu, err := New(mockAgent, logger)

		require.NoError(t, err)
		require.NotNil(t, yuruppu)
		assert.NotNil(t, yuruppu.agent)
		assert.True(t, mockAgent.configureCalled)
	})

	t.Run("creates Yuruppu with nil logger", func(t *testing.T) {
		mockAgent := &mockAgent{}

		yuruppu, err := New(mockAgent, nil)

		require.NoError(t, err)
		require.NotNil(t, yuruppu)
	})

	t.Run("returns error when Configure fails", func(t *testing.T) {
		mockAgent := &mockAgent{
			configureErr: errors.New("configure failed"),
		}
		logger := slog.New(slog.DiscardHandler)

		yuruppu, err := New(mockAgent, logger)

		require.Error(t, err)
		assert.Nil(t, yuruppu)
		assert.Equal(t, "configure failed", err.Error())
	})
}

func TestYuruppu_Respond(t *testing.T) {
	t.Run("delegates to agent successfully", func(t *testing.T) {
		mockAgent := &mockAgent{
			response: "Hello from Yuruppu!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		response, err := yuruppu.Respond(ctx, "Hello", nil)

		require.NoError(t, err)
		assert.Equal(t, "Hello from Yuruppu!", response)
	})

	t.Run("returns error from agent", func(t *testing.T) {
		mockAgent := &mockAgent{
			generateTextErr: errors.New("LLM error"),
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = yuruppu.Respond(ctx, "Hello", nil)

		require.Error(t, err)
		assert.Equal(t, "LLM error", err.Error())
	})

	t.Run("passes history to agent (FR-002)", func(t *testing.T) {
		mockAgent := &mockAgent{
			response: "I remember you, Taro!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppuBot, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		history := []historyPkg.Message{
			{Role: "user", Content: "My name is Taro"},
			{Role: "assistant", Content: "Nice to meet you!"},
		}
		response, err := yuruppuBot.Respond(ctx, "Do you remember me?", history)

		require.NoError(t, err)
		assert.Equal(t, "I remember you, Taro!", response)
		// Verify history was converted to agent.Message and passed to agent
		require.Len(t, mockAgent.lastHistory, 2)
		assert.Equal(t, "My name is Taro", mockAgent.lastHistory[0].Content)
		assert.Equal(t, "user", mockAgent.lastHistory[0].Role)
	})

	t.Run("works with nil history", func(t *testing.T) {
		mockAgent := &mockAgent{
			response: "Hello!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		response, err := yuruppu.Respond(ctx, "Hi", nil)

		require.NoError(t, err)
		assert.Equal(t, "Hello!", response)
		assert.Nil(t, mockAgent.lastHistory)
	})
}

func TestYuruppu_Close(t *testing.T) {
	t.Run("delegates to agent successfully", func(t *testing.T) {
		mockAgent := &mockAgent{}
		logger := slog.New(slog.DiscardHandler)
		yuruppu, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		err = yuruppu.Close(ctx)

		require.NoError(t, err)
		assert.True(t, mockAgent.closeCalled)
	})

	t.Run("returns error from agent", func(t *testing.T) {
		mockAgent := &mockAgent{
			closeErr: errors.New("close failed"),
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu, err := New(mockAgent, logger)
		require.NoError(t, err)

		ctx := context.Background()
		err = yuruppu.Close(ctx)

		require.Error(t, err)
		assert.Equal(t, "close failed", err.Error())
	})
}

// =============================================================================
// Mock Agent for Yuruppu Tests
// =============================================================================

type mockAgent struct {
	response        string
	configureErr    error
	generateTextErr error
	closeErr        error
	configureCalled bool
	closeCalled     bool
	lastHistory     []agent.Message
}

func (m *mockAgent) Configure(ctx context.Context, systemPrompt string) error {
	m.configureCalled = true
	return m.configureErr
}

func (m *mockAgent) GenerateText(ctx context.Context, userMessage string, history []agent.Message) (string, error) {
	m.lastHistory = history
	if m.generateTextErr != nil {
		return "", m.generateTextErr
	}
	return m.response, nil
}

func (m *mockAgent) Close(ctx context.Context) error {
	m.closeCalled = true
	return m.closeErr
}
