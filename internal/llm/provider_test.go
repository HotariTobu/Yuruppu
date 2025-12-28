package llm_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Compile-time interface conformance checks
// AC-001: Verify all mock implementations satisfy the full Provider interface
// =============================================================================

var (
	_ llm.Provider = (*mockProvider)(nil)
	_ llm.Provider = (*mockProviderWithClose)(nil)
	_ llm.Provider = (*mockProviderWithCache)(nil)
)

// =============================================================================
// Provider Interface Tests
// =============================================================================

// TestProvider_InterfaceExists tests that the Provider interface exists with expected method signature.
// TR-002: Create an abstraction layer (interface) for LLM providers
func TestProvider_InterfaceExists(t *testing.T) {
	t.Run("interface can be implemented by mock", func(t *testing.T) {
		// Given: A mock implementation of Provider interface
		mock := &mockProvider{
			response: "Hello, I am Yuruppu!",
		}

		// When: Call GenerateText method
		ctx := context.Background()
		systemPrompt := "You are Yuruppu, a friendly LINE bot."
		userMessage := "Hello!"

		got, err := mock.GenerateText(ctx, systemPrompt, userMessage)

		// Then: Should work as expected
		require.NoError(t, err)
		assert.Equal(t, "Hello, I am Yuruppu!", got,
			"mock implementation should return expected response")
	})

	t.Run("interface accepts context parameter", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProvider{response: "Response"}

		// When: Call with different contexts
		ctx1 := context.Background()
		ctx2 := context.WithValue(context.Background(), "key", "value")

		// Then: Should accept both contexts without error
		_, err1 := mock.GenerateText(ctx1, "system", "user")
		_, err2 := mock.GenerateText(ctx2, "system", "user")

		require.NoError(t, err1, "should accept background context")
		require.NoError(t, err2, "should accept context with values")
	})

	t.Run("interface returns string and error", func(t *testing.T) {
		// Given: A mock provider that returns an error
		mock := &mockProvider{
			err: errors.New("API error"),
		}

		// When: Call GenerateText
		ctx := context.Background()
		response, err := mock.GenerateText(ctx, "system", "user")

		// Then: Should return error and empty string
		require.Error(t, err)
		assert.Empty(t, response, "response should be empty on error")
		assert.Equal(t, "API error", err.Error())
	})
}

// TestProvider_MethodSignature tests the exact method signature.
// TR-002: Define a Provider interface with a method to generate text given context, system prompt, and user message
func TestProvider_MethodSignature(t *testing.T) {
	t.Run("GenerateText accepts system prompt and user message", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProvider{response: "Generated response"}

		// When: Call with specific prompts
		ctx := context.Background()
		systemPrompt := "You are Yuruppu, a friendly bot that responds in Japanese."
		userMessage := "Tell me a joke"

		response, err := mock.GenerateText(ctx, systemPrompt, userMessage)

		// Then: Should return response
		require.NoError(t, err)
		assert.NotEmpty(t, response, "should generate non-empty response")
	})

	t.Run("GenerateText handles empty system prompt", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProvider{response: "Response"}

		// When: Call with empty system prompt
		ctx := context.Background()
		response, err := mock.GenerateText(ctx, "", "user message")

		// Then: Should still work (implementation decides how to handle)
		require.NoError(t, err)
		assert.Equal(t, "Response", response)
	})

	t.Run("GenerateText handles empty user message", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProvider{response: "Response"}

		// When: Call with empty user message
		ctx := context.Background()
		response, err := mock.GenerateText(ctx, "system prompt", "")

		// Then: Should still work (implementation decides how to handle)
		require.NoError(t, err)
		assert.Equal(t, "Response", response)
	})

	t.Run("GenerateText handles long inputs", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProvider{response: "Generated"}

		// When: Call with long system prompt and user message
		ctx := context.Background()
		longSystemPrompt := "You are Yuruppu. " + string(make([]byte, 1000))
		longUserMessage := "Hello. " + string(make([]byte, 5000))

		_, err := mock.GenerateText(ctx, longSystemPrompt, longUserMessage)

		// Then: Should not panic or error on long inputs
		require.NoError(t, err)
	})
}

// TestProvider_ContextCancellation tests that the interface supports context cancellation.
// TR-002: Interface should support context for timeout and cancellation
func TestProvider_ContextCancellation(t *testing.T) {
	t.Run("provider can handle cancelled context", func(t *testing.T) {
		// Given: A mock provider that checks context
		mock := &mockProvider{
			checkContext: true,
		}

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When: Call GenerateText with cancelled context
		_, err := mock.GenerateText(ctx, "system", "user")

		// Then: Should return context cancelled error
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled,
			"should return context.Canceled error")
	})

	t.Run("provider can handle deadline exceeded context", func(t *testing.T) {
		// Given: A mock provider that checks context
		mock := &mockProvider{
			checkContext: true,
		}

		// Given: A context with deadline already exceeded
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()

		// When: Call GenerateText
		_, err := mock.GenerateText(ctx, "system", "user")

		// Then: Should return deadline exceeded error
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded,
			"should return context.DeadlineExceeded error")
	})
}

