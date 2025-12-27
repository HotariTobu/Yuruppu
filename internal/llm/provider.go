package llm

import "context"

// Provider is an abstraction layer for LLM providers.
// TR-002: Create an abstraction layer (interface) for LLM providers to allow future provider changes
type Provider interface {
	// GenerateText generates a text response given a system prompt and user message.
	// The context can be used for timeout and cancellation.
	GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error)

	// Close releases any resources held by the provider.
	// AC-004: Provider lifecycle management
	// - Close is idempotent (safe to call multiple times)
	// - After Close, subsequent GenerateText calls return an error
	Close(ctx context.Context) error
}

// LLMTimeoutError represents an LLM API timeout error.
// Error Handling: LLM API call exceeds configured timeout
type LLMTimeoutError struct {
	Message string
}

func (e *LLMTimeoutError) Error() string {
	return e.Message
}

// LLMRateLimitError represents an LLM API rate limit error.
// Error Handling: LLM API rate limit exceeded (HTTP 429)
type LLMRateLimitError struct {
	Message string
}

func (e *LLMRateLimitError) Error() string {
	return e.Message
}

// LLMNetworkError represents a network error during LLM API call.
// Error Handling: Network error during API call (connection refused, DNS failure, etc.)
type LLMNetworkError struct {
	Message string
}

func (e *LLMNetworkError) Error() string {
	return e.Message
}

// LLMResponseError represents an invalid or malformed response from LLM.
// Error Handling: Invalid or malformed response from LLM
type LLMResponseError struct {
	Message string
}

func (e *LLMResponseError) Error() string {
	return e.Message
}

// LLMAuthError represents an authentication/authorization error.
// Error Handling: Authentication/authorization error (HTTP 401/403, invalid API key)
type LLMAuthError struct {
	Message    string
	StatusCode int // HTTP status code (401 or 403)
}

func (e *LLMAuthError) Error() string {
	return e.Message
}

// LLMClosedError represents an error when using a closed provider.
// AC-004: Provider Close Method - GenerateText returns error after Close
type LLMClosedError struct {
	Message string
}

func (e *LLMClosedError) Error() string {
	return e.Message
}
