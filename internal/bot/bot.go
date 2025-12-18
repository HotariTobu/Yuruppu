package bot

import (
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// ConfigError represents an error related to missing or invalid configuration.
type ConfigError struct {
	Variable string
}

func (e *ConfigError) Error() string {
	return "Missing required environment variable: " + e.Variable
}

// Bot represents a LINE bot client.
type Bot struct {
	channelSecret string
	client        *messaging_api.MessagingApiAPI
}

// NewBot creates a new LINE bot client with the given credentials.
// channelSecret is the LINE channel secret for signature verification.
// channelAccessToken is the LINE channel access token for API calls.
// Returns the bot client or an error if initialization fails.
func NewBot(channelSecret, channelAccessToken string) (*Bot, error) {
	// Trim whitespace from credentials
	channelSecret = strings.TrimSpace(channelSecret)
	channelAccessToken = strings.TrimSpace(channelAccessToken)

	// Validate channelSecret
	if channelSecret == "" {
		return nil, &ConfigError{Variable: "LINE_CHANNEL_SECRET"}
	}

	// Validate channelAccessToken
	if channelAccessToken == "" {
		return nil, &ConfigError{Variable: "LINE_CHANNEL_ACCESS_TOKEN"}
	}

	// Create messaging API client
	client, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	if err != nil {
		return nil, err
	}

	// Create and return Bot instance
	return &Bot{
		channelSecret: channelSecret,
		client:        client,
	}, nil
}