// =============================================================================
// Error Types Tests
// =============================================================================

// TestLLMTimeoutError tests the LLMTimeoutError type.
// Error Handling Table: LLMTimeoutError - LLM API call exceeds configured timeout
func TestLLMTimeoutError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create LLMTimeoutError
		err := &llm.LLMTimeoutError{
			Message: "LLM API call timed out after 30s",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error(), "should have non-empty error message")
	})

	t.Run("error message contains timeout information", func(t *testing.T) {
		// Given: Create LLMTimeoutError with specific message
		err := &llm.LLMTimeoutError{
			Message: "request timeout after 30 seconds",
		}

		// When: Get error string
		errMsg := err.Error()

		// Then: Should contain timeout information
		assert.Contains(t, errMsg, "timeout",
			"error message should indicate timeout")
		assert.Contains(t, errMsg, "30 seconds",
			"error message should contain timeout duration")
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create LLMTimeoutError wrapped in another error
		timeoutErr := &llm.LLMTimeoutError{
			Message: "timeout occurred",
		}
		wrappedErr := errors.New("LLM error: " + timeoutErr.Error())

		// When: Check with errors.As
		var target *llm.LLMTimeoutError

		// Note: This test verifies the type exists and can be used with errors.As
		// In actual implementation, wrapping should preserve the type
		assert.NotNil(t, timeoutErr)
		assert.False(t, errors.As(wrappedErr, &target),
			"wrapped string error won't match, but type can be used with errors.As")

		// Direct type assertion should work
		directErr := error(timeoutErr)
		assert.True(t, errors.As(directErr, &target),
			"direct error should match with errors.As")
	})

	t.Run("different timeout errors can have different messages", func(t *testing.T) {
		// Given: Multiple timeout errors
		err1 := &llm.LLMTimeoutError{Message: "timeout after 10s"}
		err2 := &llm.LLMTimeoutError{Message: "timeout after 30s"}

		// Then: Should have different error messages
		assert.NotEqual(t, err1.Error(), err2.Error(),
			"different timeout errors should have different messages")
	})
}

// TestLLMRateLimitError tests the LLMRateLimitError type.
// Error Handling Table: LLMRateLimitError - LLM API rate limit exceeded (HTTP 429)
func TestLLMRateLimitError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create LLMRateLimitError
		err := &llm.LLMRateLimitError{
			Message: "Rate limit exceeded",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains rate limit information", func(t *testing.T) {
		// Given: Create LLMRateLimitError
		err := &llm.LLMRateLimitError{
			Message: "HTTP 429: rate limit exceeded",
		}

		// When: Get error string
		errMsg := err.Error()

		// Then: Should contain rate limit information
		assert.Contains(t, errMsg, "rate limit",
			"error message should indicate rate limit")
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create LLMRateLimitError
		rateLimitErr := &llm.LLMRateLimitError{
			Message: "rate limit exceeded",
		}

		// When: Check with errors.As
		var target *llm.LLMRateLimitError
		directErr := error(rateLimitErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target),
			"should match LLMRateLimitError type with errors.As")
		assert.Equal(t, rateLimitErr.Message, target.Message)
	})
}

// TestLLMNetworkError tests the LLMNetworkError type.
// Error Handling Table: LLMNetworkError - Network error during API call
func TestLLMNetworkError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create LLMNetworkError
		err := &llm.LLMNetworkError{
			Message: "connection refused",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains network error details", func(t *testing.T) {
		tests := []struct {
			name           string
			message        string
			wantSubstrings []string
		}{
			{
				name:           "connection refused error",
				message:        "connection refused",
				wantSubstrings: []string{"connection refused"},
			},
			{
				name:           "DNS failure error",
				message:        "DNS lookup failed",
				wantSubstrings: []string{"DNS"},
			},
			{
				name:           "network unreachable error",
				message:        "network is unreachable",
				wantSubstrings: []string{"network", "unreachable"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given: Create LLMNetworkError
				err := &llm.LLMNetworkError{
					Message: tt.message,
				}

				// When: Get error string
				errMsg := err.Error()

				// Then: Should contain expected substrings
				for _, substr := range tt.wantSubstrings {
					assert.Contains(t, errMsg, substr,
						"error message should contain '%s'", substr)
				}
			})
		}
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create LLMNetworkError
		networkErr := &llm.LLMNetworkError{
			Message: "network error occurred",
		}

		// When: Check with errors.As
		var target *llm.LLMNetworkError
		directErr := error(networkErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target),
			"should match LLMNetworkError type with errors.As")
	})
}

