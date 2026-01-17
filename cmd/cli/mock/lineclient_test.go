package mock_test

import (
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
	t.Run("should create client with fetcher", func(t *testing.T) {
		// Given
		fetcher := &mockProfileFetcher{}

		// When
		client := mock.NewLineClient(fetcher)

		// Then
		require.NotNil(t, client)
	})

	t.Run("should panic when fetcher is nil", func(t *testing.T) {
		// When/Then
		assert.Panics(t, func() {
			mock.NewLineClient(nil)
		})
	})
}

// TestLineClient_GetMessageContent tests the GetMessageContent method
func TestLineClient_GetMessageContent(t *testing.T) {
	t.Run("should return error indicating media is not supported", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockProfileFetcher{})

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
	t.Run("should delegate to fetcher", func(t *testing.T) {
		// Given
		expectedProfile := &lineclient.UserProfile{
			DisplayName:   "Test User",
			PictureURL:    "https://example.com/pic.jpg",
			StatusMessage: "Hello",
		}
		client := mock.NewLineClient(&mockProfileFetcher{profile: expectedProfile})

		// When
		profile, err := client.GetProfile(context.Background(), "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedProfile, profile)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("fetch failed")
		client := mock.NewLineClient(&mockProfileFetcher{err: expectedErr})

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
	t.Run("should return nil (no-op)", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockProfileFetcher{})

		// When
		err := client.SendReply("token123", "Hello, user!")

		// Then
		require.NoError(t, err)
	})
}

// TestLineClient_InterfaceCompliance verifies that LineClient implements required interfaces
func TestLineClient_InterfaceCompliance(t *testing.T) {
	t.Run("should implement bot.LineClient interface", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockProfileFetcher{})

		// When/Then
		var _ interface {
			GetMessageContent(messageID string) (data []byte, mimeType string, err error)
			GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
		} = client
	})

	t.Run("should implement reply.LineClient interface", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockProfileFetcher{})

		// When/Then
		var _ interface {
			SendReply(replyToken string, text string) error
		} = client
	})
}
