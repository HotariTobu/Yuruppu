package client

import (
	"fmt"
	"log/slog"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// SendFlexReply sends a flex message reply using the LINE Messaging API.
// replyToken is the reply token from the incoming message event.
// altText is the fallback text for devices that don't support flex messages.
// flexContainer is the parsed flex container.
// Returns any error encountered during the API call.
func (c *Client) SendFlexReply(replyToken string, altText string, flexContainer messaging_api.FlexContainerInterface) error {
	c.logger.Debug("sending flex reply",
		slog.Int("altTextLength", len(altText)),
	)

	// Create flex message
	flexMessage := messaging_api.FlexMessage{
		AltText:  altText,
		Contents: flexContainer,
	}

	// Create reply message request
	request := &messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			flexMessage,
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
		return fmt.Errorf("LINE API flex reply failed (x-line-request-id=%s): %w", requestID, err)
	}

	c.logger.Debug("flex reply sent successfully",
		slog.String("x-line-request-id", requestID),
	)
	return nil
}

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
