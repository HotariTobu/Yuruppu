package bot

import (
	"context"
)

// HandleUnsend removes a message from history when the user unsends it.
// Returns nil if the message is not found (idempotent operation).
func (h *Handler) HandleUnsend(ctx context.Context, messageID string) error {
	// TODO: Implement unsend logic (FR-002)
	// This is a stub implementation to satisfy the server.UnsendHandler interface
	return nil
}
