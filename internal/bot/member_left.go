package bot

import (
	"context"
	"errors"
	"log/slog"
	"yuruppu/internal/line"
)

// HandleMemberLeft handles members leaving a group.
func (h *Handler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return errors.New("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}

	h.logger.InfoContext(ctx, "members left group",
		slog.String("chatType", string(chatType)),
		slog.String("sourceID", sourceID),
		slog.Any("leftUserIDs", leftUserIDs),
	)

	// Decrement member count (FR-003)
	profile, err := h.groupProfileService.GetGroupProfile(ctx, sourceID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to get group profile for member count update",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil
	}
	profile.UserCount -= len(leftUserIDs)
	if err := h.groupProfileService.SetGroupProfile(ctx, sourceID, profile); err != nil {
		h.logger.WarnContext(ctx, "failed to update member count",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
	}

	return nil
}
