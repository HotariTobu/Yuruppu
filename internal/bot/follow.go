package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"yuruppu/internal/line"
	"yuruppu/internal/profile"
)

func (h *Handler) HandleFollow(ctx context.Context) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}

	lineProfile, err := h.lineClient.GetProfile(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch profile: %w", err)
	}

	p := &profile.UserProfile{
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

	if err := h.profileService.SetUserProfile(ctx, userID, p); err != nil {
		return fmt.Errorf("failed to store profile: %w", err)
	}

	return nil
}

// fetchPictureMIMEType fetches the MIME type of a picture URL via GET request.
// Uses /small suffix to minimize data transfer. Falls back to image/jpeg if
// Content-Type is not available.
func (h *Handler) fetchPictureMIMEType(ctx context.Context, url string) (string, error) {
	// Use /small suffix to minimize data transfer
	smallURL := url + "/small"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, smallURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
		h.logger.WarnContext(ctx, "Content-Type header missing, falling back to image/jpeg")
	}

	return mimeType, nil
}
