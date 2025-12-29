// Package agent provides the Agent interface for LLM interactions.
package agent

import (
	"context"
	"yuruppu/internal/message"
)

// Agent defines the interface for LLM agents.
// Implementations may have internal caching or other optimizations.
type Agent interface {
	// GenerateText generates a text response for the conversation history.
	// The last message in history must be the user message to respond to.
	// Returns ClosedError if the Agent has been closed.
	GenerateText(ctx context.Context, history []message.Message) (string, error)

	// Close releases any resources held by the agent.
	// Close is idempotent (safe to call multiple times).
	// After Close, subsequent GenerateText calls return ClosedError.
	Close(ctx context.Context) error
}
