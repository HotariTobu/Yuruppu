package mock

import (
	"context"
	"errors"
	"time"

	lineclient "yuruppu/internal/line/client"
)

// ProfileFetcher defines the interface for fetching user profiles.
// In CLI mode, this is implemented by a prompter that asks for user input.
type ProfileFetcher interface {
	FetchProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
}

// LineClient is a mock implementation of LINE client interfaces for CLI testing.
// It implements both bot.LineClient and reply.LineClient interfaces.
type LineClient struct {
	profileFetcher ProfileFetcher
}

// NewLineClient creates a new mock LINE client with the given profile fetcher.
// Panics if fetcher is nil.
func NewLineClient(fetcher ProfileFetcher) *LineClient {
	if fetcher == nil {
		panic("fetcher cannot be nil")
	}
	return &LineClient{profileFetcher: fetcher}
}

// GetMessageContent returns an error indicating that media operations are not supported in mock mode.
// This method implements the bot.LineClient interface.
func (c *LineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	return nil, "", errors.New("media operations are not supported in CLI mode")
}

// GetProfile delegates to the ProfileFetcher.
// This method implements the bot.LineClient interface.
func (c *LineClient) GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	return c.profileFetcher.FetchProfile(ctx, userID)
}

// SendReply is a no-op in CLI mode since bot output is already logged.
// This method implements the reply.LineClient interface.
func (c *LineClient) SendReply(replyToken string, text string) error {
	return nil
}

// ShowLoadingAnimation is a no-op in CLI mode since bot output is already logged.
// This method implements the bot.LineClient interface.
func (c *LineClient) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
	return nil
}
