package client

import (
	"fmt"
	"log/slog"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// SendReply sends a text message reply using the LINE Messaging API.
// replyToken is the reply token from the incoming message event.
// text is the message text to send.
// Returns any error encountered during the API call.
func (c *Client) SendReply(replyToken string, text string) error {
	c.logger.Debug("sending reply",
		slog.Int("textLength", len(text)),
	)

	// Create text message
	textMessage := messaging_api.TextMessage{
		Text: text,
	}

	// Create reply message request
	request := &messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			textMessage,
		},
	}

	// Call LINE ReplyMessage API with HTTP info for x-line-request-id
	httpResp, _, err := c.api.ReplyMessageWithHttpInfo(request)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}

	// Extract x-line-request-id for debugging (available even on error)
	var requestID string
	if httpResp != nil {
		requestID = httpResp.Header.Get("X-Line-Request-Id")
	}

	if err != nil {
		return fmt.Errorf("LINE API reply failed (x-line-request-id=%s): %w", requestID, err)
	}

	c.logger.Debug("reply sent successfully",
		slog.String("x-line-request-id", requestID),
	)
	return nil
}
