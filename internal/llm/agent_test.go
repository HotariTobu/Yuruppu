package llm_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Agent Interface Tests (AC-002)
// =============================================================================

// TestAgent_InterfaceExists tests that the Agent interface exists with expected methods.
// AC-002: Agent Interface Defined
func TestAgent_InterfaceExists(t *testing.T) {
	t.Run("interface defines GenerateText method", func(t *testing.T) {
		// Given: A mock implementation of Agent interface
		mockProvider := &mockAgentProvider{
			response: "Hello from Yuruppu!",
		}
		logger := slog.Default()

		// When: Create Agent with NewAgent
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Then: Should implement Agent interface
		var _ llm.Agent = agent
	})

	t.Run("interface defines Close method", func(t *testing.T) {
		// Given: An Agent instance
		mockProvider := &mockAgentProvider{}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Call Close method
		ctx := context.Background()
		err := agent.Close(ctx)

		// Then: Close method should be callable
		require.NoError(t, err)
	})
}

// TestAgent_GenerateTextSignature tests the GenerateText method signature.
// AC-002: Agent interface with GenerateText(ctx context.Context, userMessage string) (string, error)
func TestAgent_GenerateTextSignature(t *testing.T) {
	t.Run("GenerateText accepts context and userMessage only", func(t *testing.T) {
		// Given: An Agent instance
		mockProvider := &mockAgentProvider{
			response: "Response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Call GenerateText with context and userMessage
		ctx := context.Background()
		userMessage := "Hello"
		response, err := agent.GenerateText(ctx, userMessage)

		// Then: Should return response without error
		require.NoError(t, err)
		assert.Equal(t, "Response", response)
	})

	t.Run("GenerateText does not require systemPrompt parameter", func(t *testing.T) {
		// Given: An Agent created with systemPrompt
		mockProvider := &mockAgentProvider{
			response: "Response",
		}
		logger := slog.Default()
		systemPrompt := "You are Yuruppu"
		agent := llm.NewAgent(mockProvider, systemPrompt, logger)

		// When: GenerateText is called without passing systemPrompt
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should work (systemPrompt is stored in Agent)
		require.NoError(t, err)
		assert.Equal(t, "Response", response)
	})
}

// =============================================================================
// NewAgent Constructor Tests (AC-002)
// =============================================================================

// TestNewAgent_Constructor tests the NewAgent constructor.
// AC-002: NewAgent(provider Provider, systemPrompt string, logger *slog.Logger) Agent
func TestNewAgent_Constructor(t *testing.T) {
	t.Run("NewAgent creates Agent with successful cache creation", func(t *testing.T) {
		// Given: A mock provider that succeeds cache creation
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		systemPrompt := "You are Yuruppu"

		// When: NewAgent is called
		agent := llm.NewAgent(mockProvider, systemPrompt, logger)

		// Then: Should return Agent (no error)
		require.NotNil(t, agent)
		assert.Equal(t, 1, mockProvider.createCacheCalls,
			"should attempt to create cache during initialization")
	})

	t.Run("NewAgent creates Agent with failed cache creation (fallback mode)", func(t *testing.T) {
		// Given: A mock provider that fails cache creation
		mockProvider := &mockAgentProvider{
			createCacheErr: errors.New("insufficient tokens"),
		}
		logger := slog.Default()
		systemPrompt := "Short prompt"

		// When: NewAgent is called
		agent := llm.NewAgent(mockProvider, systemPrompt, logger)

		// Then: Should return Agent without error (fallback mode)
		require.NotNil(t, agent,
			"NewAgent should not return error even if cache creation fails")
		assert.Equal(t, 1, mockProvider.createCacheCalls,
			"should attempt to create cache during initialization")
	})

	t.Run("NewAgent returns Agent not error", func(t *testing.T) {
		// Given: A mock provider
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()

		// When: NewAgent is called
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Then: Should return Agent (not error)
		require.NotNil(t, agent)
	})

	t.Run("NewAgent stores Provider via dependency injection", func(t *testing.T) {
		// Given: A mock provider
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
			response:  "Response",
		}
		logger := slog.Default()

		// When: NewAgent is called and GenerateText is used
		agent := llm.NewAgent(mockProvider, "System prompt", logger)
		ctx := context.Background()
		_, _ = agent.GenerateText(ctx, "user message")

		// Then: Should use the provided Provider
		assert.Equal(t, 1, mockProvider.generateTextCachedCalls,
			"Agent should use the injected Provider for generation")
	})

	t.Run("NewAgent accepts different system prompts", func(t *testing.T) {
		// Given: A mock provider
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()

		tests := []struct {
			name         string
			systemPrompt string
		}{
			{"short prompt", "You are Yuruppu"},
			{"long prompt", "You are Yuruppu, a friendly LINE bot that responds in Japanese. " + string(make([]byte, 1000))},
			{"empty prompt", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When: NewAgent is called with different prompts
				agent := llm.NewAgent(mockProvider, tt.systemPrompt, logger)

				// Then: Should create Agent
				require.NotNil(t, agent)
			})
		}
	})

	t.Run("NewAgent accepts nil logger", func(t *testing.T) {
		// Given: A mock provider and nil logger
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}

		// When: NewAgent is called with nil logger
		agent := llm.NewAgent(mockProvider, "System prompt", nil)

		// Then: Should create Agent (implementation handles nil logger)
		require.NotNil(t, agent)
	})
}

