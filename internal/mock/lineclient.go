package mock

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"yuruppu/internal/line"
)

// LineClient implements bot.LineClient interface for CLI testing.
// It prints replies to stdout instead of sending to LINE API.
type LineClient struct {
	logger *slog.Logger
}

// NewLineClient creates a new mock LineClient.
func NewLineClient(logger *slog.Logger) *LineClient {
	return &LineClient{logger: logger}
}

// GetMessageContent returns an error as media is not supported in CLI mode.
func (c *LineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	return nil, "", errors.New("media not supported in CLI mode")
}

// GetProfile returns an error as profiles are created via CLI prompts.
func (c *LineClient) GetProfile(ctx context.Context, userID string) (*line.UserProfile, error) {
	return nil, errors.New("profile should be created via CLI prompts")
}

// SendReply prints the message to stdout.
// This implements the reply.Sender interface used by the reply tool.
func (c *LineClient) SendReply(replyToken string, text string) error {
	fmt.Println(text)
	return nil
}
