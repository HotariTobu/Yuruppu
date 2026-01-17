package bot

import (
	"context"
	"fmt"
	"log/slog"
	"yuruppu/internal/line"
)

// HandleJoin handles the bot being added to a group.
// Currently logs only (FR-020).
func (h *Handler) HandleJoin(ctx context.Context) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return fmt.Errorf("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}

	h.logger.InfoContext(ctx, "bot joined group",
		slog.String("chatType", string(chatType)),
		slog.String("sourceID", sourceID),
	)

	return nil
}

// HandleMemberJoined handles members joining a group.
// Currently logs only (FR-020).
func (h *Handler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return fmt.Errorf("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}

	h.logger.InfoContext(ctx, "members joined group",
		slog.String("chatType", string(chatType)),
		slog.String("sourceID", sourceID),
		slog.Any("joinedUserIDs", joinedUserIDs),
	)

	return nil
}
