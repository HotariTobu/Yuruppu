package agent_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
	"yuruppu/internal/agent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// New Constructor Tests
// =============================================================================

func TestNew_Constructor(t *testing.T) {
	t.Run("New creates Agent with successful cache creation", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		systemPrompt := "You are helpful"

		a := agent.New(mockProvider, systemPrompt, time.Hour, logger)

		require.NotNil(t, a)
		assert.Equal(t, 1, mockProvider.createCacheCalls)
	})

	t.Run("New creates Agent with failed cache creation (fallback mode)", func(t *testing.T) {
		mockProvider := &mockProvider{
			createCacheErr: errors.New("insufficient tokens"),
		}
		logger := slog.Default()
		systemPrompt := "Short prompt"

		a := agent.New(mockProvider, systemPrompt, time.Hour, logger)

		require.NotNil(t, a)
		assert.Equal(t, 1, mockProvider.createCacheCalls)
	})

	t.Run("New accepts nil logger", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
		}

		a := agent.New(mockProvider, "System prompt", time.Hour, nil)

		require.NotNil(t, a)
	})
}

// =============================================================================
// Cache Initialization Tests
// =============================================================================

func TestAgent_CacheCreation(t *testing.T) {
	t.Run("creates cache during initialization", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-initial",
		}
		logger := slog.Default()
		systemPrompt := "System prompt"

		a := agent.New(mockProvider, systemPrompt, time.Hour, logger)

		require.NotNil(t, a)
		assert.Equal(t, 1, mockProvider.createCacheCalls)
		assert.Equal(t, systemPrompt, mockProvider.lastCreateCachePrompt)
	})

	t.Run("operates in fallback mode when initial cache creation fails", func(t *testing.T) {
		mockProvider := &mockProvider{
			createCacheErr: errors.New("API error"),
			response:       "Response without cache",
		}
		logger := slog.Default()

		a := agent.New(mockProvider, "System prompt", time.Hour, logger)
		require.NotNil(t, a)

		ctx := context.Background()
		response, err := a.GenerateText(ctx, "user message")

		require.NoError(t, err)
		assert.Equal(t, "Response without cache", response)
		assert.Equal(t, 1, mockProvider.generateTextCalls)
		assert.Equal(t, 0, mockProvider.generateTextCachedCalls)
	})
}

// =============================================================================
// GenerateText Cache Usage Tests
// =============================================================================

func TestAgent_GenerateText_CachePath(t *testing.T) {
	t.Run("uses cached path when cache exists", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-success",
			response:  "Cached response",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		response, err := a.GenerateText(ctx, "user message")

		require.NoError(t, err)
		assert.Equal(t, "Cached response", response)
		assert.Equal(t, 1, mockProvider.generateTextCachedCalls)
		assert.Equal(t, 0, mockProvider.generateTextCalls)
		assert.Equal(t, "cache-success", mockProvider.lastUsedCacheName)
	})

	t.Run("uses non-cached path when no cache (fallback mode)", func(t *testing.T) {
		mockProvider := &mockProvider{
			createCacheErr: errors.New("cache creation failed"),
			response:       "Non-cached response",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		response, err := a.GenerateText(ctx, "user message")

		require.NoError(t, err)
		assert.Equal(t, "Non-cached response", response)
		assert.Equal(t, 1, mockProvider.generateTextCalls)
		assert.Equal(t, 0, mockProvider.generateTextCachedCalls)
	})
}

// =============================================================================
// Cache Error Handling and Recreation Tests
// =============================================================================

func TestAgent_CacheErrorRecreation(t *testing.T) {
	t.Run("cache error triggers recreation", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 1,
			response:                "Response after recreation",
			recreatedCacheName:      "cache-recreated",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		response, err := a.GenerateText(ctx, "user message")

		require.NoError(t, err)
		assert.Equal(t, "Response after recreation", response)
		assert.Equal(t, 2, mockProvider.createCacheCalls)
	})

	t.Run("recreation failure falls back to non-cached mode", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 1,
			createCacheErr:          errors.New("recreation failed"),
			response:                "Fallback response",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		mockProvider.createCacheErrOnRecreate = true

		ctx := context.Background()
		response, err := a.GenerateText(ctx, "user message")

		require.NoError(t, err)
		assert.Equal(t, "Fallback response", response)
	})
}

// =============================================================================
// Concurrent Cache Recreation Tests
// =============================================================================

func TestAgent_ConcurrentCacheRecreation(t *testing.T) {
	t.Run("concurrent cache recreation attempts are serialized", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 10,
			recreatedCacheName:      "cache-new",
			response:                "Response",
			recreationDelay:         50 * time.Millisecond,
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		concurrency := 5
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for range concurrency {
			go func() {
				defer wg.Done()
				_, _ = a.GenerateText(ctx, "concurrent message")
			}()
		}

		wg.Wait()

		assert.LessOrEqual(t, mockProvider.createCacheCalls, concurrency+1)
	})
}

// =============================================================================
// Close Method Tests
// =============================================================================

func TestAgent_Close(t *testing.T) {
	t.Run("Close deletes cache successfully", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-to-delete",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		err := a.Close(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls)
		assert.Equal(t, "cache-to-delete", mockProvider.lastDeletedCacheName)
	})

	t.Run("Close handles cache deletion failure gracefully", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:      "cache-delete-fails",
			deleteCacheErr: errors.New("cache deletion failed"),
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		err := a.Close(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls)
	})

	t.Run("Close does not close Provider", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()
		err := a.Close(ctx)

		require.NoError(t, err)
		assert.False(t, mockProvider.closed)
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()

		err1 := a.Close(ctx)
		err2 := a.Close(ctx)
		err3 := a.Close(ctx)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})
}

