package yuruppu

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/llm"

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
	t.Run("creates Yuruppu with provider", func(t *testing.T) {
		mockProvider := &mockProvider{}
		logger := slog.New(slog.DiscardHandler)

		yuruppu := New(mockProvider, time.Hour, logger)

		require.NotNil(t, yuruppu)
		assert.NotNil(t, yuruppu.agent)
	})

	t.Run("creates Yuruppu with nil logger", func(t *testing.T) {
		mockProvider := &mockProvider{}

		yuruppu := New(mockProvider, time.Hour, nil)

		require.NotNil(t, yuruppu)
	})
}

func TestYuruppu_Respond(t *testing.T) {
	t.Run("delegates to agent successfully", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
			response:  "Hello from Yuruppu!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		response, err := yuruppu.Respond(ctx, "Hello", nil)

		require.NoError(t, err)
		assert.Equal(t, "Hello from Yuruppu!", response)
	})

	t.Run("returns error from agent", func(t *testing.T) {
		mockProvider := &mockProvider{
			createCacheErr:  errors.New("cache failed"),
			generateTextErr: errors.New("LLM error"),
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		_, err := yuruppu.Respond(ctx, "Hello", nil)

		require.Error(t, err)
		assert.Equal(t, "LLM error", err.Error())
	})

	t.Run("passes history to agent (FR-002)", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
			response:  "I remember you, Taro!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppuBot := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		history := []historyPkg.Message{
			{Role: "user", Content: "My name is Taro"},
			{Role: "assistant", Content: "Nice to meet you!"},
		}
		response, err := yuruppuBot.Respond(ctx, "Do you remember me?", history)

		require.NoError(t, err)
		assert.Equal(t, "I remember you, Taro!", response)
		// Verify history was converted to llm.Message and passed to provider
		require.Len(t, mockProvider.lastHistory, 2)
		assert.Equal(t, "My name is Taro", mockProvider.lastHistory[0].Content)
		assert.Equal(t, "user", mockProvider.lastHistory[0].Role)
	})

	t.Run("works with nil history", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
			response:  "Hello!",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		response, err := yuruppu.Respond(ctx, "Hi", nil)

		require.NoError(t, err)
		assert.Equal(t, "Hello!", response)
		assert.Nil(t, mockProvider.lastHistory)
	})
}

func TestYuruppu_Close(t *testing.T) {
	t.Run("delegates to agent successfully", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		err := yuruppu.Close(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls)
	})

	t.Run("handles close error gracefully", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:      "cache-123",
			deleteCacheErr: errors.New("delete failed"),
		}
		logger := slog.New(slog.DiscardHandler)
		yuruppu := New(mockProvider, time.Hour, logger)

		ctx := context.Background()
		err := yuruppu.Close(ctx)

		// Agent logs error but returns nil
		require.NoError(t, err)
	})
}

// =============================================================================
// Mock Provider for Yuruppu Tests
// =============================================================================

type mockProvider struct {
	response         string
	cacheName        string
	createCacheErr   error
	deleteCacheErr   error
	generateTextErr  error
	deleteCacheCalls int
	lastHistory      []llm.Message
}

func (m *mockProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string, history []llm.Message) (string, error) {
	m.lastHistory = history
	if m.generateTextErr != nil {
		return "", m.generateTextErr
	}
	return m.response, nil
}

func (m *mockProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string, history []llm.Message) (string, error) {
	m.lastHistory = history
	return m.response, nil
}

func (m *mockProvider) CreateCachedConfig(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.createCacheErr != nil {
		return "", m.createCacheErr
	}
	return m.cacheName, nil
}

func (m *mockProvider) DeleteCachedConfig(ctx context.Context, cacheName string) error {
	m.deleteCacheCalls++
	return m.deleteCacheErr
}

func (m *mockProvider) Close(ctx context.Context) error {
	return nil
}
