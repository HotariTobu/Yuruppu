//go:build integration

package client_test

import (
	"context"
	"log/slog"
	"testing"
	"yuruppu/internal/line/client"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserProfile_Integration tests that GetUserProfile returns user profile from LINE API.
// FR-001: Fetch user profile from LINE when processing a message.
func TestGetUserProfile_Integration(t *testing.T) {
	_, channelAccessToken := requireLINECredentials(t)

	c, err := client.NewClient(channelAccessToken, slog.New(slog.DiscardHandler))
	require.NoError(t, err, "NewClient should succeed")

	// Get bot info first to get a valid user ID (the bot itself)
	api, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	require.NoError(t, err)
	botInfo, err := api.GetBotInfo()
	require.NoError(t, err)

	// Use the bot's own user ID for testing
	profile, err := c.GetUserProfile(context.Background(), botInfo.UserId)

	require.NoError(t, err, "GetUserProfile should succeed for bot's own user ID")
	assert.NotNil(t, profile, "profile should not be nil")
	assert.NotEmpty(t, profile.DisplayName, "display name should not be empty")
	assert.Equal(t, botInfo.DisplayName, profile.DisplayName, "display name should match bot info")
}
