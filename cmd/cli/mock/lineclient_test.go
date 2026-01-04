package mock_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"yuruppu/cmd/cli/mock"
	lineclient "yuruppu/internal/line/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLineClient tests the constructor
func TestNewLineClient(t *testing.T) {
	t.Run("should create client with valid writer", func(t *testing.T) {
		// Given
		var buf bytes.Buffer

		// When
		client := mock.NewLineClient(&buf)

		// Then
		require.NotNil(t, client)
	})

	t.Run("should panic when writer is nil", func(t *testing.T) {
		// Given/When/Then
		// AC-004: Constructor requires a non-nil writer
		assert.Panics(t, func() {
			mock.NewLineClient(nil)
		})
	})
}

// TestLineClient_GetMessageContent tests the GetMessageContent method from bot.LineClient interface
func TestLineClient_GetMessageContent(t *testing.T) {
	// AC-001: Test GetMessageContent returns an error indicating media is not supported
	// FR-003: LINE API is mocked - media operations should not be supported

	t.Run("should return error indicating media is not supported", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		messageID := "msg123456"

		// When
		data, mimeType, err := client.GetMessageContent(messageID)

		// Then
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Empty(t, mimeType)
		assert.Contains(t, err.Error(), "media")
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("should return error for any message ID", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		tests := []struct {
			name      string
			messageID string
		}{
			{
				name:      "short ID",
				messageID: "123",
			},
			{
				name:      "long ID",
				messageID: "very-long-message-id-12345678901234567890",
			},
			{
				name:      "empty ID",
				messageID: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When
				data, mimeType, err := client.GetMessageContent(tt.messageID)

				// Then
				require.Error(t, err)
				assert.Nil(t, data)
				assert.Empty(t, mimeType)
			})
		}
	})
}

// TestLineClient_GetProfile tests the GetProfile method from bot.LineClient interface
func TestLineClient_GetProfile(t *testing.T) {
	// AC-002: Test GetProfile returns an error indicating profile should be created via CLI
	// Design: GetProfile returns error (profile created via CLI prompts)

	t.Run("should return error indicating profile should be created via CLI", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		ctx := context.Background()
		userID := "user123"

		// When
		profile, err := client.GetProfile(ctx, userID)

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "profile")
		assert.Contains(t, err.Error(), "CLI")
	})

	t.Run("should return error for any user ID", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		ctx := context.Background()

		tests := []struct {
			name   string
			userID string
		}{
			{
				name:   "alphanumeric user ID",
				userID: "user123",
			},
			{
				name:   "underscore user ID",
				userID: "test_user",
			},
			{
				name:   "empty user ID",
				userID: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When
				profile, err := client.GetProfile(ctx, tt.userID)

				// Then
				require.Error(t, err)
				assert.Nil(t, profile)
			})
		}
	})

	t.Run("should respect context cancellation", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When
		profile, err := client.GetProfile(ctx, "user123")

		// Then
		// Should still return error (context may or may not be checked)
		require.Error(t, err)
		assert.Nil(t, profile)
	})
}

// TestLineClient_SendReply tests the SendReply method from reply.LineClient interface
func TestLineClient_SendReply(t *testing.T) {
	// AC-003: Test SendReply writes the message to the configured stdout writer
	// FR-003: LINE API is mocked - replies are printed to stdout instead of sent via API

	t.Run("should write message to stdout", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := "Hello, user!"

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, message)
	})

	t.Run("should handle multiline messages", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := "Line 1\nLine 2\nLine 3"

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Line 1")
		assert.Contains(t, output, "Line 2")
		assert.Contains(t, output, "Line 3")
	})

	t.Run("should handle empty messages", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := ""

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		// Output should exist even if empty
	})

	t.Run("should handle Japanese characters", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := "こんにちは、世界！"

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, message)
	})

	t.Run("should handle special characters", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := "Special: !@#$%^&*()_+-=[]{}|;':\",./<>?"

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, message)
	})

	t.Run("should support multiple calls", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		err1 := client.SendReply("token1", "First message")
		err2 := client.SendReply("token2", "Second message")
		err3 := client.SendReply("token3", "Third message")

		// Then
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
		output := buf.String()
		assert.Contains(t, output, "First message")
		assert.Contains(t, output, "Second message")
		assert.Contains(t, output, "Third message")
	})

	t.Run("should format output with clear message boundaries", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		replyToken := "token123"
		message := "Test message"

		// When
		err := client.SendReply(replyToken, message)

		// Then
		require.NoError(t, err)
		output := buf.String()
		// Output should have some formatting (newline at minimum)
		// to distinguish messages in the terminal
		assert.True(t, len(output) > len(message), "output should include formatting")
	})

	t.Run("should ignore reply token", func(t *testing.T) {
		// Given
		var buf1 bytes.Buffer
		var buf2 bytes.Buffer
		client1 := mock.NewLineClient(&buf1)
		client2 := mock.NewLineClient(&buf2)
		message := "Same message"

		// When
		err1 := client1.SendReply("token1", message)
		err2 := client2.SendReply("token2", message)

		// Then
		require.NoError(t, err1)
		require.NoError(t, err2)
		// Both should produce similar output (token doesn't affect output)
		assert.True(t, strings.Contains(buf1.String(), message))
		assert.True(t, strings.Contains(buf2.String(), message))
	})
}

// TestLineClient_InterfaceCompliance verifies that LineClient implements required interfaces
func TestLineClient_InterfaceCompliance(t *testing.T) {
	t.Run("should implement bot.LineClient interface", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When/Then
		// This will fail to compile if the interface is not implemented
		var _ interface {
			GetMessageContent(messageID string) (data []byte, mimeType string, err error)
			GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
		} = client
	})

	t.Run("should implement reply.LineClient interface", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When/Then
		// This will fail to compile if the interface is not implemented
		var _ interface {
			SendReply(replyToken string, text string) error
		} = client
	})
}
