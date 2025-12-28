package llm_test

import (
	"context"
	"testing"
	"time"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD Tests for Provider.Close Method (AC-004)
// =============================================================================
// These tests verify the Close method exists in Provider interface and vertexAIClient.
// They will FAIL until the implementation is complete.
//
// AC-004 Requirements:
// - Close(ctx) is callable
// - Close() is idempotent (safe to call multiple times)
// - After Close(), subsequent GenerateText() calls return an error

// TestProviderInterfaceHasClose verifies Close method exists in Provider interface.
// This test will fail to compile until Close is added to the Provider interface.
func TestProviderInterfaceHasClose(t *testing.T) {
	// Given: A Provider interface variable
	var provider llm.Provider = &testProviderImpl{}

	// When: Close is called through the interface
	ctx := context.Background()
	err := provider.Close(ctx) // WILL FAIL: Provider has no method Close

	// Then: Should succeed
	require.NoError(t, err)
}

// =============================================================================
// Test Implementation
// =============================================================================

// testProviderImpl is a minimal implementation for testing.
type testProviderImpl struct {
	closed bool
}

func (t *testProviderImpl) GenerateText(ctx context.Context, systemPrompt, userMessage string, history []llm.Message) (string, error) {
	if t.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	return "test response", nil
}

func (t *testProviderImpl) GenerateTextCached(ctx context.Context, cacheName, userMessage string, history []llm.Message) (string, error) {
	if t.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	return "test response from cache", nil
}

func (t *testProviderImpl) CreateCachedConfig(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if t.closed {
		return "", &llm.LLMClosedError{Message: "provider is closed"}
	}
	return "test-cache-name", nil
}

func (t *testProviderImpl) DeleteCachedConfig(ctx context.Context, cacheName string) error {
	return nil
}

func (t *testProviderImpl) Close(ctx context.Context) error {
	t.closed = true
	return nil
}
