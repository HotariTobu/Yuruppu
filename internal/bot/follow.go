package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"yuruppu/internal/line"
	"yuruppu/internal/userprofile"
)

func (h *Handler) HandleFollow(ctx context.Context) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}

	lineProfile, err := h.lineClient.GetUserProfile(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch profile: %w", err)
	}

	p := &userprofile.UserProfile{
		DisplayName:   lineProfile.DisplayName,
		PictureURL:    lineProfile.PictureURL,
		StatusMessage: lineProfile.StatusMessage,
	}

	if p.PictureURL != "" {
		if mimeType, err := h.fetchPictureMIMEType(ctx, p.PictureURL); err != nil {
			h.logger.WarnContext(ctx, "failed to fetch picture MIME type",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			p.PictureURL = ""
		} else {
			p.PictureMIMEType = mimeType
		}
	}

	if err := h.userProfileService.SetUserProfile(ctx, userID, p); err != nil {
		return fmt.Errorf("failed to store profile: %w", err)
	}

	return nil
}
