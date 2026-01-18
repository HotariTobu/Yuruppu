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

// mockGroupSim is a test implementation of GroupSim.
type mockGroupSim struct {
	members []string
	err     error
}

func (m *mockGroupSim) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	return m.members, m.err
}

// TestNewLineClient tests the constructor
func TestNewLineClient(t *testing.T) {
	t.Run("should create client with fetcher and groupSim", func(t *testing.T) {
		// Given
		fetcher := &mockFetcher{}
		groupSim := &mockGroupSim{}

		// When
		client := mock.NewLineClient(fetcher, groupSim)

		// Then
		require.NotNil(t, client)
	})

	t.Run("should panic when fetcher is nil", func(t *testing.T) {
		// When/Then
		assert.Panics(t, func() {
			mock.NewLineClient(nil, &mockGroupSim{})
		})
	})

	t.Run("should panic when groupSim is nil", func(t *testing.T) {
		// When/Then
		assert.Panics(t, func() {
			mock.NewLineClient(&mockFetcher{}, nil)
		})
	})
}

// TestLineClient_GetMessageContent tests the GetMessageContent method
func TestLineClient_GetMessageContent(t *testing.T) {
	t.Run("should return error indicating media is not supported", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{}, &mockGroupSim{})

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
		client := mock.NewLineClient(&mockFetcher{userProfile: expectedProfile}, &mockGroupSim{})

		// When
		profile, err := client.GetUserProfile(context.Background(), "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedProfile, profile)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("fetch failed")
		client := mock.NewLineClient(&mockFetcher{userErr: expectedErr}, &mockGroupSim{})

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
		client := mock.NewLineClient(&mockFetcher{groupSummary: expectedSummary}, &mockGroupSim{})

		// When
		summary, err := client.GetGroupSummary(context.Background(), "group123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, expectedSummary, summary)
	})

	t.Run("should propagate fetcher error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("fetch failed")
		client := mock.NewLineClient(&mockFetcher{groupErr: expectedErr}, &mockGroupSim{})

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
		client := mock.NewLineClient(&mockFetcher{}, &mockGroupSim{})

		// When
		err := client.SendReply("token123", "Hello, user!")

		// Then
		require.NoError(t, err)
	})
}

// TestLineClient_GetGroupMemberCount tests the GetGroupMemberCount method
func TestLineClient_GetGroupMemberCount(t *testing.T) {
	t.Run("should return member count via groupSim", func(t *testing.T) {
		// Given
		groupSim := &mockGroupSim{members: []string{"user1", "user2", "user3"}}
		client := mock.NewLineClient(&mockFetcher{}, groupSim)

		// When
		count, err := client.GetGroupMemberCount(context.Background(), "group123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("should return 0 when group has no members", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{}, &mockGroupSim{members: []string{}})

		// When
		count, err := client.GetGroupMemberCount(context.Background(), "group123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("should propagate groupSim error", func(t *testing.T) {
		// Given
		expectedErr := errors.New("group not found")
		groupSim := &mockGroupSim{err: expectedErr}
		client := mock.NewLineClient(&mockFetcher{}, groupSim)

		// When
		count, err := client.GetGroupMemberCount(context.Background(), "group123")

		// Then
		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, expectedErr, err)
	})
}

// TestLineClient_InterfaceCompliance verifies that LineClient implements required interfaces
func TestLineClient_InterfaceCompliance(t *testing.T) {
	t.Run("should implement bot.LineClient interface", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{}, &mockGroupSim{})

		// When/Then
		var _ interface {
			GetMessageContent(messageID string) (data []byte, mimeType string, err error)
			GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
			GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
		} = client
	})

	t.Run("should implement reply.LineClient interface", func(t *testing.T) {
		// Given
		client := mock.NewLineClient(&mockFetcher{}, &mockGroupSim{})

		// When/Then
		var _ interface {
			SendReply(replyToken string, text string) error
		} = client
	})
}