// =============================================================================
// Close State Tests
// =============================================================================

func TestAgent_GenerateTextAfterClose(t *testing.T) {
	t.Run("GenerateText returns ClosedError after Close", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName: "cache-123",
			response:  "Response",
		}
		logger := slog.Default()
		a := agent.New(mockProvider, "System prompt", time.Hour, logger)

		ctx := context.Background()

		response, err := a.GenerateText(ctx, "message before close")
		require.NoError(t, err)
		assert.Equal(t, "Response", response)

		err = a.Close(ctx)
		require.NoError(t, err)

		response, err = a.GenerateText(ctx, "message after close")
		require.Error(t, err)
		assert.Empty(t, response)

		var closedErr *agent.ClosedError
		assert.ErrorAs(t, err, &closedErr)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestAgent_FullLifecycle(t *testing.T) {
	t.Run("complete lifecycle: create, use, recreate, close", func(t *testing.T) {
		mockProvider := &mockProvider{
			cacheName:          "cache-initial",
			response:           "Response",
			recreatedCacheName: "cache-new",
		}
		logger := slog.Default()
		systemPrompt := "System prompt"

		a := agent.New(mockProvider, systemPrompt, time.Hour, logger)
		require.NotNil(t, a)
		assert.Equal(t, 1, mockProvider.createCacheCalls)

		ctx := context.Background()

		response1, err1 := a.GenerateText(ctx, "message 1")
		require.NoError(t, err1)
		assert.Equal(t, "Response", response1)
		assert.Equal(t, 1, mockProvider.generateTextCachedCalls)

		mockProvider.generateTextCachedErr = errors.New("cache expired")
		mockProvider.generateTextCachedFails = 1

		response2, err2 := a.GenerateText(ctx, "message 2")
		require.NoError(t, err2)
		assert.Equal(t, "Response", response2)
		assert.Equal(t, 2, mockProvider.createCacheCalls)

		mockProvider.generateTextCachedErr = nil
		response3, err3 := a.GenerateText(ctx, "message 3")
		require.NoError(t, err3)
		assert.Equal(t, "Response", response3)

		err := a.Close(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls)

		_, err = a.GenerateText(ctx, "message after close")
		var closedErr *agent.ClosedError
		assert.ErrorAs(t, err, &closedErr)
	})
}

// =============================================================================
// Mock Provider
// =============================================================================

type mockProvider struct {
	response           string
	cacheName          string
	recreatedCacheName string
	checkContext       bool
	recreationDelay    time.Duration

	createCacheErr             error
	createCacheErrOnRecreate   bool
	deleteCacheErr             error
	generateTextErr            error
	generateTextCachedErr      error
	generateTextCachedFails    int
	generateTextCachedFailsNow int

	closed                   bool
	createCacheCalls         int
	deleteCacheCalls         int
	generateTextCalls        int
	generateTextCachedCalls  int
	lastCreateCachePrompt    string
	lastDeletedCacheName     string
	lastUsedCacheName        string
	lastGenerateTextPrompt   string
	lastGenerateTextMessage  string
	recreationInProgress     bool
	recreationInProgressLock sync.Mutex
}

func (m *mockProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	m.generateTextCalls++
	m.lastGenerateTextPrompt = systemPrompt
	m.lastGenerateTextMessage = userMessage

	if m.generateTextErr != nil {
		return "", m.generateTextErr
	}
	return m.response, nil
}

func (m *mockProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	m.generateTextCachedCalls++
	m.lastUsedCacheName = cacheName

	if m.generateTextCachedFails > 0 && m.generateTextCachedFailsNow < m.generateTextCachedFails {
		m.generateTextCachedFailsNow++
		return "", m.generateTextCachedErr
	}

	if m.generateTextCachedErr != nil && m.generateTextCachedFails == 0 {
		return "", m.generateTextCachedErr
	}

	return m.response, nil
}

func (m *mockProvider) CreateCachedConfig(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	if m.recreationDelay > 0 {
		m.recreationInProgressLock.Lock()
		m.recreationInProgress = true
		m.recreationInProgressLock.Unlock()

		time.Sleep(m.recreationDelay)

		m.recreationInProgressLock.Lock()
		m.recreationInProgress = false
		m.recreationInProgressLock.Unlock()
	}

	m.createCacheCalls++
	m.lastCreateCachePrompt = systemPrompt

	if m.createCacheErrOnRecreate && m.createCacheCalls > 1 {
		return "", m.createCacheErr
	}

	if m.createCacheErr != nil && m.createCacheCalls == 1 {
		return "", m.createCacheErr
	}

	if m.createCacheCalls > 1 && m.recreatedCacheName != "" {
		return m.recreatedCacheName, nil
	}

	return m.cacheName, nil
}

func (m *mockProvider) DeleteCachedConfig(ctx context.Context, cacheName string) error {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	m.deleteCacheCalls++
	m.lastDeletedCacheName = cacheName

	if m.deleteCacheErr != nil {
		return m.deleteCacheErr
	}

	return nil
}

func (m *mockProvider) Close(ctx context.Context) error {
	m.closed = true
	return nil
}
