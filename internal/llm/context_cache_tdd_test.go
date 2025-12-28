package llm_test

import (
	"context"
	"testing"
	"time"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface conformance check
var _ llm.Provider = (*mockCachedProvider)(nil)

// =============================================================================
// TDD Tests for Provider Cache Methods (AC-001: Pure API Layer)
// =============================================================================
// These tests verify Provider cache methods work as a pure API layer.
// The Provider does NOT manage cache lifecycle - that's the Agent's responsibility.
//
// ADR: 20251228-provider-cache-interface - Separate methods for cached/non-cached calls
// AC-001: Provider is Pure API Layer

// =============================================================================
// Provider Cache API Tests
// =============================================================================

// TestProvider_CreateCache_APILayer tests that CreateCache is a pure API call.
// AC-001: CreateCache creates cache and returns cacheName (no internal storage)
func TestProvider_CreateCache_APILayer(t *testing.T) {
	t.Run("CreateCache is stateless", func(t *testing.T) {
		// Given: A mock provider (pure API layer)
		provider := &mockCachedProvider{
			cacheCounter: 0,
		}

		// When: CreateCache is called multiple times
		ctx := context.Background()
		cache1, err1 := provider.CreateCachedConfig(ctx, "System prompt 1", time.Hour)
		cache2, err2 := provider.CreateCachedConfig(ctx, "System prompt 2", time.Hour)

		// Then: Each call creates a new cache (no internal state)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, cache1, cache2,
			"Provider should not reuse caches - each call creates new cache")
	})
}

// TestProvider_GenerateTextCached_UsesProvidedCacheName tests that
// GenerateTextCached uses the cacheName provided by the caller.
// AC-001: GenerateTextCached uses provided cacheName directly
func TestProvider_GenerateTextCached_UsesProvidedCacheName(t *testing.T) {
	t.Run("uses provided cacheName", func(t *testing.T) {
		// Given: A mock provider
		provider := &mockCachedProvider{
			response: "Cached response",
		}

		// When: GenerateTextCached is called with specific cacheName
		ctx := context.Background()
		cacheName := "my-custom-cache-123"
		response, err := provider.GenerateTextCached(ctx, cacheName, "Hello")

		// Then: Should use the exact cacheName provided
		require.NoError(t, err)
		assert.Equal(t, "Cached response", response)
		assert.Equal(t, cacheName, provider.lastUsedCacheName,
			"Should use the exact cacheName provided by caller")
	})
}

// TestProvider_DeleteCache_IsStateless tests that DeleteCache does not
// affect Provider's internal state.
// AC-001: DeleteCache deletes specified cache (no internal state update)
func TestProvider_DeleteCache_IsStateless(t *testing.T) {
	t.Run("DeleteCache is stateless", func(t *testing.T) {
		// Given: A mock provider
		provider := &mockCachedProvider{}

		// When: DeleteCache is called
		ctx := context.Background()
		cacheName := "cache-to-delete"
		err := provider.DeleteCachedConfig(ctx, cacheName)

		// Then: Should delete without maintaining internal state
		require.NoError(t, err)
		assert.Equal(t, cacheName, provider.lastDeletedCacheName)
	})
}

// =============================================================================
// Cache Lifecycle Flow Tests (Provider + Agent interaction pattern)
// =============================================================================

// TestCacheLifecycle_CreateUseDelete tests the complete cache lifecycle
// using only Provider's pure API methods.
// This demonstrates how Agent will use Provider for cache management.
func TestCacheLifecycle_CreateUseDelete(t *testing.T) {
	t.Run("complete cache lifecycle", func(t *testing.T) {
		// Given: A mock provider (simulating pure API layer)
		provider := &mockCachedProvider{
			response: "Response from cached context",
		}

		ctx := context.Background()
		systemPrompt := "You are Yuruppu, a friendly LINE bot."
		userMessage := "Hello!"

		// Step 1: Create cache (would be done by Agent during initialization)
		cacheName, err := provider.CreateCachedConfig(ctx, systemPrompt, time.Hour)
		require.NoError(t, err)
		require.NotEmpty(t, cacheName)

		// Step 2: Use cache for generation (would be done by Agent.GenerateText)
		response, err := provider.GenerateTextCached(ctx, cacheName, userMessage)
		require.NoError(t, err)
		assert.Equal(t, "Response from cached context", response)
		assert.Equal(t, cacheName, provider.lastUsedCacheName)

		// Step 3: Delete cache (would be done by Agent.Close)
		err = provider.DeleteCachedConfig(ctx, cacheName)
		require.NoError(t, err)
		assert.Equal(t, cacheName, provider.lastDeletedCacheName)

		// Provider state should NOT track any of this - it's purely transactional
	})
}

// TestNonCachedFallback tests that GenerateText (non-cached) works independently.
// This is used when cache creation fails or as a fallback.
func TestNonCachedFallback(t *testing.T) {
	t.Run("non-cached generation works", func(t *testing.T) {
		// Given: A mock provider
		provider := &mockCachedProvider{
			response: "Non-cached response",
		}

		// When: GenerateText is called (non-cached path)
		ctx := context.Background()
		response, err := provider.GenerateText(ctx, "System prompt", "User message")

		// Then: Should work without cache
		require.NoError(t, err)
		assert.Equal(t, "Non-cached response", response)
		assert.Empty(t, provider.lastUsedCacheName,
			"Should not use cache for non-cached call")
	})
}

// =============================================================================
// Close Method Tests (Updated for Pure API Layer)
// =============================================================================

// TestClose_DoesNotDeleteCache tests that Provider.Close does NOT delete cache.
// AC-001: Provider is pure API layer - cache cleanup is Agent's responsibility
func TestClose_DoesNotDeleteCache(t *testing.T) {
	t.Run("Close does not delete cache", func(t *testing.T) {
		// Given: A mock provider (pure API layer with no internal cache state)
		provider := &mockCachedProvider{}

		// When: Close is called
		err := provider.Close(context.Background())

		// Then: Close should not delete any cache (that's Agent's job)
		require.NoError(t, err)
		assert.Empty(t, provider.lastDeletedCacheName,
			"Provider.Close should NOT delete cache - that's Agent's responsibility")
	})
}

// =============================================================================
// Mock Implementation for Testing
// =============================================================================

// mockCachedProvider simulates a Provider with cache methods (pure API layer).
type mockCachedProvider struct {
	response             string
	cacheCounter         int
	lastUsedCacheName    string
	lastCreatedPrompt    string
	lastDeletedCacheName string
	closed               bool
}

func (m *mockCachedProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	// Non-cached path - doesn't use cacheName
	return m.response, nil
}

func (m *mockCachedProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	// Pure API layer: just use the provided cacheName
	m.lastUsedCacheName = cacheName
	return m.response, nil
}

func (m *mockCachedProvider) CreateCachedConfig(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	// Pure API layer: create new cache each time (no internal state)
	m.lastCreatedPrompt = systemPrompt
	m.cacheCounter++
	return "cache-" + string(rune('0'+m.cacheCounter)), nil
}

func (m *mockCachedProvider) DeleteCachedConfig(ctx context.Context, cacheName string) error {
	// Pure API layer: just delete the specified cache
	m.lastDeletedCacheName = cacheName
	return nil
}

func (m *mockCachedProvider) Close(ctx context.Context) error {
	// Pure API layer: Close does NOT delete cache
	// Cache deletion is the Agent's responsibility
	m.closed = true
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// Note: discardLogger is defined in vertexai_test.go and available to all tests in this package
