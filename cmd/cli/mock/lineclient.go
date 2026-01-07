package mock

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	writer         io.Writer
	profileFetcher ProfileFetcher
}

// NewLineClient creates a new mock LINE client that writes output to the given writer.
// Panics if the writer is nil.
func NewLineClient(w io.Writer) *LineClient {
	if w == nil {
		panic("writer cannot be nil")
	}
	return &LineClient{
		writer: w,
	}
}

// RegisterProfileFetcher registers a profile fetcher for GetProfile calls.
func (c *LineClient) RegisterProfileFetcher(fetcher ProfileFetcher) {
	c.profileFetcher = fetcher
}

// GetMessageContent returns an error indicating that media operations are not supported in mock mode.
// This method implements the bot.LineClient interface.
func (c *LineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	return nil, "", errors.New("media operations are not supported in CLI mode")
}

// GetProfile delegates to the registered ProfileFetcher.
// This method implements the bot.LineClient interface.
func (c *LineClient) GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	if c.profileFetcher == nil {
		return nil, errors.New("profile fetcher not registered")
	}
	return c.profileFetcher.FetchProfile(ctx, userID)
}

// SendReply writes the message to the configured output writer.
// This method implements the reply.LineClient interface.
func (c *LineClient) SendReply(replyToken string, text string) error {
	_, err := fmt.Fprintf(c.writer, "%s\n", text)
	return err
}
