package line

import (
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// MessagingAPI is the interface for LINE messaging API operations.
// This allows mocking in tests while using the real client in production.
type MessagingAPI interface {
	ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
}

// Client sends messages via LINE Messaging API.
type Client struct {
	api MessagingAPI
}

// NewClient creates a new LINE messaging client.
// channelToken is the LINE channel access token for API calls.
// Returns an error if channelToken is empty after trimming whitespace.
func NewClient(channelToken string) (*Client, error) {
	channelToken = strings.TrimSpace(channelToken)
	if channelToken == "" {
		return nil, &ConfigError{Variable: "channelToken"}
	}

	// Create messaging API client using LINE SDK
	api, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		return nil, err
	}

	return &Client{
		api: api,
	}, nil
}

// NewClientWithAPI creates a new LINE messaging client with a custom API implementation.
// This is used for testing with mock API implementations.
func NewClientWithAPI(api MessagingAPI) *Client {
	return &Client{
		api: api,
	}
}

// SendReply sends a text message reply using the LINE Messaging API.
// replyToken is the reply token from the incoming message event.
// text is the message text to send.
// Returns any error encountered during the API call.
func (c *Client) SendReply(replyToken string, text string) error {
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
	return err
}
