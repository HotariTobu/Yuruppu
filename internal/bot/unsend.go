package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
)

// HandleUnsend removes a message from history when the user unsends it.
// Returns nil if the message is not found (idempotent operation).
func (h *Handler) HandleUnsend(ctx context.Context, messageID string) error {
	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return err
	}

	// Extract sourceID from context
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}

	// Load history from storage
	messages, generation, err := h.history.GetHistory(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Filter out messages with matching MessageID
	filteredMessages := make([]history.Message, 0, len(messages))
	removed := false
	for _, msg := range messages {
		if userMsg, ok := msg.(*history.UserMessage); ok && userMsg.MessageID == messageID {
			removed = true
			continue // Skip this message (remove it)
		}
		filteredMessages = append(filteredMessages, msg)
	}

	// If no message was removed, log a warning but return nil (idempotent)
	if !removed {
		h.logger.WarnContext(ctx, "unsend requested for message not found in history",
			slog.String("messageID", messageID),
			slog.String("sourceID", sourceID),
		)
		return nil
	}

	// Save updated history to storage
	_, err = h.history.PutHistory(ctx, sourceID, filteredMessages, generation)
	if err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}
