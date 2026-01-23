package bot

import (
	"context"
	"log/slog"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
)

// HandleUnsend removes a message from history when the user unsends it.
// Returns nil if the message is not found (idempotent operation).
func (h *Handler) HandleUnsend(ctx context.Context, messageID string) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		h.logger.WarnContext(ctx, "unsend: sourceID not found in context")
		return nil
	}

	hist, gen, err := h.history.GetHistory(ctx, sourceID)
	if err != nil {
		h.logger.ErrorContext(ctx, "unsend: failed to get history",
			slog.String("messageID", messageID),
			slog.Any("error", err),
		)
		return err
	}

	filtered, found := filterOutMessage(hist, messageID)
	if !found {
		h.logger.WarnContext(ctx, "unsend: message not found in history",
			slog.String("sourceID", sourceID),
			slog.String("messageID", messageID),
		)
		return nil
	}

	_, err = h.history.PutHistory(ctx, sourceID, filtered, gen)
	if err != nil {
		h.logger.ErrorContext(ctx, "unsend: failed to save history",
			slog.String("messageID", messageID),
			slog.Any("error", err),
		)
		return err
	}

	h.logger.InfoContext(ctx, "unsend: message removed from history",
		slog.String("sourceID", sourceID),
		slog.String("messageID", messageID),
	)
	return nil
}

// filterOutMessage removes the message with the given ID from history.
// Returns the filtered history and whether the message was found.
func filterOutMessage(hist []history.Message, messageID string) ([]history.Message, bool) {
	filtered := make([]history.Message, 0, len(hist))
	found := false

	for _, msg := range hist {
		if userMsg, ok := msg.(*history.UserMessage); ok && userMsg.MessageID == messageID {
			found = true
			continue
		}
		filtered = append(filtered, msg)
	}

	return filtered, found
}