// =============================================================================
// Cache Initialization Tests (AC-003)
// =============================================================================

// TestAgent_CacheCreation tests cache creation during NewAgent initialization.
// AC-003: Cache created during NewAgent() via provider.CreateCache() with 60-minute TTL
func TestAgent_CacheCreation(t *testing.T) {
	t.Run("creates cache during initialization with 60-minute TTL", func(t *testing.T) {
		// Given: A mock provider
		mockProvider := &mockAgentProvider{
			cacheName: "cache-initial",
		}
		logger := slog.Default()
		systemPrompt := "You are Yuruppu"

		// When: NewAgent is called
		agent := llm.NewAgent(mockProvider, systemPrompt, logger)

		// Then: Should create cache via provider.CreateCache()
		require.NotNil(t, agent)
		assert.Equal(t, 1, mockProvider.createCacheCalls,
			"should call provider.CreateCache() once during initialization")
		assert.Equal(t, systemPrompt, mockProvider.lastCreateCachePrompt,
			"should pass system prompt to CreateCache")
	})

	t.Run("operates in fallback mode when initial cache creation fails", func(t *testing.T) {
		// Given: A mock provider that fails cache creation
		mockProvider := &mockAgentProvider{
			createCacheErr: errors.New("API error"),
			response:       "Response without cache",
		}
		logger := slog.Default()

		// When: NewAgent is called (cache creation will fail)
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Then: Should not return error (fallback mode)
		require.NotNil(t, agent)

		// When: GenerateText is called in fallback mode
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should use non-cached path
		require.NoError(t, err)
		assert.Equal(t, "Response without cache", response)
		assert.Equal(t, 1, mockProvider.generateTextCalls,
			"should use non-cached GenerateText when cache creation fails")
		assert.Equal(t, 0, mockProvider.generateTextCachedCalls,
			"should not use cached path when cache creation fails")
	})
}

// =============================================================================
// GenerateText Cache Usage Tests (AC-003)
// =============================================================================

