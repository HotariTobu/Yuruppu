// Package agent provides the Agent interface for LLM interactions.
package agent

import (
	"context"
	"time"
)

// Message represents a single message in conversation history.
type Message struct {
	Role    string // "user" or "model"
	Content string
}

// Agent defines the interface for LLM agents.
// Implementations may have internal caching or other optimizations.
type Agent interface {
	// Configure sets up the system prompt and creates cache.
	// Must be called before GenerateText.
	// Returns error if configuration fails.
	Configure(ctx context.Context, systemPrompt string) error

	// GenerateText generates a text response given a user message.
	// history provides optional conversation context (may be nil).
	// Returns NotConfiguredError if Configure has not been called.
	// Returns ClosedError if the Agent has been closed.
	GenerateText(ctx context.Context, userMessage string, history []Message) (string, error)

	// Close releases any resources held by the agent.
	// Close is idempotent (safe to call multiple times).
	// After Close, subsequent GenerateText calls return ClosedError.
	Close(ctx context.Context) error
}

// Config holds configuration for creating an Agent.
type Config struct {
	ProjectID string
	Region    string
	Model     string
	CacheTTL  time.Duration
}
