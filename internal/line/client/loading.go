package client

import (
	"context"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// ShowLoadingAnimation displays a loading animation in a 1:1 chat.
// chatID is the user ID to show the loading animation to.
// loadingSeconds is how long to display the animation (5-60 seconds).
func (c *Client) ShowLoadingAnimation(ctx context.Context, chatID string, loadingSeconds int) error {
	req := &messaging_api.ShowLoadingAnimationRequest{
		ChatId:         chatID,
		LoadingSeconds: int32(loadingSeconds),
	}
	_, err := c.api.ShowLoadingAnimation(req)
	return err
}
