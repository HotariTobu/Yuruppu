package bot_test

import (
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleImage Tests
// =============================================================================

func TestHandleImage(t *testing.T) {
	t.Run("success - downloads and stores image", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("fake-image-data"),
			mimeType: "image/jpeg",
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "Nice image!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "msg-456", mockClient.lastMessageID)
		assert.Equal(t, "user-123", mockMedia.lastSourceID)
		assert.Equal(t, []byte("fake-image-data"), mockMedia.lastData)
		assert.Equal(t, "image/jpeg", mockMedia.lastMIMEType)
	})

	t.Run("success - storage key format is sourceID/uuid", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("image-data"),
			mimeType: "image/png",
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "Nice!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "group-789", mockMedia.lastSourceID)
	})

	t.Run("fallback - download error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			err: errors.New("LINE API failed"),
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})

	t.Run("fallback - storage error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("image-data"),
			mimeType: "image/jpeg",
		}
		mockMedia := &mockMediaService{storeErr: errors.New("GCS failed")}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})
}
