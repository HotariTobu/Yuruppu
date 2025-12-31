//go:build integration

package line_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yuruppu/internal/line"
)

// requireMediaTestCredentials fails the test if required credentials and test data are not available.
func requireMediaTestCredentials(t *testing.T) (channelAccessToken, testMessageID string) {
	t.Helper()
	channelAccessToken = os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelAccessToken == "" {
		t.Fatal("LINE_CHANNEL_ACCESS_TOKEN environment variable is not set")
	}
	testMessageID = os.Getenv("TEST_MESSAGE_ID")
	if testMessageID == "" {
		t.Fatal("TEST_MESSAGE_ID environment variable is not set (must be a valid media message ID)")
	}
	return channelAccessToken, testMessageID
}

// TestGetMessageContent_Integration_ValidMediaMessage tests downloading media content from LINE API.
// AC-001: Successfully download media content
// Given: A valid message ID for media sent by a user
// When: Content is downloaded using the message ID
// Then: Binary data is obtained AND MIME type is obtained
func TestGetMessageContent_Integration_ValidMediaMessage(t *testing.T) {
	// Given: A valid message ID for media sent by a user
	channelAccessToken, testMessageID := requireMediaTestCredentials(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client, err := line.NewClient(channelAccessToken, logger)
	require.NoError(t, err, "NewClient should succeed with valid token")

	// When: Content is downloaded using the message ID
	mediaContent, err := client.GetMessageContent(testMessageID)

	// Then: Binary data is obtained AND MIME type is obtained
	require.NoError(t, err, "GetMessageContent should succeed with valid message ID")
	require.NotNil(t, mediaContent, "MediaContent should not be nil")

	// FR-002: Obtain both the binary content and the MIME type
	assert.NotEmpty(t, mediaContent.Data, "Data should not be empty")
	assert.NotEmpty(t, mediaContent.MIMEType, "MIMEType should not be empty")

	// Verify MIME type is valid media type
	validPrefixes := []string{"image/", "video/", "audio/", "application/"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(mediaContent.MIMEType, prefix) {
			hasValidPrefix = true
			break
		}
	}
	assert.True(t, hasValidPrefix, "MIMEType should start with valid media type prefix (image/, video/, audio/, application/), got: %s", mediaContent.MIMEType)

	t.Logf("Downloaded media content: %d bytes, MIME type: %s", len(mediaContent.Data), mediaContent.MIMEType)
}

// TestGetMessageContent_Integration_InvalidMessageID tests error handling for invalid message IDs.
func TestGetMessageContent_Integration_InvalidMessageID(t *testing.T) {
	channelAccessToken, _ := requireMediaTestCredentials(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client, err := line.NewClient(channelAccessToken, logger)
	require.NoError(t, err, "NewClient should succeed with valid token")

	// Test with obviously invalid message ID
	invalidMessageID := "invalid-message-id-12345"

	mediaContent, err := client.GetMessageContent(invalidMessageID)

	// Should return error for invalid message ID
	assert.Error(t, err, "GetMessageContent should fail with invalid message ID")
	assert.Nil(t, mediaContent, "MediaContent should be nil on error")
}

// TestGetMessageContent_Integration_EmptyMessageID tests error handling for empty message IDs.
func TestGetMessageContent_Integration_EmptyMessageID(t *testing.T) {
	channelAccessToken, _ := requireMediaTestCredentials(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client, err := line.NewClient(channelAccessToken, logger)
	require.NoError(t, err, "NewClient should succeed with valid token")

	mediaContent, err := client.GetMessageContent("")

	// Should return error for empty message ID
	assert.Error(t, err, "GetMessageContent should fail with empty message ID")
	assert.Nil(t, mediaContent, "MediaContent should be nil on error")
}
