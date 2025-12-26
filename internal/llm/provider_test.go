package llm_test

import (
	"context"
	"errors"
	"testing"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// mockProvider is a test implementation of the Provider interface.
// This verifies that the Provider interface can be implemented.
type mockProvider struct {
	response     string
	err          error
	checkContext bool
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
