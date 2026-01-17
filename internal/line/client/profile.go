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

// GroupSummary contains LINE group summary information.
type GroupSummary struct {
	GroupID    string
	GroupName  string
	PictureURL string
}

// GetUserProfile fetches user profile from LINE API.
func (c *Client) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
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

	c.logger.DebugContext(ctx, "user profile fetched successfully",
		slog.String("userID", userID),
		slog.String("displayName", profile.DisplayName),
	)

	return profile, nil
}

// GetGroupSummary fetches group summary from LINE API.
func (c *Client) GetGroupSummary(ctx context.Context, groupID string) (*GroupSummary, error) {
	c.logger.DebugContext(ctx, "fetching group summary",
		slog.String("groupID", groupID),
	)

	resp, err := c.api.GetGroupSummary(groupID)
	if err != nil {
		return nil, fmt.Errorf("LINE API GetGroupSummary failed: %w", err)
	}

	summary := &GroupSummary{
		GroupID:    resp.GroupId,
		GroupName:  resp.GroupName,
		PictureURL: resp.PictureUrl,
	}

	c.logger.DebugContext(ctx, "group summary fetched successfully",
		slog.String("groupID", groupID),
		slog.String("groupName", summary.GroupName),
	)

	return summary, nil
}
