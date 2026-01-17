package bot

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"
)

// HandleJoin handles the bot being added to a group.
// Currently logs only (FR-020).
func (h *Handler) HandleJoin(ctx context.Context) error {
	sourceID, _ := line.SourceIDFromContext(ctx)
	chatType, _ := line.ChatTypeFromContext(ctx)

	h.logger.InfoContext(ctx, "bot joined group",
		slog.String("sourceID", sourceID),
		slog.String("chatType", string(chatType)),
	)

	return nil
}

// HandleMemberJoined handles members joining a group.
// Currently logs only (FR-020).
func (h *Handler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error {
	sourceID, _ := line.SourceIDFromContext(ctx)
	chatType, _ := line.ChatTypeFromContext(ctx)

	h.logger.InfoContext(ctx, "members joined group",
		slog.String("sourceID", sourceID),
		slog.String("chatType", string(chatType)),
		slog.Any("joinedUserIDs", joinedUserIDs),
	)

	return nil
}
