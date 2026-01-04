package client

import (
	"context"
	"fmt"
	"log/slog"
)

// UserProfile contains LINE user profile information.
type UserProfile struct {
	DisplayName   string
	PictureURL    string
	StatusMessage string
}

// GetProfile fetches user profile from LINE API.
// Returns profile information including display name, picture URL, and status message.
func (c *Client) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	c.logger.DebugContext(ctx, "fetching user profile",
		slog.String("userID", userID),
	)

	resp, err := c.api.GetProfile(userID)
	if err != nil {
		return nil, fmt.Errorf("LINE API GetProfile failed: %w", err)
	}

	profile := &UserProfile{
		DisplayName:   resp.DisplayName,
		PictureURL:    resp.PictureUrl,
		StatusMessage: resp.StatusMessage,
	}

	c.logger.DebugContext(ctx, "profile fetched successfully",
		slog.String("userID", userID),
		slog.String("displayName", profile.DisplayName),
	)

	return profile, nil
}
