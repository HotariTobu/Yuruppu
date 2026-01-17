package bot_test

import (
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	lineclient "yuruppu/internal/line/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleFollow Tests
// =============================================================================

func TestHandler_HandleFollow(t *testing.T) {
	t.Run("fetches profile from LINE and stores it", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profile: &lineclient.UserProfile{
				DisplayName:   "Alice",
				PictureURL:    "",
				StatusMessage: "Hello!",
			},
		}
		mockPS := &mockProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, mockPS, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.NoError(t, err)
		assert.Equal(t, "user-123", mockPS.lastUserID)
		require.NotNil(t, mockPS.profile)
		assert.Equal(t, "Alice", mockPS.profile.DisplayName)
		assert.Equal(t, "Hello!", mockPS.profile.StatusMessage)
	})

	t.Run("returns error when userID not in context", func(t *testing.T) {
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := t.Context() // No userID in context
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID not found")
	})

	t.Run("returns error when GetProfile fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profileErr: errors.New("LINE API error"),
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch profile")
	})

	t.Run("returns error when SetUserProfile fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profile: &lineclient.UserProfile{DisplayName: "Alice"},
		}
		mockPS := &mockProfileService{
			setErr: errors.New("storage error"),
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, mockPS, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store profile")
	})
}