// TestLLMResponseError tests the LLMResponseError type.
// Error Handling Table: LLMResponseError - Invalid or malformed response from LLM
func TestLLMResponseError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create LLMResponseError
		err := &llm.LLMResponseError{
			Message: "malformed JSON response",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains response error details", func(t *testing.T) {
		tests := []struct {
			name    string
			message string
		}{
			{
				name:    "malformed JSON",
				message: "invalid JSON response",
			},
			{
				name:    "missing required field",
				message: "response missing 'text' field",
			},
			{
				name:    "unexpected format",
				message: "unexpected response format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given: Create LLMResponseError
				err := &llm.LLMResponseError{
					Message: tt.message,
				}

				// When: Get error string
				errMsg := err.Error()

				// Then: Should contain error details
				assert.NotEmpty(t, errMsg,
					"error message should not be empty")
				assert.Contains(t, errMsg, tt.message,
					"error message should contain original message")
			})
		}
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create LLMResponseError
		responseErr := &llm.LLMResponseError{
			Message: "invalid response",
		}

		// When: Check with errors.As
		var target *llm.LLMResponseError
		directErr := error(responseErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target),
			"should match LLMResponseError type with errors.As")
	})
}

// TestLLMAuthError tests the LLMAuthError type.
// Error Handling Table: LLMAuthError - Authentication/authorization error (HTTP 401/403)
func TestLLMAuthError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create LLMAuthError
		err := &llm.LLMAuthError{
			Message: "invalid API key",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains auth error details", func(t *testing.T) {
		tests := []struct {
			name       string
			message    string
			statusCode int
		}{
			{
				name:       "HTTP 401 unauthorized",
				message:    "invalid API key",
				statusCode: 401,
			},
			{
				name:       "HTTP 403 forbidden",
				message:    "access denied",
				statusCode: 403,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given: Create LLMAuthError
				err := &llm.LLMAuthError{
					Message:    tt.message,
					StatusCode: tt.statusCode,
				}

				// When: Get error string
				errMsg := err.Error()

				// Then: Should contain auth error details
				assert.NotEmpty(t, errMsg)
				assert.Contains(t, errMsg, tt.message,
					"error message should contain auth error details")
			})
		}
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create LLMAuthError
		authErr := &llm.LLMAuthError{
			Message:    "authentication failed",
			StatusCode: 401,
		}

		// When: Check with errors.As
		var target *llm.LLMAuthError
		directErr := error(authErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target),
			"should match LLMAuthError type with errors.As")
		assert.Equal(t, 401, target.StatusCode)
	})

	t.Run("supports HTTP status code", func(t *testing.T) {
		// Given: Create LLMAuthError with status code
		err := &llm.LLMAuthError{
			Message:    "unauthorized",
			StatusCode: 401,
		}

		// Then: Should store status code
		assert.Equal(t, 401, err.StatusCode,
			"should store HTTP status code")
	})
}

// =============================================================================
// Error Type Distinction Tests
// =============================================================================

