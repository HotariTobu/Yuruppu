package mock

import (
	"context"
	"errors"
	"time"

	lineclient "yuruppu/internal/line/client"
)

// Fetcher defines the interface for fetching user profiles and group summaries.
type Fetcher interface {
	FetchUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
	FetchGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
}

// LineClient is a mock implementation of LINE client interfaces for CLI testing.
type LineClient struct {
	fetcher Fetcher
}

// NewLineClient creates a new mock LINE client with the given fetcher.
func NewLineClient(fetcher Fetcher) *LineClient {
	if fetcher == nil {
		panic("fetcher cannot be nil")
	}
	return &LineClient{fetcher: fetcher}
}

// GetMessageContent returns an error indicating that media operations are not supported in mock mode.
func (c *LineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	return nil, "", errors.New("media operations are not supported in CLI mode")
}

// GetUserProfile delegates to the Fetcher.
func (c *LineClient) GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	return c.fetcher.FetchUserProfile(ctx, userID)
}

// GetGroupSummary delegates to the Fetcher.
func (c *LineClient) GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error) {
	return c.fetcher.FetchGroupSummary(ctx, groupID)
}

// SendReply is a no-op in CLI mode since bot output is already logged.
func (c *LineClient) SendReply(replyToken string, text string) error {
	return nil
}

// ShowLoadingAnimation is a no-op in CLI mode since bot output is already logged.
func (c *LineClient) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
	return nil
}

// GetGroupMemberCount returns 0 in CLI mode since member count is not available.
func (c *LineClient) GetGroupMemberCount(ctx context.Context, groupID string) (int, error) {
	return 0, nil
}
