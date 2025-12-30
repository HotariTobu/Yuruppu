package line

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Client sends messages via LINE Messaging API.
type Client struct {
	api    *messaging_api.MessagingApiAPI
	logger *slog.Logger
}

// NewClient creates a new LINE messaging client.
// channelToken is the LINE channel access token for API calls.
// logger is the structured logger for the client.
// Returns an error if channelToken is empty after trimming whitespace.
func NewClient(channelToken string, logger *slog.Logger) (*Client, error) {
	channelToken = strings.TrimSpace(channelToken)
	if channelToken == "" {
		return nil, errors.New("missing required configuration: channelToken")
	}

	// Create messaging API client using LINE SDK
	api, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		return nil, err
	}

	return &Client{
		api:    api,
		logger: logger,
	}, nil
}

// SendReply sends a text message reply using the LINE Messaging API.
// replyToken is the reply token from the incoming message event.
// text is the message text to send.
// Returns any error encountered during the API call.
func (c *Client) SendReply(replyToken string, text string) error {
	c.logger.Debug("sending reply",
		slog.String("replyToken", replyToken),
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

	// Call LINE ReplyMessage API
	_, err := c.api.ReplyMessage(request)
	if err != nil {
		c.logger.Error("reply failed",
			slog.String("replyToken", replyToken),
			slog.Any("error", err),
		)
		return err
	}

	c.logger.Debug("reply sent successfully",
		slog.String("replyToken", replyToken),
	)
	return nil
}
