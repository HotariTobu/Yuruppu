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

// GroupSim defines the interface for group simulation operations.
type GroupSim interface {
	GetMembers(ctx context.Context, groupID string) ([]string, error)
}

// LineClient is a mock implementation of LINE client interfaces for CLI testing.
type LineClient struct {
	fetcher  Fetcher
	groupSim GroupSim
}

// NewLineClient creates a new mock LINE client with the given fetcher and group simulator.
func NewLineClient(fetcher Fetcher, groupSim GroupSim) *LineClient {
	if fetcher == nil {
		panic("fetcher cannot be nil")
	}
	if groupSim == nil {
		panic("groupSim cannot be nil")
	}
	return &LineClient{fetcher: fetcher, groupSim: groupSim}
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

// GetGroupMemberCount returns the number of members in a group via GroupSim.
func (c *LineClient) GetGroupMemberCount(ctx context.Context, groupID string) (int, error) {
	members, err := c.groupSim.GetMembers(ctx, groupID)
	if err != nil {
		return 0, err
	}
	return len(members), nil
}
