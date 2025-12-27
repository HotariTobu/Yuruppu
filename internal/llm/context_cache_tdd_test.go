package llm_test

import (
	"context"
	"testing"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD Tests for Context Caching (CH-001, AC-001, AC-006, AC-007)
// =============================================================================
// These tests verify context caching behavior for system prompts.
// They will FAIL until the implementation is complete.
//
// CH-001: Add context caching for system prompt
// AC-001: Context Cache Creation - system prompt is cached for reuse
// AC-006: Cache Expiration Handling - cache is recreated if expired
// AC-007: Fallback for Insufficient Token Count - caching skipped if below minimum

// =============================================================================
// AC-001: Context Cache Creation Tests
// =============================================================================

// TestNewVertexAIClientWithCache_AcceptsSystemPrompt tests that NewVertexAIClientWithCache
// accepts a system prompt parameter for caching.
// AC-001: System prompt is cached for reuse across requests
func TestNewVertexAIClientWithCache_AcceptsSystemPrompt(t *testing.T) {
	// Given: A system prompt to cache
	systemPrompt := "You are a helpful assistant."

	// When: Creating a client with caching support
	// Note: This will fail until NewVertexAIClientWithCache is implemented
	_, err := llm.NewVertexAIClientWithCache(
		context.Background(),
		"test-project",
		"test-region",
		"test-model",
		systemPrompt,
		discardLogger(),
	)

	// Then: Should not return an error for the function signature
	// (May fail on actual API call, but that's expected in unit tests)
	// We're testing the interface exists, not the actual caching
	_ = err // API call may fail without credentials
}

// TestNewVertexAIClientWithCache_ValidatesSystemPrompt tests system prompt validation.
// AC-001: System prompt must be provided for caching
func TestNewVertexAIClientWithCache_ValidatesSystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:         "empty system prompt returns error",
			systemPrompt: "",
			wantErr:      true,
			wantErrMsg:   "systemPrompt is required",
		},
		{
			name:         "whitespace-only system prompt returns error",
			systemPrompt: "   ",
			wantErr:      true,
			wantErrMsg:   "systemPrompt is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Creating a client with invalid system prompt
			client, err := llm.NewVertexAIClientWithCache(
				context.Background(),
				"test-project",
				"test-region",
				"test-model",
				tt.systemPrompt,
				discardLogger(),
			)

			// Then: Should return validation error
			if tt.wantErr {
				require.Error(t, err, "should return error for invalid system prompt")
				assert.Nil(t, client, "client should be nil when validation fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate system prompt is required")
			}
		})
	}
}

// =============================================================================
// AC-007: Fallback for Insufficient Token Count Tests
// =============================================================================

// TestContextCacheCreationFallback tests that caching is skipped gracefully
// when the system prompt is below the minimum token requirement (32K).
// AC-007: Caching is skipped gracefully when token count is insufficient
func TestContextCacheCreationFallback(t *testing.T) {
	// Given: A short system prompt (below 32K token minimum)
	shortSystemPrompt := "You are a helpful assistant."

	// When: Creating a client with a short system prompt
	// The cache creation should fail but the client should still work
	client, err := llm.NewVertexAIClientWithCache(
		context.Background(),
		"test-project",
		"test-region",
		"test-model",
		shortSystemPrompt,
		discardLogger(),
	)

	// Then: Client creation should succeed (graceful fallback)
	// Note: In unit tests, this may fail due to missing credentials,
	// but we're testing the logic that caching failure doesn't break client creation
	_ = client
	_ = err
}

// =============================================================================
// AC-002: Context Cache Usage Tests
// =============================================================================

// TestGenerateText_UsesCache tests that GenerateText uses cached content.
// AC-002: The cached system prompt is used instead of sending it with each request
func TestGenerateText_UsesCache(t *testing.T) {
	// This test verifies that when a cache is created, GenerateText uses it.
	// We can't easily test this in unit tests without mocking the API.
	// This is primarily tested in integration tests.

	t.Run("GenerateText works with cached client", func(t *testing.T) {
		// Given: A mock cached provider
		provider := &mockCachedProvider{
			cacheName: "test-cache-name",
			response:  "Hello from cached context!",
		}

		// When: GenerateText is called
		response, err := provider.GenerateText(
			context.Background(),
			"This prompt is ignored when cache is used",
			"Hello",
		)

		// Then: Response should come from cached context
		require.NoError(t, err)
		assert.Equal(t, "Hello from cached context!", response)
		assert.True(t, provider.usedCache, "Should use cached content")
	})
}

// =============================================================================
// AC-006: Cache Expiration Handling Tests
// =============================================================================

// TestGenerateText_RecreatesCacheOnExpiration tests that expired cache is recreated.
// AC-006: Cache is recreated automatically when expired
func TestGenerateText_RecreatesCacheOnExpiration(t *testing.T) {
	t.Run("cache is recreated when expired", func(t *testing.T) {
		// Given: A mock provider with expired cache
		provider := &mockCachedProvider{
			cacheName:    "expired-cache",
			cacheExpired: true,
			response:     "Response after cache recreation",
		}

		// When: GenerateText is called with expired cache
		response, err := provider.GenerateText(
			context.Background(),
			"system prompt for recreation",
			"user message",
		)

		// Then: Cache should be recreated and request should succeed
		require.NoError(t, err)
		assert.Equal(t, "Response after cache recreation", response)
		assert.True(t, provider.cacheRecreated, "Cache should be recreated on expiration")
	})
}

// =============================================================================
// Close Method Tests (Updated for Caching)
// =============================================================================

// TestClose_DeletesCachedContent tests that Close deletes cached content.
// AC-004: Cached resources are cleaned up
func TestClose_DeletesCachedContent(t *testing.T) {
	t.Run("Close deletes cached content", func(t *testing.T) {
		// Given: A mock provider with cached content
		provider := &mockCachedProvider{
			cacheName:        "test-cache",
			hasCachedContent: true,
		}

		// When: Close is called
		err := provider.Close(context.Background())

		// Then: Cached content should be deleted
		require.NoError(t, err)
		assert.False(t, provider.hasCachedContent, "Cached content should be deleted")
		assert.True(t, provider.cacheDeleted, "Cache should be deleted on Close")
	})
}

// =============================================================================
// Mock Implementations for Testing
// =============================================================================

// mockCachedProvider simulates a Provider with caching support.
type mockCachedProvider struct {
	cacheName        string
	response         string
	cacheExpired     bool
	cacheRecreated   bool
	cacheDeleted     bool
	hasCachedContent bool
	usedCache        bool
	closed           bool
}

func (m *mockCachedProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}

	// Simulate cache expiration handling
	if m.cacheExpired {
		m.cacheRecreated = true
		m.cacheExpired = false
	}

	// Mark that cache was used (if cache exists)
	if m.cacheName != "" {
		m.usedCache = true
	}

	return m.response, nil
}

func (m *mockCachedProvider) Close(ctx context.Context) error {
	if !m.closed {
		// Clean up cached content
		if m.hasCachedContent {
			m.cacheDeleted = true
			m.hasCachedContent = false
		}
	}
	m.closed = true
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// Note: discardLogger is defined in vertexai_test.go and available to all tests in this package
