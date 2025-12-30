// Package agent provides the Agent interface for LLM interactions.
package agent

import (
	"context"
)

// Agent defines the interface for LLM agents.
// Implementations may have internal caching or other optimizations.
type Agent interface {
	// Generate generates a text response for the conversation history.
	// The last message in history must be the user message to respond to.
	// Returns an error if the Agent has been closed.
	Generate(ctx context.Context, history []Message, userMessage *UserMessage) (*AssistantMessage, error)

	// Close releases any resources held by the agent.
	// Close is idempotent (safe to call multiple times).
	// After Close, subsequent Generate calls return an error.
	Close(ctx context.Context) error
}
