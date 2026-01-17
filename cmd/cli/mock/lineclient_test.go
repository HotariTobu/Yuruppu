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

// mockFetcher is a test implementation of Fetcher.
type mockFetcher struct {
	userProfile  *lineclient.UserProfile
	userErr      error
	groupSummary *lineclient.GroupSummary
	groupErr     error
}

func (m *mockFetcher) FetchUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	return m.userProfile, m.userErr
}

func (m *mockFetcher) FetchGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error) {
	return m.groupSummary, m.groupErr
}

// TestNewLineClient tests the constructor
func TestNewLineClient(t *testing.T) {
	t.Run("should create client with fetcher", func(t *testing.T) {
		// Given
		fetcher := &mockFetcher{}

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
		client := mock.NewLineClient(&mockFetcher{})

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

// TestLineClient_GetUserProfile tests the GetUserProfile method
func TestLineClient_GetUserProfile(t *testing.T) {
	t.Run("should delegate to fetcher", func(t *testing.T) {
		// Given
		expectedProfile := &lineclient.UserProfile{
			DisplayName:   "Test User",
			PictureURL:    "https://example.com/pic.jpg",
			StatusMessage: "Hello",
		}
		client := mock.NewLineClient(&mockFetcher{userProfile: expectedProfile})

		// When
		profile, err := client.GetUserProfile(context.Background(), "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedProfile, profile)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("fetch failed")
		client := mock.NewLineClient(&mockFetcher{userErr: expectedErr})

		// When
		profile, err := client.GetUserProfile(context.Background(), "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Equal(t, expectedErr, err)
	})
}

// TestLineClient_GetGroupSummary tests the GetGroupSummary method
func TestLineClient_GetGroupSummary(t *testing.T) {
	t.Run("should delegate to fetcher", func(t *testing.T) {
		// Given
		expectedSummary := &lineclient.GroupSummary{
			GroupID:    "group123",
			GroupName:  "Test Group",
			PictureURL: "https://example.com/group.jpg",
		}
		client := mock.NewLineClient(&mockFetcher{groupSummary: expectedSummary})

		// When
		summary, err := client.GetGroupSummary(context.Background(), "group123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedSummary, summary)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("fetch failed")
		client := mock.NewLineClient(&mockFetcher{groupErr: expectedErr})

		// When
		summary, err := client.GetGroupSummary(context.Background(), "group123")

		// Then
		require.Error(t, err)
		assert.Nil(t, summary)
		assert.Equal(t, expectedErr, err)
	})
}

// TestLineClient_SendReply tests the SendReply method
func TestLineClient_SendReply(t *testing.T) {
	t.Run("should return nil (no-op)", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{})

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
		client := mock.NewLineClient(&mockFetcher{})

		// When/Then
		var _ interface {
			GetMessageContent(messageID string) (data []byte, mimeType string, err error)
			GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
			GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
		} = client
	})

	t.Run("should implement reply.LineClient interface", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{})

		// When/Then
		var _ interface {
			SendReply(replyToken string, text string) error
		} = client
	})
}
