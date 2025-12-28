package yuruppu

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		response, err := yuruppu.Respond(ctx, "Hello")

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
		_, err := yuruppu.Respond(ctx, "Hello")

		require.Error(t, err)
		assert.Equal(t, "LLM error", err.Error())
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
}

func (m *mockProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.generateTextErr != nil {
		return "", m.generateTextErr
	}
	return m.response, nil
}

func (m *mockProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
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
