//go:build integration

package line_test

import (
	"os"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireLINECredentials fails the test if required LINE credentials are not available.
// AC-001 (fix-integration-test-issues): Fail instead of skip when credentials are missing.
func requireLINECredentials(t *testing.T) (channelSecret, channelAccessToken string) {
	t.Helper()
	channelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	if channelSecret == "" {
		t.Fatal("LINE_CHANNEL_SECRET environment variable is not set")
	}
	channelAccessToken = os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelAccessToken == "" {
		t.Fatal("LINE_CHANNEL_ACCESS_TOKEN environment variable is not set")
	}
	return channelSecret, channelAccessToken
}

// TestLINE_Integration_GetBotInfo tests that GetBotInfo returns bot information from LINE API.
func TestLINE_Integration_GetBotInfo(t *testing.T) {
	_, channelAccessToken := requireLINECredentials(t)

	api, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	require.NoError(t, err, "NewMessagingApiAPI should succeed with valid token")

	botInfo, err := api.GetBotInfo()

	require.NoError(t, err, "GetBotInfo should succeed")
	assert.NotNil(t, botInfo, "botInfo should not be nil")
	assert.NotEmpty(t, botInfo.UserId, "bot user ID should not be empty")
	assert.NotEmpty(t, botInfo.DisplayName, "bot display name should not be empty")
}
