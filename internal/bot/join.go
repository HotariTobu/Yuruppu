package bot

import (
	"context"
	"fmt"
	"log/slog"

	"yuruppu/internal/line"
)

// HandleJoin handles the bot being added to a group.
func (h *Handler) HandleJoin(ctx context.Context) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}

	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return fmt.Errorf("chatType not found in context")
	}

	if chatType != line.ChatTypeGroup {
		return fmt.Errorf("HandleJoin called for non-group chat")
	}

	h.logger.InfoContext(ctx, "Bot joined group",
		slog.String("groupID", sourceID),
	)

	return nil
}

// HandleMemberJoined handles members joining a group.
func (h *Handler) HandleMemberJoined(ctx context.Context, memberUserIDs []string) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}

	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return fmt.Errorf("chatType not found in context")
	}

	if chatType != line.ChatTypeGroup {
		return fmt.Errorf("HandleMemberJoined called for non-group chat")
	}

	if len(memberUserIDs) == 0 {
		return fmt.Errorf("memberUserIDs must not be empty")
	}

	h.logger.InfoContext(ctx, "Members joined group",
		slog.String("groupID", sourceID),
		slog.Any("memberUserIDs", memberUserIDs),
	)

	return nil
}
