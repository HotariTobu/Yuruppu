package client

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Client sends messages via LINE Messaging API.
type Client struct {
	api     *messaging_api.MessagingApiAPI
	blobAPI *messaging_api.MessagingApiBlobAPI
	logger  *slog.Logger
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
		return nil, fmt.Errorf("failed to create LINE messaging API client: %w", err)
	}

	// Create blob API client for media content retrieval
	blobAPI, err := messaging_api.NewMessagingApiBlobAPI(channelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create LINE messaging blob API client: %w", err)
	}

	return &Client{
		api:     api,
		blobAPI: blobAPI,
		logger:  logger,
	}, nil
}
