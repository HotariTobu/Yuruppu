package llm

import (
	"context"
	"time"
)

// Provider is an abstraction layer for LLM providers.
// TR-002: Create an abstraction layer (interface) for LLM providers to allow future provider changes
// ADR: 20251228-provider-cache-interface - Extended with cache methods (Option C: Separate methods)
type Provider interface {
	// GenerateText generates a text response given a system prompt and user message.
	// The context can be used for timeout and cancellation.
	// This is the non-cached path - the system prompt is sent with each request.
	GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error)

	// GenerateTextCached generates a text response using a cached system prompt.
	// The cacheName must be a valid cache reference returned by CreateCachedConfig.
	// AC-001: Uses provided cacheName directly (pure API layer, no internal state).
	GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error)

	// CreateCachedConfig creates a cached content for the given system prompt.
	// Returns the cache name on success, which can be used with GenerateTextCached.
	// AC-001: Returns cacheName but does not store it internally (pure API layer).
	// The caller (Agent) is responsible for managing the cache lifecycle.
	CreateCachedConfig(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error)

	// DeleteCachedConfig deletes the specified cache.
	// AC-001: Deletes the cache but does not update internal state (pure API layer).
	// This method is idempotent - safe to call multiple times or on non-existent caches.
	DeleteCachedConfig(ctx context.Context, cacheName string) error

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
