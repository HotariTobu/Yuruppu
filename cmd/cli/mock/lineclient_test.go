package mock_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"yuruppu/cmd/cli/mock"
	lineclient "yuruppu/internal/line/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProfileFetcher is a test implementation of ProfileFetcher.
type mockProfileFetcher struct {
	profile *lineclient.UserProfile
	err     error
}

func (m *mockProfileFetcher) FetchProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	return m.profile, m.err
}

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
		// When/Then
		assert.Panics(t, func() {
			mock.NewLineClient(nil)
		})
	})
}

// TestLineClient_RegisterProfileFetcher tests the RegisterProfileFetcher method
func TestLineClient_RegisterProfileFetcher(t *testing.T) {
	t.Run("should register profile fetcher", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		fetcher := &mockProfileFetcher{
			profile: &lineclient.UserProfile{DisplayName: "Test User"},
		}

		// When
		client.RegisterProfileFetcher(fetcher)

		// Then
		profile, err := client.GetProfile(context.Background(), "user123")
		require.NoError(t, err)
		assert.Equal(t, "Test User", profile.DisplayName)
	})
}

// TestLineClient_GetMessageContent tests the GetMessageContent method
func TestLineClient_GetMessageContent(t *testing.T) {
	t.Run("should return error indicating media is not supported", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		data, mimeType, err := client.GetMessageContent("msg123")

		// Then
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Empty(t, mimeType)
		assert.Contains(t, err.Error(), "media")
		assert.Contains(t, err.Error(), "not supported")
	})
}

// TestLineClient_GetProfile tests the GetProfile method
func TestLineClient_GetProfile(t *testing.T) {
	t.Run("should return error when no fetcher registered", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		profile, err := client.GetProfile(context.Background(), "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("should delegate to registered fetcher", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		expectedProfile := &lineclient.UserProfile{
			DisplayName:   "Test User",
			PictureURL:    "https://example.com/pic.jpg",
			StatusMessage: "Hello",
		}
		client.RegisterProfileFetcher(&mockProfileFetcher{profile: expectedProfile})

		// When
		profile, err := client.GetProfile(context.Background(), "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedProfile, profile)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)
		expectedErr := errors.New("fetch failed")
		client.RegisterProfileFetcher(&mockProfileFetcher{err: expectedErr})

		// When
		profile, err := client.GetProfile(context.Background(), "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Equal(t, expectedErr, err)
	})
}

// TestLineClient_SendReply tests the SendReply method
func TestLineClient_SendReply(t *testing.T) {
	t.Run("should write message to stdout", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		err := client.SendReply("token123", "Hello, user!")

		// Then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Hello, user!")
	})

	t.Run("should handle multiline messages", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		err := client.SendReply("token123", "Line 1\nLine 2\nLine 3")

		// Then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Line 1")
		assert.Contains(t, output, "Line 2")
		assert.Contains(t, output, "Line 3")
	})

	t.Run("should handle Japanese characters", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When
		err := client.SendReply("token123", "こんにちは、世界！")

		// Then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "こんにちは、世界！")
	})
}

// TestLineClient_InterfaceCompliance verifies that LineClient implements required interfaces
func TestLineClient_InterfaceCompliance(t *testing.T) {
	t.Run("should implement bot.LineClient interface", func(t *testing.T) {
		// Given
		var buf bytes.Buffer
		client := mock.NewLineClient(&buf)

		// When/Then
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
		var _ interface {
			SendReply(replyToken string, text string) error
		} = client
	})
}
