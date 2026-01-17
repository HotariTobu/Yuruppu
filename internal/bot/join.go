package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/line"
)

// HandleJoin handles the bot being added to a group.
func (h *Handler) HandleJoin(ctx context.Context) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return errors.New("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}

	h.logger.InfoContext(ctx, "bot joined group",
		slog.String("chatType", string(chatType)),
		slog.String("sourceID", sourceID),
	)

	summary, err := h.lineClient.GetGroupSummary(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get group summary: %w", err)
	}

	profile := &groupprofile.GroupProfile{
		DisplayName: summary.GroupName,
		PictureURL:  summary.PictureURL,
	}

	if profile.PictureURL != "" {
		if mimeType, err := h.fetchPictureMIMEType(ctx, profile.PictureURL); err != nil {
			h.logger.WarnContext(ctx, "failed to fetch group picture MIME type",
				slog.String("sourceID", sourceID),
				slog.Any("error", err),
			)
			profile.PictureURL = ""
		} else {
			profile.PictureMIMEType = mimeType
		}
	}

	if err := h.groupProfileService.SetGroupProfile(ctx, sourceID, profile); err != nil {
		return fmt.Errorf("failed to save group profile: %w", err)
	}

	return nil
}

// HandleMemberJoined handles members joining a group.
// Currently logs only (FR-020).
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

	return nil
}
