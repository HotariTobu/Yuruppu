package history

import "context"

// Storage defines the interface for conversation history persistence.
type Storage interface {
	// GetHistory retrieves conversation history for a source.
	// Returns empty slice if no history exists.
	GetHistory(ctx context.Context, sourceID string) ([]Message, error)

	// AppendMessages saves user message and bot response atomically.
	AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg Message) error

	// Close releases storage resources.
	Close(ctx context.Context) error
}
