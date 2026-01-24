package bot

import (
	"context"
	"errors"
	"log/slog"
	"yuruppu/internal/line"
)

// HandleMemberJoined handles members joining a group.
func (h *Handler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return errors.New("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}

	h.logger.InfoContext(ctx, "members joined group",
		slog.String("chatType", string(chatType)),
		slog.String("sourceID", sourceID),
		slog.Any("joinedUserIDs", joinedUserIDs),
	)

	// Increment member count (FR-002)
	profile, err := h.groupProfileService.GetGroupProfile(ctx, sourceID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to get group profile for member count update",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil
	}
	profile.UserCount += len(joinedUserIDs)
	if err := h.groupProfileService.SetGroupProfile(ctx, sourceID, profile); err != nil {
		h.logger.WarnContext(ctx, "failed to update member count",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
	}

	return nil
}