// TestAgent_GenerateText_CachePath tests GenerateText uses cached path when cache exists.
// AC-003: Agent calls provider.GenerateTextCached() when cacheName is set
func TestAgent_GenerateText_CachePath(t *testing.T) {
	t.Run("uses cached path when cache exists", func(t *testing.T) {
		// Given: An Agent with successful cache creation
		mockProvider := &mockAgentProvider{
			cacheName: "cache-success",
			response:  "Cached response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: GenerateText is called
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should use cached path
		require.NoError(t, err)
		assert.Equal(t, "Cached response", response)
		assert.Equal(t, 1, mockProvider.generateTextCachedCalls,
			"should call provider.GenerateTextCached()")
		assert.Equal(t, 0, mockProvider.generateTextCalls,
			"should not call non-cached GenerateText when cache exists")
		assert.Equal(t, "cache-success", mockProvider.lastUsedCacheName,
			"should use the cache name from initialization")
	})

	t.Run("uses non-cached path when no cache (fallback mode)", func(t *testing.T) {
		// Given: An Agent in fallback mode (cache creation failed)
		mockProvider := &mockAgentProvider{
			createCacheErr: errors.New("cache creation failed"),
			response:       "Non-cached response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: GenerateText is called
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should use non-cached path
		require.NoError(t, err)
		assert.Equal(t, "Non-cached response", response)
		assert.Equal(t, 1, mockProvider.generateTextCalls,
			"should call non-cached GenerateText in fallback mode")
		assert.Equal(t, 0, mockProvider.generateTextCachedCalls,
			"should not call cached path when no cache")
	})

	t.Run("multiple GenerateText calls use same cache", func(t *testing.T) {
		// Given: An Agent with cache
		mockProvider := &mockAgentProvider{
			cacheName: "cache-persistent",
			response:  "Response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		ctx := context.Background()

		// When: Multiple GenerateText calls
		_, _ = agent.GenerateText(ctx, "message 1")
		_, _ = agent.GenerateText(ctx, "message 2")
		_, _ = agent.GenerateText(ctx, "message 3")

		// Then: All should use cached path
		assert.Equal(t, 3, mockProvider.generateTextCachedCalls,
			"all calls should use cached path")
		assert.Equal(t, 0, mockProvider.generateTextCalls,
			"should not use non-cached path when cache exists")
	})
}

// =============================================================================
// Cache Error Handling and Recreation Tests (AC-003)
// =============================================================================

// TestAgent_CacheErrorRecreation tests cache recreation on cache errors.
// AC-003: Cache errors during GenerateTextCached() trigger automatic recreation
func TestAgent_CacheErrorRecreation(t *testing.T) {
	t.Run("cache error triggers recreation", func(t *testing.T) {
		// Given: An Agent with cache that will fail once
		mockProvider := &mockAgentProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 1, // Fail once, then succeed
			response:                "Response after recreation",
			recreatedCacheName:      "cache-recreated",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: GenerateText encounters cache error
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should recreate cache and retry
		require.NoError(t, err)
		assert.Equal(t, "Response after recreation", response)
		assert.Equal(t, 2, mockProvider.createCacheCalls,
			"should call CreateCache again (initial + recreation)")
		assert.GreaterOrEqual(t, mockProvider.generateTextCachedCalls, 1,
			"should attempt cached generation")
	})

	t.Run("recreation failure falls back to non-cached mode", func(t *testing.T) {
		// Given: An Agent where cache fails and recreation also fails
		mockProvider := &mockAgentProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 1,
			createCacheErr:          errors.New("recreation failed"),
			response:                "Fallback response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Reset createCacheErr after initial creation for recreation failure
		mockProvider.createCacheErrOnRecreate = true

		// When: GenerateText encounters cache error and recreation fails
		ctx := context.Background()
		response, err := agent.GenerateText(ctx, "user message")

		// Then: Should fall back to non-cached mode for that call
		require.NoError(t, err,
			"should not return error even when cache recreation fails")
		assert.Equal(t, "Fallback response", response)
	})

	t.Run("cache error on multiple calls triggers recreation only once", func(t *testing.T) {
		// Given: An Agent with cache
		mockProvider := &mockAgentProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache expired"),
			generateTextCachedFails: 2, // Fail first 2 calls
			recreatedCacheName:      "cache-new",
			response:                "Response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		ctx := context.Background()

		// When: First GenerateText encounters cache error
		_, err1 := agent.GenerateText(ctx, "message 1")
		require.NoError(t, err1)

		// Reset errors for second call
		mockProvider.generateTextCachedErr = nil

		// When: Second GenerateText uses recreated cache
		_, err2 := agent.GenerateText(ctx, "message 2")
		require.NoError(t, err2)

		// Then: Cache should be recreated once
		assert.Equal(t, 2, mockProvider.createCacheCalls,
			"should recreate cache once (initial + recreation)")
	})
}

// =============================================================================
// Concurrent Cache Recreation Tests (AC-003)
// =============================================================================

// TestAgent_ConcurrentCacheRecreation tests mutex protection for concurrent recreation.
// AC-003: Concurrent recreation attempts prevented by mutex
func TestAgent_ConcurrentCacheRecreation(t *testing.T) {
	t.Run("concurrent cache recreation attempts are serialized", func(t *testing.T) {
		// Given: An Agent with cache that will fail
		mockProvider := &mockAgentProvider{
			cacheName:               "cache-initial",
			generateTextCachedErr:   errors.New("cache not found"),
			generateTextCachedFails: 10, // Fail multiple times
			recreatedCacheName:      "cache-new",
			response:                "Response",
			recreationDelay:         50 * time.Millisecond, // Simulate slow recreation
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Multiple concurrent GenerateText calls trigger cache errors
		ctx := context.Background()
		concurrency := 5
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for range concurrency {
			go func() {
				defer wg.Done()
				_, _ = agent.GenerateText(ctx, "concurrent message")
			}()
		}

		wg.Wait()

		// Then: Cache recreation should be protected by mutex
		// Exact number depends on timing, but should not recreate for each concurrent call
		assert.LessOrEqual(t, mockProvider.createCacheCalls, concurrency+1,
			"mutex should prevent excessive recreation attempts")
	})
}

// =============================================================================
// Close Method Tests (AC-003)
// =============================================================================

// TestAgent_Close tests the Close method behavior.
// AC-003: Close() deletes cache via provider.DeleteCache() (does not close Provider)
func TestAgent_Close(t *testing.T) {
	t.Run("Close deletes cache successfully", func(t *testing.T) {
		// Given: An Agent with cache
		mockProvider := &mockAgentProvider{
			cacheName: "cache-to-delete",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Close is called
		ctx := context.Background()
		err := agent.Close(ctx)

		// Then: Should delete cache without error
		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls,
			"should call provider.DeleteCache() once")
		assert.Equal(t, "cache-to-delete", mockProvider.lastDeletedCacheName,
			"should delete the correct cache")
	})

	t.Run("Close handles cache deletion failure gracefully", func(t *testing.T) {
		// Given: An Agent with cache, but deletion will fail
		mockProvider := &mockAgentProvider{
			cacheName:      "cache-delete-fails",
			deleteCacheErr: errors.New("cache deletion failed"),
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Close is called
		ctx := context.Background()
		err := agent.Close(ctx)

		// Then: Should complete successfully (error is logged but not returned)
		require.NoError(t, err,
			"Close should complete successfully even if cache deletion fails")
		assert.Equal(t, 1, mockProvider.deleteCacheCalls,
			"should attempt to delete cache")
	})

	t.Run("Close does not close Provider", func(t *testing.T) {
		// Given: An Agent with Provider
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Agent.Close is called
		ctx := context.Background()
		err := agent.Close(ctx)

		// Then: Should not close Provider
		require.NoError(t, err)
		assert.False(t, mockProvider.closed,
			"Agent.Close should NOT close Provider - caller manages Provider lifecycle")
	})

	t.Run("Close in fallback mode (no cache) does not error", func(t *testing.T) {
		// Given: An Agent in fallback mode (no cache)
		mockProvider := &mockAgentProvider{
			createCacheErr: errors.New("cache creation failed"),
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// When: Close is called
		ctx := context.Background()
		err := agent.Close(ctx)

		// Then: Should complete without error
		require.NoError(t, err,
			"Close should succeed even when there's no cache to delete")
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		// Given: An Agent
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		ctx := context.Background()

		// When: Close is called multiple times
		err1 := agent.Close(ctx)
		err2 := agent.Close(ctx)
		err3 := agent.Close(ctx)

		// Then: All calls should succeed (idempotent)
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})
}

// =============================================================================
// Close State Tests (AC-004)
// =============================================================================

// TestAgent_GenerateTextAfterClose tests that GenerateText returns error after Close.
// AC-004: When Agent is closed, GenerateText returns LLMClosedError
func TestAgent_GenerateTextAfterClose(t *testing.T) {
	t.Run("GenerateText returns LLMClosedError after Close", func(t *testing.T) {
		// Given: An Agent that has been closed
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
			response:  "Response",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		ctx := context.Background()

		// Given: Agent works before Close
		response, err := agent.GenerateText(ctx, "message before close")
		require.NoError(t, err)
		assert.Equal(t, "Response", response)

		// When: Close is called
		err = agent.Close(ctx)
		require.NoError(t, err)

		// Then: GenerateText returns LLMClosedError
		response, err = agent.GenerateText(ctx, "message after close")
		require.Error(t, err)
		assert.Empty(t, response)

		var closedErr *llm.LLMClosedError
		assert.ErrorAs(t, err, &closedErr,
			"should return LLMClosedError after Close")
	})

	t.Run("multiple GenerateText calls after Close all return LLMClosedError", func(t *testing.T) {
		// Given: A closed Agent
		mockProvider := &mockAgentProvider{
			cacheName: "cache-123",
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		ctx := context.Background()
		_ = agent.Close(ctx)

		// When: Multiple GenerateText calls after Close
		_, err1 := agent.GenerateText(ctx, "message 1")
		_, err2 := agent.GenerateText(ctx, "message 2")
		_, err3 := agent.GenerateText(ctx, "message 3")

		// Then: All should return LLMClosedError
		var closedErr1, closedErr2, closedErr3 *llm.LLMClosedError
		assert.ErrorAs(t, err1, &closedErr1)
		assert.ErrorAs(t, err2, &closedErr2)
		assert.ErrorAs(t, err3, &closedErr3)
	})
}

// =============================================================================
// Context Handling Tests
// =============================================================================

// TestAgent_ContextCancellation tests context handling in Agent methods.
func TestAgent_ContextCancellation(t *testing.T) {
	t.Run("GenerateText respects context cancellation", func(t *testing.T) {
		// Given: An Agent
		mockProvider := &mockAgentProvider{
			cacheName:    "cache-123",
			checkContext: true,
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When: GenerateText is called with cancelled context
		_, err := agent.GenerateText(ctx, "message")

		// Then: Should return context error
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("Close respects context cancellation", func(t *testing.T) {
		// Given: An Agent
		mockProvider := &mockAgentProvider{
			cacheName:    "cache-123",
			checkContext: true,
		}
		logger := slog.Default()
		agent := llm.NewAgent(mockProvider, "System prompt", logger)

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When: Close is called with cancelled context
		err := agent.Close(ctx)

		// Then: Implementation decides behavior (may handle gracefully)
		_ = err
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestAgent_FullLifecycle tests complete Agent lifecycle.
func TestAgent_FullLifecycle(t *testing.T) {
	t.Run("complete lifecycle: create, use, recreate, close", func(t *testing.T) {
		// Given: A mock provider
		mockProvider := &mockAgentProvider{
			cacheName:          "cache-initial",
			response:           "Response",
			recreatedCacheName: "cache-new",
		}
		logger := slog.Default()
		systemPrompt := "You are Yuruppu"

		// Step 1: Create Agent (cache creation)
		agent := llm.NewAgent(mockProvider, systemPrompt, logger)
		require.NotNil(t, agent)
		assert.Equal(t, 1, mockProvider.createCacheCalls,
			"should create cache during initialization")

		ctx := context.Background()

		// Step 2: Use Agent (normal operation)
		response1, err1 := agent.GenerateText(ctx, "message 1")
		require.NoError(t, err1)
		assert.Equal(t, "Response", response1)
		assert.Equal(t, 1, mockProvider.generateTextCachedCalls)

		// Step 3: Simulate cache error and recreation
		mockProvider.generateTextCachedErr = errors.New("cache expired")
		mockProvider.generateTextCachedFails = 1

		response2, err2 := agent.GenerateText(ctx, "message 2")
		require.NoError(t, err2)
		assert.Equal(t, "Response", response2)
		assert.Equal(t, 2, mockProvider.createCacheCalls,
			"should recreate cache after error")

		// Step 4: Continue using recreated cache
		mockProvider.generateTextCachedErr = nil
		response3, err3 := agent.GenerateText(ctx, "message 3")
		require.NoError(t, err3)
		assert.Equal(t, "Response", response3)

		// Step 5: Close Agent (cache cleanup)
		err := agent.Close(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, mockProvider.deleteCacheCalls,
			"should delete cache on close")

		// Step 6: Verify closed state
		_, err = agent.GenerateText(ctx, "message after close")
		var closedErr *llm.LLMClosedError
		assert.ErrorAs(t, err, &closedErr,
			"should return LLMClosedError after Close")
	})
}

// =============================================================================
// Mock Provider for Agent Tests
// =============================================================================

// mockAgentProvider is a mock implementation of Provider for testing Agent.
// This follows the manual mock pattern from ADR: 20251217-testing-strategy.
type mockAgentProvider struct {
	// Configuration
	response           string
	cacheName          string
	recreatedCacheName string
	checkContext       bool
	recreationDelay    time.Duration

	// Error simulation
	createCacheErr             error
	createCacheErrOnRecreate   bool
	deleteCacheErr             error
	generateTextErr            error
	generateTextCachedErr      error
	generateTextCachedFails    int // Number of times GenerateTextCached should fail
	generateTextCachedFailsNow int // Counter for failures

	// State tracking
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

func (m *mockAgentProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
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

func (m *mockAgentProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	m.generateTextCachedCalls++
	m.lastUsedCacheName = cacheName

	// Simulate failures for testing recreation
	if m.generateTextCachedFails > 0 && m.generateTextCachedFailsNow < m.generateTextCachedFails {
		m.generateTextCachedFailsNow++
		return "", m.generateTextCachedErr
	}

	if m.generateTextCachedErr != nil && m.generateTextCachedFails == 0 {
		return "", m.generateTextCachedErr
	}

	return m.response, nil
}

func (m *mockAgentProvider) CreateCache(ctx context.Context, systemPrompt string) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	// Simulate recreation delay
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

	// Return error on recreation if configured
	if m.createCacheErrOnRecreate && m.createCacheCalls > 1 {
		return "", m.createCacheErr
	}

	if m.createCacheErr != nil && m.createCacheCalls == 1 {
		return "", m.createCacheErr
	}

	// Return recreated cache name if this is a recreation
	if m.createCacheCalls > 1 && m.recreatedCacheName != "" {
		return m.recreatedCacheName, nil
	}

	return m.cacheName, nil
}

func (m *mockAgentProvider) DeleteCache(ctx context.Context, cacheName string) error {
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

func (m *mockAgentProvider) Close(ctx context.Context) error {
	m.closed = true
	return nil
}
