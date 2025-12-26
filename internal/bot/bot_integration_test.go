//go:build integration

package bot_test

import (
	"os"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfMissingCredentials skips the test if required LINE credentials are not available.
// AC-003: Integration tests skip without credentials with descriptive message.
func skipIfMissingCredentials(t *testing.T) {
	t.Helper()
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelAccessToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelSecret == "" {
		t.Skip("Skipping integration test: LINE_CHANNEL_SECRET environment variable is not set")
	}
	if channelAccessToken == "" {
		t.Skip("Skipping integration test: LINE_CHANNEL_ACCESS_TOKEN environment variable is not set")
	}
}

// TestBot_Integration_GetBotInfo tests that GetBotInfo returns bot information from LINE API.
// AC-004: GetBotInfo() returns bot information from LINE API.
func TestBot_Integration_GetBotInfo(t *testing.T) {
	skipIfMissingCredentials(t)

	channelAccessToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")

	client, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	require.NoError(t, err, "NewMessagingApiAPI should succeed with valid credentials")

	botInfo, err := client.GetBotInfo()

	require.NoError(t, err, "GetBotInfo should succeed")
	assert.NotNil(t, botInfo, "botInfo should not be nil")
	assert.NotEmpty(t, botInfo.DisplayName, "bot should have a display name")
}
