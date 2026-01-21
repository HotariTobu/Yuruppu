package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// messagingAPI defines the LINE Messaging API operations used by Client.
type messagingAPI interface {
	ReplyMessageWithHttpInfo(request *messaging_api.ReplyMessageRequest) (*http.Response, *messaging_api.ReplyMessageResponse, error)
	ShowLoadingAnimation(request *messaging_api.ShowLoadingAnimationRequest) (*map[string]any, error)
	GetProfile(userId string) (*messaging_api.UserProfileResponse, error)
	GetGroupSummary(groupId string) (*messaging_api.GroupSummaryResponse, error)
	GetGroupMemberCount(groupId string) (*messaging_api.GroupMemberCountResponse, error)
}

// Client sends messages via LINE Messaging API.
type Client struct {
	api     messagingAPI
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

// NewClientForTest creates a Client with injected dependencies for testing.
func NewClientForTest(api messagingAPI, blobAPI *messaging_api.MessagingApiBlobAPI, logger *slog.Logger) *Client {
	return &Client{api: api, blobAPI: blobAPI, logger: logger}
}

// ShowLoadingAnimation displays a loading animation in a 1:1 chat.
// timeout is converted to seconds (5-60) for LINE API.
func (c *Client) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
	loadingSeconds := int32(timeout.Seconds())
	req := &messaging_api.ShowLoadingAnimationRequest{
		ChatId:         chatID,
		LoadingSeconds: loadingSeconds,
	}
	_, err := c.api.ShowLoadingAnimation(req)
	return err
}