// TestErrorTypes_Distinction tests that different error types can be distinguished.
// TR-002: Define error types for different failure scenarios
func TestErrorTypes_Distinction(t *testing.T) {
	t.Run("can distinguish between different error types", func(t *testing.T) {
		// Given: Different error types
		timeoutErr := &llm.LLMTimeoutError{Message: "timeout"}
		rateLimitErr := &llm.LLMRateLimitError{Message: "rate limit"}
		networkErr := &llm.LLMNetworkError{Message: "network"}
		responseErr := &llm.LLMResponseError{Message: "response"}
		authErr := &llm.LLMAuthError{Message: "auth"}

		// When: Check each error type
		errs := []error{timeoutErr, rateLimitErr, networkErr, responseErr, authErr}

		// Then: Each should be distinguishable
		for i, err1 := range errs {
			for j, err2 := range errs {
				if i == j {
					assert.Equal(t, err1, err2,
						"same index should have equal errors")
				} else {
					assert.NotEqual(t, err1, err2,
						"different error types should not be equal")
				}
			}
		}
	})

	t.Run("can use type assertions to identify error type", func(t *testing.T) {
		// Given: An error that could be any LLM error type
		var err error = &llm.LLMTimeoutError{Message: "timeout"}

		// When: Check type with type assertion
		_, isTimeout := err.(*llm.LLMTimeoutError)
		_, isRateLimit := err.(*llm.LLMRateLimitError)
		_, isNetwork := err.(*llm.LLMNetworkError)

		// Then: Should correctly identify type
		assert.True(t, isTimeout, "should identify as LLMTimeoutError")
		assert.False(t, isRateLimit, "should not identify as LLMRateLimitError")
		assert.False(t, isNetwork, "should not identify as LLMNetworkError")
	})

	t.Run("can use errors.As to identify error type in wrapped errors", func(t *testing.T) {
		t.Run("timeout error matches LLMTimeoutError", func(t *testing.T) {
			err := &llm.LLMTimeoutError{Message: "timeout"}
			var target *llm.LLMTimeoutError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("timeout error does not match LLMRateLimitError", func(t *testing.T) {
			err := &llm.LLMTimeoutError{Message: "timeout"}
			var target *llm.LLMRateLimitError
			assert.False(t, errors.As(err, &target))
		})

		t.Run("rate limit error matches LLMRateLimitError", func(t *testing.T) {
			err := &llm.LLMRateLimitError{Message: "rate limit"}
			var target *llm.LLMRateLimitError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("network error matches LLMNetworkError", func(t *testing.T) {
			err := &llm.LLMNetworkError{Message: "network"}
			var target *llm.LLMNetworkError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("response error matches LLMResponseError", func(t *testing.T) {
			err := &llm.LLMResponseError{Message: "response"}
			var target *llm.LLMResponseError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("auth error matches LLMAuthError", func(t *testing.T) {
			err := &llm.LLMAuthError{Message: "auth"}
			var target *llm.LLMAuthError
			assert.True(t, errors.As(err, &target))
		})
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// =============================================================================
// Provider Close Method Tests (AC-004)
// =============================================================================

// TestProvider_Close tests the Close method lifecycle.
// AC-004: Provider Close Method
func TestProvider_Close(t *testing.T) {
	t.Run("Close method is callable", func(t *testing.T) {
		// Given: A Provider instance is created
		provider := &mockProviderWithClose{
			response: "test response",
		}

		// When: Close(ctx) is called
		ctx := context.Background()
		err := provider.Close(ctx)

		// Then: Close completes without error
		require.NoError(t, err, "Close should complete successfully")
	})

	t.Run("Close is idempotent - safe to call multiple times", func(t *testing.T) {
		// Given: A Provider instance is created
		provider := &mockProviderWithClose{
			response: "test response",
		}

		ctx := context.Background()

		// When: Close is called multiple times
		err1 := provider.Close(ctx)
		err2 := provider.Close(ctx)
		err3 := provider.Close(ctx)

		// Then: All calls complete without error (idempotent)
		require.NoError(t, err1, "First Close should succeed")
		require.NoError(t, err2, "Second Close should succeed (idempotent)")
		require.NoError(t, err3, "Third Close should succeed (idempotent)")
	})

	t.Run("GenerateText returns error after Close is called", func(t *testing.T) {
		// Given: A Provider instance is created
		provider := &mockProviderWithClose{
			response: "test response",
		}

		// Given: Provider is functioning normally before Close
		ctx := context.Background()
		response, err := provider.GenerateText(ctx, "system", "user")
		require.NoError(t, err, "GenerateText should work before Close")
		assert.Equal(t, "test response", response)

		// When: Close(ctx) is called
		err = provider.Close(ctx)
		require.NoError(t, err, "Close should succeed")

		// Then: Subsequent GenerateText calls return an error
		response, err = provider.GenerateText(ctx, "system", "user")
		require.Error(t, err, "GenerateText should return error after Close")
		assert.Empty(t, response, "Response should be empty after Close")
		assert.Contains(t, err.Error(), "closed",
			"Error message should indicate provider is closed")
	})

	t.Run("Multiple GenerateText calls after Close all return errors", func(t *testing.T) {
		// Given: A Provider instance that has been closed
		provider := &mockProviderWithClose{
			response: "test response",
		}

		ctx := context.Background()
		err := provider.Close(ctx)
		require.NoError(t, err)

		// When: Multiple GenerateText calls are made after Close
		_, err1 := provider.GenerateText(ctx, "system1", "user1")
		_, err2 := provider.GenerateText(ctx, "system2", "user2")
		_, err3 := provider.GenerateText(ctx, "system3", "user3")

		// Then: All calls should return errors
		assert.Error(t, err1, "First GenerateText after Close should error")
		assert.Error(t, err2, "Second GenerateText after Close should error")
		assert.Error(t, err3, "Third GenerateText after Close should error")
	})

	t.Run("Close respects context cancellation", func(t *testing.T) {
		// Given: A Provider instance
		provider := &mockProviderWithClose{
			response: "test response",
		}

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When: Close is called with cancelled context
		err := provider.Close(ctx)

		// Then: Close should handle the cancelled context gracefully
		// (Either succeed or return context.Canceled, depending on implementation)
		// For mock, we'll succeed even with cancelled context
		assert.NoError(t, err, "Close should handle cancelled context gracefully")
	})
}

// TestProvider_CloseOrder tests Close can be called before or after GenerateText.
// AC-004: Close method lifecycle verification
func TestProvider_CloseOrder(t *testing.T) {
	t.Run("Close without ever calling GenerateText", func(t *testing.T) {
		// Given: A Provider instance that has never been used
		provider := &mockProviderWithClose{
			response: "test response",
		}

		// When: Close is called without prior GenerateText calls
		ctx := context.Background()
		err := provider.Close(ctx)

		// Then: Close should succeed
		require.NoError(t, err, "Close should succeed even if GenerateText was never called")
	})

	t.Run("GenerateText, then Close, then GenerateText again fails", func(t *testing.T) {
		// Given: A Provider instance
		provider := &mockProviderWithClose{
			response: "test response",
		}

		ctx := context.Background()

		// When: GenerateText is called, then Close, then GenerateText again
		_, err1 := provider.GenerateText(ctx, "system", "user")
		require.NoError(t, err1, "First GenerateText should succeed")

		err2 := provider.Close(ctx)
		require.NoError(t, err2, "Close should succeed")

		_, err3 := provider.GenerateText(ctx, "system", "user")

		// Then: Second GenerateText should fail
		assert.Error(t, err3, "GenerateText after Close should fail")
	})
}

// =============================================================================
// Cache Methods Tests (AC-001 from 20251228-refact-llm-agent-separation)
// =============================================================================

// TestProvider_InterfaceHasCacheMethods tests that the Provider interface
// includes the new cache methods.
// AC-001: Provider interface extended with cache methods (ADR: 20251228-provider-cache-interface)
func TestProvider_InterfaceHasCacheMethods(t *testing.T) {
	t.Run("interface defines GenerateTextCached method", func(t *testing.T) {
		// Given: A mock implementation with cache methods
		var provider llm.Provider = &mockProviderWithCache{
			response: "test",
		}

		// When: Call GenerateTextCached
		ctx := context.Background()
		response, err := provider.GenerateTextCached(ctx, "cache-name", "message")

		// Then: Method should be callable through interface
		require.NoError(t, err)
		assert.Equal(t, "test", response)
	})

	t.Run("interface defines CreateCache method", func(t *testing.T) {
		// Given: A mock implementation with cache methods
		var provider llm.Provider = &mockProviderWithCache{}

		// When: Call CreateCache
		ctx := context.Background()
		cacheName, err := provider.CreateCache(ctx, "system prompt", time.Hour)

		// Then: Method should be callable through interface
		require.NoError(t, err)
		assert.NotEmpty(t, cacheName)
	})

	t.Run("interface defines DeleteCache method", func(t *testing.T) {
		// Given: A mock implementation with cache methods
		var provider llm.Provider = &mockProviderWithCache{}

		// When: Call DeleteCache
		ctx := context.Background()
		err := provider.DeleteCache(ctx, "cache-name")

		// Then: Method should be callable through interface
		require.NoError(t, err)
	})
}

// TestProvider_GenerateTextCached tests the GenerateTextCached method.
// AC-001: GenerateTextCached(ctx, cacheName, userMessage) uses provided cacheName directly
func TestProvider_GenerateTextCached(t *testing.T) {
	t.Run("uses provided cacheName directly", func(t *testing.T) {
		// Given: A mock provider with cache support
		mock := &mockProviderWithCache{
			response: "Response using cache",
		}

		// When: Call GenerateTextCached with cacheName
		ctx := context.Background()
		cacheName := "test-cache-123"
		userMessage := "Hello from user"

		response, err := mock.GenerateTextCached(ctx, cacheName, userMessage)

		// Then: Should use the provided cacheName
		require.NoError(t, err)
		assert.Equal(t, "Response using cache", response)
		assert.Equal(t, cacheName, mock.lastUsedCacheName,
			"should use the provided cacheName directly")
	})

	t.Run("accepts different cache names", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{
			response: "Response",
		}

		ctx := context.Background()

		// When: Call with different cache names
		cache1 := "cache-alpha"
		cache2 := "cache-beta"
		cache3 := "cache-gamma"

		_, _ = mock.GenerateTextCached(ctx, cache1, "message1")
		assert.Equal(t, cache1, mock.lastUsedCacheName)

		_, _ = mock.GenerateTextCached(ctx, cache2, "message2")
		assert.Equal(t, cache2, mock.lastUsedCacheName)

		_, _ = mock.GenerateTextCached(ctx, cache3, "message3")
		assert.Equal(t, cache3, mock.lastUsedCacheName)

		// Then: Should accept and use each cache name
	})

	t.Run("handles errors during cached generation", func(t *testing.T) {
		// Given: A mock provider that returns error
		mock := &mockProviderWithCache{
			err: errors.New("cache API error"),
		}

		// When: Call GenerateTextCached
		ctx := context.Background()
		response, err := mock.GenerateTextCached(ctx, "test-cache", "message")

		// Then: Should return error
		require.Error(t, err)
		assert.Empty(t, response)
		assert.Equal(t, "cache API error", err.Error())
	})

	t.Run("works with empty user message", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{
			response: "Response to empty message",
		}

		// When: Call with empty user message
		ctx := context.Background()
		response, err := mock.GenerateTextCached(ctx, "cache-name", "")

		// Then: Should not error (implementation decides behavior)
		require.NoError(t, err)
		assert.Equal(t, "Response to empty message", response)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Given: A mock provider that checks context
		mock := &mockProviderWithCache{
			checkContext: true,
		}

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When: Call GenerateTextCached with cancelled context
		_, err := mock.GenerateTextCached(ctx, "cache-name", "message")

		// Then: Should return context cancelled error
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns error after provider is closed", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{
			response: "Response",
		}

		ctx := context.Background()

		// Given: Provider is closed
		err := mock.Close(ctx)
		require.NoError(t, err)

		// When: Call GenerateTextCached after Close
		response, err := mock.GenerateTextCached(ctx, "cache-name", "message")

		// Then: Should return error
		require.Error(t, err)
		assert.Empty(t, response)
		assert.Contains(t, err.Error(), "closed")
	})
}

// TestProvider_CreateCache tests the CreateCache method.
// AC-001: CreateCache(ctx, systemPrompt) creates cache and returns cacheName (no internal storage)
func TestProvider_CreateCache(t *testing.T) {
	t.Run("creates cache and returns cacheName", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		// When: Call CreateCache with systemPrompt
		ctx := context.Background()
		systemPrompt := "You are Yuruppu, a friendly LINE bot."

		cacheName, err := mock.CreateCache(ctx, systemPrompt, time.Hour)

		// Then: Should return cacheName without error
		require.NoError(t, err)
		assert.NotEmpty(t, cacheName, "cacheName should not be empty")
		assert.Equal(t, systemPrompt, mock.lastCreatedCachePrompt,
			"should create cache with provided systemPrompt")
	})

	t.Run("does not store cacheName internally", func(t *testing.T) {
		// Given: A mock provider with no internal state
		mock := &mockProviderWithCache{}

		ctx := context.Background()
		systemPrompt := "System prompt"

		// When: CreateCache is called multiple times
		cache1, err1 := mock.CreateCache(ctx, systemPrompt, time.Hour)
		cache2, err2 := mock.CreateCache(ctx, systemPrompt, time.Hour)

		// Then: Should return different cache names (no internal state)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, cache1, cache2,
			"Provider should not reuse internal state, each call creates new cache")
	})

	t.Run("handles different system prompts", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		ctx := context.Background()

		// When: Create caches with different prompts
		prompt1 := "You are a helpful assistant."
		prompt2 := "You are Yuruppu."

		cache1, err1 := mock.CreateCache(ctx, prompt1, time.Hour)
		cache2, err2 := mock.CreateCache(ctx, prompt2, time.Hour)

		// Then: Should create different caches
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEmpty(t, cache1)
		assert.NotEmpty(t, cache2)
	})

	t.Run("handles long system prompts", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		// Given: A very long system prompt (over 32K tokens potential)
		longPrompt := "You are Yuruppu. " + string(make([]byte, 100000))

		// When: CreateCache with long prompt
		ctx := context.Background()
		cacheName, err := mock.CreateCache(ctx, longPrompt, time.Hour)

		// Then: Should not panic (implementation decides if it succeeds)
		_ = cacheName
		_ = err
	})

	t.Run("handles empty system prompt", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		// When: CreateCache with empty prompt
		ctx := context.Background()
		cacheName, err := mock.CreateCache(ctx, "", time.Hour)

		// Then: Implementation decides behavior (may succeed or error)
		_ = cacheName
		_ = err
	})

	t.Run("returns error when cache creation fails", func(t *testing.T) {
		// Given: A mock provider that fails cache creation
		mock := &mockProviderWithCache{
			createCacheErr: errors.New("insufficient tokens"),
		}

		// When: Call CreateCache
		ctx := context.Background()
		cacheName, err := mock.CreateCache(ctx, "Short prompt", time.Hour)

		// Then: Should return error
		require.Error(t, err)
		assert.Empty(t, cacheName, "cacheName should be empty on error")
		assert.Equal(t, "insufficient tokens", err.Error())
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Given: A mock provider that checks context
		mock := &mockProviderWithCache{
			checkContext: true,
		}

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When: Call CreateCache with cancelled context
		cacheName, err := mock.CreateCache(ctx, "System prompt", time.Hour)

		// Then: Should return context error
		require.Error(t, err)
		assert.Empty(t, cacheName)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("can be called after provider is closed", func(t *testing.T) {
		// Given: A closed provider
		mock := &mockProviderWithCache{}
		ctx := context.Background()
		_ = mock.Close(ctx)

		// When: CreateCache is called after Close
		cacheName, err := mock.CreateCache(ctx, "System prompt", time.Hour)

		// Then: Should return error (provider is closed)
		require.Error(t, err)
		assert.Empty(t, cacheName)
		assert.Contains(t, err.Error(), "closed")
	})
}

// TestProvider_DeleteCache tests the DeleteCache method.
// AC-001: DeleteCache(ctx, cacheName) deletes specified cache (no internal state update)
func TestProvider_DeleteCache(t *testing.T) {
	t.Run("deletes specified cache", func(t *testing.T) {
		// Given: A mock provider with a cache
		mock := &mockProviderWithCache{}

		// When: Call DeleteCache
		ctx := context.Background()
		cacheName := "test-cache-to-delete"

		err := mock.DeleteCache(ctx, cacheName)

		// Then: Should delete without error
		require.NoError(t, err)
		assert.Equal(t, cacheName, mock.lastDeletedCacheName,
			"should delete the specified cache")
	})

	t.Run("does not update internal state", func(t *testing.T) {
		// Given: A mock provider (pure API layer, no state)
		mock := &mockProviderWithCache{}

		ctx := context.Background()

		// When: Delete multiple caches
		cache1 := "cache-1"
		cache2 := "cache-2"

		err1 := mock.DeleteCache(ctx, cache1)
		err2 := mock.DeleteCache(ctx, cache2)

		// Then: Should delete each without maintaining state
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, cache2, mock.lastDeletedCacheName,
			"Provider is stateless, only tracks last call for testing")
	})

	t.Run("handles deletion errors", func(t *testing.T) {
		// Given: A mock provider that fails deletion
		mock := &mockProviderWithCache{
			deleteCacheErr: errors.New("cache not found"),
		}

		// When: Call DeleteCache
		ctx := context.Background()
		err := mock.DeleteCache(ctx, "non-existent-cache")

		// Then: Should return error
		require.Error(t, err)
		assert.Equal(t, "cache not found", err.Error())
	})

	t.Run("accepts different cache names", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		ctx := context.Background()

		tests := []struct {
			name      string
			cacheName string
		}{
			{"cache with hyphens", "cache-123-abc"},
			{"cache with underscores", "cache_456_def"},
			{"cache with slashes", "projects/test/caches/789"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When: Delete cache
				err := mock.DeleteCache(ctx, tt.cacheName)

				// Then: Should accept various cache name formats
				require.NoError(t, err)
				assert.Equal(t, tt.cacheName, mock.lastDeletedCacheName)
			})
		}
	})

	t.Run("handles empty cache name", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		// When: DeleteCache with empty name
		ctx := context.Background()
		err := mock.DeleteCache(ctx, "")

		// Then: Implementation decides behavior
		_ = err
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Given: A mock provider that checks context
		mock := &mockProviderWithCache{
			checkContext: true,
		}

		// Given: A cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When: Call DeleteCache with cancelled context
		err := mock.DeleteCache(ctx, "cache-name")

		// Then: Should return context error
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("is idempotent - safe to delete same cache multiple times", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{}

		ctx := context.Background()
		cacheName := "cache-to-delete-multiple-times"

		// When: Delete same cache multiple times
		err1 := mock.DeleteCache(ctx, cacheName)
		err2 := mock.DeleteCache(ctx, cacheName)
		err3 := mock.DeleteCache(ctx, cacheName)

		// Then: Should not error (idempotent)
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})

	t.Run("can be called after provider is closed", func(t *testing.T) {
		// Given: A closed provider
		mock := &mockProviderWithCache{}
		ctx := context.Background()
		_ = mock.Close(ctx)

		// When: DeleteCache is called after Close
		err := mock.DeleteCache(ctx, "cache-name")

		// Then: Implementation decides behavior (may succeed or error)
		_ = err
	})
}

// TestProvider_CacheMethodsIntegration tests cache methods work together.
// AC-001: Provider is Pure API Layer (integration verification)
func TestProvider_CacheMethodsIntegration(t *testing.T) {
	t.Run("create, use, and delete cache lifecycle", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{
			response: "Response from cache",
		}

		ctx := context.Background()
		systemPrompt := "You are Yuruppu."
		userMessage := "Hello"

		// When: Create cache
		cacheName, err := mock.CreateCache(ctx, systemPrompt, time.Hour)
		require.NoError(t, err)
		require.NotEmpty(t, cacheName)

		// When: Use cache for generation
		response, err := mock.GenerateTextCached(ctx, cacheName, userMessage)
		require.NoError(t, err)
		assert.Equal(t, "Response from cache", response)
		assert.Equal(t, cacheName, mock.lastUsedCacheName)

		// When: Delete cache
		err = mock.DeleteCache(ctx, cacheName)
		require.NoError(t, err)
		assert.Equal(t, cacheName, mock.lastDeletedCacheName)

		// Then: Lifecycle completes successfully
	})

	t.Run("provider does not track cache state across calls", func(t *testing.T) {
		// Given: A mock provider (pure API layer)
		mock := &mockProviderWithCache{
			response: "Response",
		}

		ctx := context.Background()

		// When: Create cache
		cache1, err := mock.CreateCache(ctx, "Prompt 1", time.Hour)
		require.NoError(t, err)

		// When: Use different cache (provider doesn't validate)
		cache2 := "different-cache-name"
		_, err = mock.GenerateTextCached(ctx, cache2, "message")
		require.NoError(t, err)

		// Then: Provider accepts any cache name (no internal state validation)
		assert.NotEqual(t, cache1, cache2,
			"Provider is pure API layer, doesn't track cache state")
	})

	t.Run("multiple caches can be created and used independently", func(t *testing.T) {
		// Given: A mock provider
		mock := &mockProviderWithCache{
			response: "Response",
		}

		ctx := context.Background()

		// When: Create multiple caches
		cache1, err1 := mock.CreateCache(ctx, "Prompt 1", time.Hour)
		cache2, err2 := mock.CreateCache(ctx, "Prompt 2", time.Hour)
		cache3, err3 := mock.CreateCache(ctx, "Prompt 3", time.Hour)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)

		// When: Use different caches
		_, err := mock.GenerateTextCached(ctx, cache1, "msg1")
		require.NoError(t, err)
		_, err = mock.GenerateTextCached(ctx, cache2, "msg2")
		require.NoError(t, err)
		_, err = mock.GenerateTextCached(ctx, cache3, "msg3")
		require.NoError(t, err)

		// Then: All caches work independently
		assert.Equal(t, cache3, mock.lastUsedCacheName)
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// mockProvider is a test implementation of the Provider interface.
// This verifies that the Provider interface can be implemented.
// AC-001: Updated to implement full Provider interface with cache methods.
type mockProvider struct {
	response     string
	err          error
	checkContext bool
	closed       bool
}

func (m *mockProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// Check context cancellation if requested
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	if m.err != nil {
		return "", m.err
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

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProvider) CreateCache(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	return "mock-cache", nil
}

func (m *mockProvider) DeleteCache(ctx context.Context, cacheName string) error {
	return nil
}

func (m *mockProvider) Close(ctx context.Context) error {
	m.closed = true
	return nil
}

// mockProviderWithClose is a test implementation with Close method for AC-004 tests.
// AC-001: Updated to implement full Provider interface with cache methods.
type mockProviderWithClose struct {
	response string
	err      error
	closed   bool
}

func (m *mockProviderWithClose) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProviderWithClose) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProviderWithClose) CreateCache(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}
	return "mock-cache", nil
}

func (m *mockProviderWithClose) DeleteCache(ctx context.Context, cacheName string) error {
	return nil
}

func (m *mockProviderWithClose) Close(ctx context.Context) error {
	// Idempotent - multiple calls to Close are safe
	m.closed = true
	return nil
}

// mockProviderWithCache is a test implementation with cache methods for AC-001 tests.
// This mock simulates the Provider interface with cache support (pure API layer).
type mockProviderWithCache struct {
	response               string
	err                    error
	createCacheErr         error
	deleteCacheErr         error
	checkContext           bool
	closed                 bool
	lastUsedCacheName      string
	lastCreatedCachePrompt string
	lastDeletedCacheName   string
	cacheCounter           int
}

func (m *mockProviderWithCache) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProviderWithCache) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	// Pure API layer: just use the provided cacheName
	m.lastUsedCacheName = cacheName

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockProviderWithCache) CreateCache(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.checkContext {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	if m.createCacheErr != nil {
		return "", m.createCacheErr
	}

	// Pure API layer: create new cache each time (no internal state)
	m.lastCreatedCachePrompt = systemPrompt
	m.cacheCounter++
	cacheName := fmt.Sprintf("cache-%d", m.cacheCounter)
	return cacheName, nil
}

func (m *mockProviderWithCache) DeleteCache(ctx context.Context, cacheName string) error {
	// Note: DeleteCache can be called even after Close in some implementations
	// This is left to implementation to decide

	if m.checkContext {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if m.deleteCacheErr != nil {
		return m.deleteCacheErr
	}

	// Pure API layer: just delete the specified cache (no state update)
	m.lastDeletedCacheName = cacheName
	return nil
}

func (m *mockProviderWithCache) Close(ctx context.Context) error {
	m.closed = true
	return nil
}
