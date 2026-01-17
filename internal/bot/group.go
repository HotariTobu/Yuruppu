package bot

import (
	"context"
	"fmt"
	"log/slog"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/line"
)

// HandleJoin handles the bot being added to a group.
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

	// Fetch and save group profile asynchronously (NFR-001)
	go func() {
		summary, err := h.lineClient.GetGroupSummary(context.Background(), sourceID)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to get group summary",
				slog.String("sourceID", sourceID),
				slog.Any("error", err),
			)
			return
		}

		profile := &groupprofile.GroupProfile{
			GroupName:  summary.GroupName,
			PictureURL: summary.PictureURL,
		}

		if err := h.groupProfileService.SetGroupProfile(context.Background(), sourceID, profile); err != nil {
			h.logger.ErrorContext(ctx, "failed to save group profile",
				slog.String("sourceID", sourceID),
				slog.Any("error", err),
			)
			return
		}

		h.logger.InfoContext(ctx, "group profile saved",
			slog.String("sourceID", sourceID),
			slog.String("groupName", profile.GroupName),
		)
	}()

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
